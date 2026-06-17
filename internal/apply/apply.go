package apply

import (
	"context"
	"errors"
	"fmt"

	"github.com/alexremn/finalizer-doctor/internal/cluster"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

// ErrReverifyChanged is returned when a per-action re-verify is no longer DEAD.
var ErrReverifyChanged = errors.New("re-verify no longer DEAD; aborting")

// ReverifyResult is the fresh state read immediately before a mutation: the
// parent verdict, the action target's resourceVersion, and its CURRENT
// finalizers (so the clear is surgical, not a blind wipe).
type ReverifyResult struct {
	State              model.State
	ResourceVersion    string
	MetadataFinalizers []string
	SpecFinalizers     []string
}

// ReverifyFunc re-snapshots and re-runs the verdict for one target. Injected so
// apply stays testable.
type ReverifyFunc func(model.ResourceRef) (ReverifyResult, error)

// Result reports how far execution got.
type Result struct {
	Completed int
	Audit     []string
}

// Execute runs the plan's actions in order, re-verifying DEAD immediately before
// each irreversible action and pinning resourceVersion on the mutation
// (safety-model.md §4, §6, §11). Stops on first error.
func Execute(ctx context.Context, c cluster.Client, plan model.Plan, reverify ReverifyFunc) (Result, error) {
	var res Result
	for _, a := range plan.Actions {
		rr, err := reverify(a.Target)
		if err != nil {
			return res, fmt.Errorf("re-verify %s: %w", a.Target, err)
		}
		if rr.State != model.StateDead {
			return res, ErrReverifyChanged
		}
		if err := perform(ctx, c, a, rr); err != nil {
			return res, fmt.Errorf("action %s on %s: %w", a.Kind, a.Target, err)
		}
		res.Completed++
		res.Audit = append(res.Audit, fmt.Sprintf("%s %s finalizer=%s rv=%s", a.Kind, a.Target, a.Finalizer, rr.ResourceVersion))
	}
	return res, nil
}

func perform(ctx context.Context, c cluster.Client, a model.Action, rr ReverifyResult) error {
	switch a.Kind {
	case model.ActionCleanOrphan:
		return c.Delete(ctx, a.Target.GVR, a.Target.Namespace, a.Target.Name, rr.ResourceVersion)
	case model.ActionFinalizeNamespace:
		// Remove only the named spec finalizer, preserving any others.
		return c.FinalizeNamespace(ctx, a.Target.Name, removeFinalizer(rr.SpecFinalizers, a.Finalizer), rr.ResourceVersion)
	case model.ActionClearFinalizer:
		// Remove only the named metadata finalizer, preserving any others.
		return c.PatchFinalizers(ctx, a.Target.GVR, a.Target.Namespace, a.Target.Name, removeFinalizer(rr.MetadataFinalizers, a.Finalizer), rr.ResourceVersion)
	default:
		return fmt.Errorf("unknown action kind %q", a.Kind)
	}
}

// removeFinalizer returns list with x removed (least-blast-radius: other
// finalizers, incl. any added after the snapshot, are preserved).
func removeFinalizer(list []string, x string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if s != x {
			out = append(out, s)
		}
	}
	return out
}
