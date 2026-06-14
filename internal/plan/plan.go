// Package plan builds the ordered remediation plan from verdicts
// (safety-model.md §6, §8; verdict-engine.md §9.4-9.5).
package plan

import "github.com/alexremn/finalizer-doctor/internal/model"

// Build produces the plan. It refuses (no actions) unless every blocking
// finalizer is DEAD, and refuses on an unresolved webhook blocker. Action order:
// orphans first, then metadata finalizer clears, then the namespace /finalize
// (content-deleting) last.
func Build(obj model.StuckObject, verdicts []model.Verdict, orphans []model.ResourceRef, webhookBlocker bool) model.Plan {
	var p model.Plan

	// Joint gate: ALL blocking finalizers must be DEAD (incl. namespace
	// spec+metadata, which are both present in verdicts).
	for _, v := range verdicts {
		if v.State != model.StateDead {
			p.Refused = append(p.Refused, v)
		}
	}
	if len(p.Refused) > 0 {
		return p
	}

	if webhookBlocker {
		p.Notes = append(p.Notes, "refused: a failurePolicy=Fail webhook with dead backing would reject the clear; remove/fix it and re-run")
		return p
	}

	// 1. Orphans first.
	for _, o := range orphans {
		p.Actions = append(p.Actions, model.Action{Kind: model.ActionCleanOrphan, Target: o, Reason: "owned by dead controller's resource", Reversible: false})
	}

	// 2. Metadata finalizer clears, then 3. namespace /finalize last.
	isNamespace := obj.Ref.GVR.Resource == "namespaces"
	var finalizeAction *model.Action
	for _, v := range verdicts {
		if isNamespace && v.Finalizer == "kubernetes" {
			finalizeAction = &model.Action{Kind: model.ActionFinalizeNamespace, Target: obj.Ref, Finalizer: v.Finalizer, Reason: "dead aggregated API; content orphaned", Reversible: false}
			continue
		}
		p.Actions = append(p.Actions, model.Action{Kind: model.ActionClearFinalizer, Target: obj.Ref, Finalizer: v.Finalizer, Reason: "owner proven dead", Reversible: false})
	}
	if finalizeAction != nil {
		p.Actions = append(p.Actions, *finalizeAction)
	}
	return p
}
