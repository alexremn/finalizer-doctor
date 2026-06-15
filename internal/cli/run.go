package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/apply"
	"github.com/alexremn/finalizer-doctor/internal/cluster"
	"github.com/alexremn/finalizer-doctor/internal/discover"
	"github.com/alexremn/finalizer-doctor/internal/mapping"
	"github.com/alexremn/finalizer-doctor/internal/model"
	"github.com/alexremn/finalizer-doctor/internal/orphan"
	"github.com/alexremn/finalizer-doctor/internal/plan"
	"github.com/alexremn/finalizer-doctor/internal/probe"
	"github.com/alexremn/finalizer-doctor/internal/render"
	"github.com/alexremn/finalizer-doctor/internal/snapshot"
	"github.com/alexremn/finalizer-doctor/internal/verdict"
	"github.com/alexremn/finalizer-doctor/internal/webhook"
)

// InvalidInvocation marks a usage error → exit code 1 (design §11).
type InvalidInvocation struct{ Msg string }

func (e *InvalidInvocation) Error() string { return e.Msg }

// preflight checks the RBAC verbs the run needs via SelfSubjectAccessReview and
// returns a clear error on the first denial. Reads degrade gracefully elsewhere,
// so it checks `get` on the target always and the mutating verb when --apply.
func preflight(ctx context.Context, c cluster.Client, ref model.ResourceRef, apply bool) error {
	get := authv1.ResourceAttributes{Group: ref.GVR.Group, Resource: ref.GVR.Resource, Namespace: ref.Namespace, Name: ref.Name, Verb: "get"}
	if err := mustAllow(ctx, c, get, "get", resourceDesc(ref)); err != nil {
		return err
	}
	if !apply {
		return nil
	}
	if ref.GVR.Resource == "namespaces" {
		fin := authv1.ResourceAttributes{Resource: "namespaces", Subresource: "finalize", Verb: "update", Name: ref.Name}
		return mustAllow(ctx, c, fin, "update", "namespaces/finalize")
	}
	patch := authv1.ResourceAttributes{Group: ref.GVR.Group, Resource: ref.GVR.Resource, Namespace: ref.Namespace, Name: ref.Name, Verb: "patch"}
	return mustAllow(ctx, c, patch, "patch", resourceDesc(ref))
}

func mustAllow(ctx context.Context, c cluster.Client, attrs authv1.ResourceAttributes, verb, desc string) error {
	ok, err := c.Can(ctx, attrs)
	if err != nil {
		return fmt.Errorf("RBAC pre-flight failed: %w", err)
	}
	if !ok {
		return fmt.Errorf("permission denied: you need %q on %s", verb, desc)
	}
	return nil
}

func resourceDesc(ref model.ResourceRef) string {
	if ref.GVR.Group != "" {
		return ref.GVR.Resource + "." + ref.GVR.Group
	}
	return ref.GVR.Resource
}

// appendAudit appends one tab-separated line per audit record to path.
func appendAudit(path string, ref model.ResourceRef, records []string) error {
	if len(records) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	ts := time.Now().UTC().Format(time.RFC3339)
	for _, r := range records {
		if _, err := fmt.Fprintf(f, "%s\t%s\t%s\n", ts, ref.String(), r); err != nil {
			return err
		}
	}
	return nil
}

// Options holds parsed CLI flags.
type Options struct {
	Target      string
	Namespace   string
	All         bool
	Apply       bool
	Confirm     string
	Interactive bool   // TTY; root.go sets this
	TypedName   string // typed-name prompt input; root.go captures it
	Verdict     string // "strict" | "score"
	Output      string // "human" | "json"
	AuditFile   string // optional path to append audit records to
	Now         time.Time
}

// targetResult is one stuck object with the verdicts for all its finalizers.
type targetResult struct {
	obj      model.StuckObject
	verdicts []model.Verdict
}

