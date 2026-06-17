// Package probe gathers liveness evidence for a finalizer's owner from a
// Snapshot. It is pure: Snapshot in, []model.Evidence out.
package probe

import "github.com/alexremn/finalizer-doctor/internal/model"

// For returns all evidence for one finalizer's owner. The owner carries the
// resolved API group (owner.Group) the probes match against.
func For(owner model.OwnerCandidate, snap model.Snapshot) []model.Evidence {
	if owner.Kind == "Builtin" {
		// Built-in finalizers are owned by core control-plane controllers
		// (kube-controller-manager), which are essentially never dead. The
		// finalizer is held for a reason (e.g. a Pod still using a PVC), not
		// because a controller crashed -> treat as live, verdict SLOW.
		// ponytail: matching arbitrary *-operator/*-controller-manager Deployments
		// was unsound (false attribution); owner->workload mapping deferred to v0.2.
		return []model.Evidence{{
			Signal:   "built-in owner",
			Source:   model.SourceTargets,
			Observed: owner.MatchReason + " — core control-plane controller assumed running; finalizer held for a reason, not dead",
			Class:    model.ClassLive,
		}}
	}
	var ev []model.Evidence
	if e := probeAPIService(owner, snap); e != nil {
		ev = append(ev, *e)
	}
	if e := probeCRD(owner, snap); e != nil {
		ev = append(ev, *e)
	}
	return ev
}
