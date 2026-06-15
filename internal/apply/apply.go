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

// ReverifyFunc re-snapshots and re-runs the verdict for one target, returning
// the fresh state and resourceVersion. Injected so apply stays testable.
type ReverifyFunc func(model.ResourceRef) (model.State, string, error)

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
		state, rv, err := reverify(a.Target)
		if err != nil {
			return res, fmt.Errorf("re-verify %s: %w", a.Target, err)
		}
		if state != model.StateDead {
			return res, ErrReverifyChanged
		}
		if err := perform(ctx, c, a, rv); err != nil {
			return res, fmt.Errorf("action %s on %s: %w", a.Kind, a.Target, err)
		}
		res.Completed++
		res.Audit = append(res.Audit, fmt.Sprintf("%s %s finalizer=%s rv=%s", a.Kind, a.Target, a.Finalizer, rv))
	}
	return res, nil
}

func perform(ctx context.Context, c cluster.Client, a model.Action, rv string) error {
	switch a.Kind {
	case model.ActionCleanOrphan:
		return c.Delete(ctx, a.Target.GVR, a.Target.Namespace, a.Target.Name, rv)
	case model.ActionFinalizeNamespace:
		return c.FinalizeNamespace(ctx, a.Target.Name, nil, rv) // empties spec.finalizers
	case model.ActionClearFinalizer:
		return c.PatchFinalizers(ctx, a.Target.GVR, a.Target.Namespace, a.Target.Name, nil, rv)
	default:
		return fmt.Errorf("unknown action kind %q", a.Kind)
	}
}