// Run executes the read pipeline (and, when Apply, the gated mutation). Returns
// rendered output, a process exit code, and an error for operational failures.
func Run(ctx context.Context, c cluster.Client, o Options) (string, int, error) {
	if o.Apply && o.All {
		return "", 1, &InvalidInvocation{Msg: "--apply cannot be combined with --all"}
	}
	now := o.Now
	if now.IsZero() {
		now = time.Now()
	}

	refs, err := resolveTargets(ctx, c, o)
	if err != nil {
		return "", 1, err
	}
	if len(refs) == 0 {
		return "no stuck resources found\n", 0, nil
	}

	// RBAC pre-flight for a single target: fail fast with a clear message rather
	// than a confusing mid-run 403. (--all reads degrade gracefully per source.)
	if !o.All {
		if err := preflight(ctx, c, refs[0], o.Apply); err != nil {
			return "", 1, err
		}
	}

	snap, results, err := diagnose(ctx, c, refs, now, strategyFor(o))
	if err != nil {
		return "", 1, err
	}

	if !o.Apply {
		return renderDryRun(o, snap, results), 2, nil
	}
	return runApply(ctx, c, o, snap, results, now)
}

func strategyFor(o Options) verdict.Verdicter {
	if o.Verdict == "score" {
		return verdict.Score{}
	}
	return verdict.Strict{}
}

// diagnose builds a snapshot for the given refs and verdicts every finalizer.
func diagnose(ctx context.Context, c cluster.Client, refs []model.ResourceRef, now time.Time, strat verdict.Verdicter) (model.Snapshot, []targetResult, error) {
	snap, err := snapshot.Build(ctx, c, refs, now)
	if err != nil {
		return model.Snapshot{}, nil, fmt.Errorf("snapshot: %w", err)
	}
	var out []targetResult
	for _, obj := range snap.Targets {
		tr := targetResult{obj: obj}
		isNamespace := obj.Ref.GVR.Resource == "namespaces"
		for _, fin := range obj.AllFinalizers() {
			if isNamespace && fin == verdict.NSKubernetesFinalizer {
				// The namespace `kubernetes` spec finalizer is attributed to a
				// failing dependency, not the (never-dead) namespace controller.
				tr.verdicts = append(tr.verdicts, verdict.NamespaceKubernetes(obj, snap))
				continue
			}
			owner := mapping.Map(fin, snap)
			ev := probe.For(owner, snap)
			tr.verdicts = append(tr.verdicts, strat.Verdict(owner, ev))
		}
		out = append(out, tr)
	}
	return snap, out, nil
}

// orphanScanNamespace is the namespace to scan for orphans: a namespace target's
// own name, otherwise the target's namespace.
func orphanScanNamespace(obj model.StuckObject) string {
	if obj.Ref.GVR.Resource == "namespaces" {
		return obj.Ref.Name
	}
	return obj.Ref.Namespace
}

func detectOrphans(ctx context.Context, c cluster.Client, obj model.StuckObject) []model.ResourceRef {
	ns := orphanScanNamespace(obj)
	if ns == "" {
		return nil
	}
	candidates, err := discover.NamespaceObjects(ctx, c, ns)
	if err != nil {
		return nil // best-effort: a failed candidate scan never blocks the clear
	}
	return orphan.Detect(obj, candidates)
}

func allVerdicts(results []targetResult) []model.Verdict {
	var vs []model.Verdict
	for _, r := range results {
		vs = append(vs, r.verdicts...)
	}
	return vs
}

// combinedState is DEAD only when every blocking finalizer is DEAD.
func combinedState(verdicts []model.Verdict) model.State {
	if len(verdicts) == 0 {
		return model.StateUnknown
	}
	for _, v := range verdicts {
		if v.State != model.StateDead {
			if v.State == model.StateSlow {
				return model.StateSlow
			}
			return model.StateUnknown
		}
	}
	return model.StateDead
}

func renderOut(o Options, verdicts []model.Verdict, p model.Plan) string {
	if o.Output == "json" {
		return render.JSON(verdicts, p)
	}
	return render.Human(verdicts, p)
}

func renderDryRun(o Options, snap model.Snapshot, results []targetResult) string {
	out := renderOut(o, allVerdicts(results), model.Plan{})
	if o.Output == "json" {
		return out
	}
	for _, r := range results {
		if combinedState(r.verdicts) != model.StateDead {
			continue
		}
		if blocked, note := webhook.Blocks(snap, r.obj.Ref); blocked {
			out += "blocked: " + note + " — remove/fix the webhook, then re-run\n"
			continue
		}
		d := apply.Digest(r.obj.Ref, r.verdicts, r.obj.ResourceVersion)
		out += fmt.Sprintf("to apply: kubectl finalizer-doctor %s --apply --confirm=%s\n", r.obj.Ref.String(), d)
	}
	return out
}

