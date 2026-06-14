// Package orphan detects Kubernetes objects orphaned by a dead controller.
// Detection is best-effort and limited to objects owned by the target
// (safety-model.md §7); external infra is out of scope.
package orphan

import "github.com/alexremn/finalizer-doctor/internal/model"

// Detect returns objects whose ownerReferences point at the target.
func Detect(target model.ResourceRef, candidates []model.StuckObject) []model.ResourceRef {
	var out []model.ResourceRef
	for _, c := range candidates {
		for _, owner := range c.OwnerRefs {
			if owner.Name == target.Name && owner.GVR == target.GVR {
				out = append(out, c.Ref)
				break
			}
		}
	}
	return out
}
