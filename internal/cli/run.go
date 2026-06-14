package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/apply"
	"github.com/alexremn/finalizer-doctor/internal/cluster"
	"github.com/alexremn/finalizer-doctor/internal/discover"
	"github.com/alexremn/finalizer-doctor/internal/mapping"
	"github.com/alexremn/finalizer-doctor/internal/model"
	"github.com/alexremn/finalizer-doctor/internal/plan"
	"github.com/alexremn/finalizer-doctor/internal/probe"
	"github.com/alexremn/finalizer-doctor/internal/render"
	"github.com/alexremn/finalizer-doctor/internal/snapshot"
	"github.com/alexremn/finalizer-doctor/internal/verdict"
)

// InvalidInvocation marks a usage error → exit code 1 (design §11).
type InvalidInvocation struct{ Msg string }

func (e *InvalidInvocation) Error() string { return e.Msg }

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
	Now         time.Time
}

// targetResult is one stuck object with the verdicts for all its finalizers.
type targetResult struct {
	obj      model.StuckObject
	verdicts []model.Verdict
}

// Run executes the read pipeline (and, when Apply, the gated mutation). Returns
// rendered output, a process exit code, and an error for operational failures.
func Run(ctx context.Context, c cluster.ClusterClient, o Options) (string, int, error) {
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

	results, err := diagnose(ctx, c, refs, now, strategyFor(o))
	if err != nil {
		return "", 1, err
	}

	if !o.Apply {
		return renderDryRun(o, results), 2, nil
	}
	return runApply(ctx, c, o, results, now)
}

func strategyFor(o Options) verdict.Verdicter {
	if o.Verdict == "score" {
		return verdict.Score{}
	}
	return verdict.Strict{}
}

// diagnose builds a snapshot for the given refs and verdicts every finalizer.
func diagnose(ctx context.Context, c cluster.ClusterClient, refs []model.ResourceRef, now time.Time, strat verdict.Verdicter) ([]targetResult, error) {
	snap, err := snapshot.Build(ctx, c, refs, now)
	if err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
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
	return out, nil
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

func renderDryRun(o Options, results []targetResult) string {
	out := renderOut(o, allVerdicts(results), model.Plan{})
	if o.Output == "json" {
		return out
	}
	for _, r := range results {
		if combinedState(r.verdicts) == model.StateDead {
			d := apply.Digest(r.obj.Ref, r.verdicts, r.obj.ResourceVersion)
			out += fmt.Sprintf("to apply: kubectl finalizer-doctor %s --apply --confirm=%s\n", r.obj.Ref.String(), d)
		}
	}
	return out
}

func runApply(ctx context.Context, c cluster.ClusterClient, o Options, results []targetResult, now time.Time) (string, int, error) {
	var out strings.Builder
	for _, r := range results {
		if combinedState(r.verdicts) != model.StateDead {
			fmt.Fprintf(&out, "refused: %s is not all-DEAD; investigate (dry-run for evidence)\n", r.obj.Ref)
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
		// v1.0: orphan-cleanup and webhook-blocker auto-handling are best-effort
		// and not yet wired; the plan clears the proven-dead finalizers only.
		p := plan.Build(r.obj, r.verdicts, nil, false)
		reverify := func(ref model.ResourceRef) (model.State, string, error) {
			rr, err := diagnose(ctx, c, []model.ResourceRef{ref}, now, strategyFor(o))
			if err != nil || len(rr) == 0 {
				return model.StateUnknown, "", err
			}
			return combinedState(rr[0].verdicts), rr[0].obj.ResourceVersion, nil
		}
		res, err := apply.Execute(ctx, c, p, reverify)
		if err != nil {
			fmt.Fprintf(&out, "aborted after %d action(s): %v\n", res.Completed, err)
			return out.String(), 3, nil
		}
		for _, a := range res.Audit {
			fmt.Fprintf(&out, "applied: %s\n", a)
		}
	}
	return out.String(), 0, nil
}

func resolveTargets(ctx context.Context, c cluster.ClusterClient, o Options) ([]model.ResourceRef, error) {
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
func resolveGVR(ctx context.Context, c cluster.ClusterClient, t discover.Target) (model.ResourceRef, error) {
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
