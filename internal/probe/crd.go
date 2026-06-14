package probe

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// probeCRD reports a hard dead signal when the owner is a CRD-backed group whose
// CRD is absent from the cluster.
func probeCRD(owner model.OwnerCandidate, snap model.Snapshot) *model.Evidence {
	if owner.Kind != "CRD" {
		return nil
	}
	if !snap.Readable(model.SourceCRDs) {
		return &model.Evidence{Signal: "CRD presence", Source: model.SourceCRDs, Observed: "could not read CRDs", Class: model.ClassUnreadable}
	}
	for _, crd := range snap.RawCRDs {
		if g, _, _ := unstructured.NestedString(crd.Object, "spec", "group"); g == owner.Group {
			return nil // CRD present
		}
	}
	return &model.Evidence{Signal: "CRD presence", Source: model.SourceCRDs, Observed: "no CRD for group " + owner.Group, Class: model.ClassDeadHard, Weight: 40}
}