func runApply(ctx context.Context, c cluster.Client, o Options, snap model.Snapshot, results []targetResult, now time.Time) (string, int, error) {
	var out strings.Builder
	for _, r := range results {
		if combinedState(r.verdicts) != model.StateDead {
			fmt.Fprintf(&out, "refused: %s is not all-DEAD; investigate (dry-run for evidence)\n", r.obj.Ref)
			return out.String(), 3, nil
		}
		if blocked, note := webhook.Blocks(snap, r.obj.Ref); blocked {
			fmt.Fprintf(&out, "refused: %s\n", note)
			return out.String(), 3, nil
		}
		gate := apply.Gate{Interactive: o.Interactive, Confirm: o.Confirm}
		if o.Interactive {
			gate = gate.WithTypedName(o.TypedName)
		}
		if err := gate.Authorize(r.obj.Ref, r.verdicts, r.obj.ResourceVersion); err != nil {
			fmt.Fprintf(&out, "refused: %v\n", err)
			return out.String(), 3, nil
		}
		orphans := detectOrphans(ctx, c, r.obj)
		p := plan.Build(r.obj, r.verdicts, orphans, false)
		target := r.obj.Ref
		// Re-verify gates on the PARENT target's DEAD state (an orphan is not a
		// finalizer-bearing object), but pins the action target's OWN
		// resourceVersion so each mutation has a correct precondition.
		reverify := func(actionRef model.ResourceRef) (model.State, string, error) {
			_, rr, err := diagnose(ctx, c, []model.ResourceRef{target}, now, strategyFor(o))
			if err != nil || len(rr) == 0 {
				return model.StateUnknown, "", err
			}
			if st := combinedState(rr[0].verdicts); st != model.StateDead {
				return st, "", nil
			}
			cur, err := c.Get(ctx, actionRef.GVR, actionRef.Namespace, actionRef.Name)
			if err != nil {
				return model.StateUnknown, "", err
			}
			return model.StateDead, cur.GetResourceVersion(), nil
		}
		res, err := apply.Execute(ctx, c, p, reverify)
		for _, a := range res.Audit {
			fmt.Fprintf(&out, "applied: %s\n", a)
		}
		if o.AuditFile != "" {
			if werr := appendAudit(o.AuditFile, r.obj.Ref, res.Audit); werr != nil {
				fmt.Fprintf(&out, "warning: could not write audit file %q: %v\n", o.AuditFile, werr)
			}
		}
		if err != nil {
			fmt.Fprintf(&out, "aborted after %d action(s): %v\n", res.Completed, err)
			return out.String(), 3, nil
		}
	}
	return out.String(), 0, nil
}

func resolveTargets(ctx context.Context, c cluster.Client, o Options) ([]model.ResourceRef, error) {
	if o.All {
		return discover.Scan(ctx, c)
	}
	t, err := discover.ParseTarget(o.Target, o.Namespace)
	if err != nil {
		return nil, err
	}
	ref, err := resolveGVR(ctx, c, t)
	if err != nil {
		return nil, err
	}
	return []model.ResourceRef{ref}, nil
}

// resolveGVR turns a parsed target into a concrete GVR using discovery, falling
// back to a v1 guess when discovery does not list the resource.
func resolveGVR(ctx context.Context, c cluster.Client, t discover.Target) (model.ResourceRef, error) {
	lists, err := c.ServerPreferredResources(ctx)
	if err == nil {
		for _, rl := range lists {
			gv, e := schema.ParseGroupVersion(rl.GroupVersion)
			if e != nil {
				continue
			}
			if t.Group != "" && gv.Group != t.Group {
				continue
			}
			for _, r := range rl.APIResources {
				if strings.Contains(r.Name, "/") {
					continue
				}
				if matchesResource(r, t.Resource) {
					return model.ResourceRef{GVR: gv.WithResource(r.Name), Namespace: t.Namespace, Name: t.Name}, nil
				}
			}
		}
	}
	return model.ResourceRef{
		GVR:       schema.GroupVersionResource{Group: t.Group, Version: "v1", Resource: t.Resource},
		Namespace: t.Namespace, Name: t.Name,
	}, nil
}

func matchesResource(r metav1.APIResource, want string) bool {
	if r.Name == want || r.SingularName == want || strings.EqualFold(r.Kind, want) {
		return true
	}
	for _, s := range r.ShortNames {
		if s == want {
			return true
		}
	}
	return false
}
