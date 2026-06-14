// Package probe gathers liveness evidence for a finalizer's owner from a
// Snapshot. It is pure: Snapshot in, []model.Evidence out.
package probe

import "github.com/alexremn/finalizer-doctor/internal/model"

// For returns all evidence for one finalizer's owner. The owner carries the
// resolved API group (owner.Group) the probes match against.
func For(owner model.OwnerCandidate, snap model.Snapshot) []model.Evidence {
	var ev []model.Evidence
	if e := probeAPIService(owner, snap); e != nil {
		ev = append(ev, *e)
	}
	if e := probeCRD(owner, snap); e != nil {
		ev = append(ev, *e)
	}
	ev = append(ev, probeWorkload(owner, snap)...)
	return ev
}
