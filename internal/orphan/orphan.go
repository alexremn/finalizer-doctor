// Package orphan detects Kubernetes objects orphaned by a dead controller.
// Detection is best-effort and limited to objects owned by the target via an
// ownerReference (safety-model.md §7); external infra is out of scope.
package orphan

import "github.com/alexremn/finalizer-doctor/internal/model"

// Detect returns candidate objects whose ownerReferences point at the target.
// Matching is by APIVersion+Kind+Name (ownerReferences carry Kind, not GVR); each
// returned ref is the candidate's own GVR, suitable for deletion.
func Detect(target model.StuckObject, candidates []model.StuckObject) []model.ResourceRef {
	var out []model.ResourceRef
	for _, c := range candidates {
		if c.Ref == target.Ref {
			continue // never treat the target as its own orphan
		}
		for _, owner := range c.OwnerRefs {
			if owner.Name == target.Ref.Name && owner.Kind == target.Kind && owner.APIVersion == target.APIVersion {
				out = append(out, c.Ref)
				break
			}
		}
	}
	return out
}
