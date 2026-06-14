package orphan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestDetectByOwnerRef(t *testing.T) {
	target := model.StuckObject{
		Ref:        model.ResourceRef{Name: "foo", GVR: schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}},
		Kind:       "Widget",
		APIVersion: "example.com/v1",
	}
	candidates := []model.StuckObject{
		{
			Ref:       model.ResourceRef{Name: "child", Namespace: "team-a", GVR: schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}},
			OwnerRefs: []model.OwnerRef{{APIVersion: "example.com/v1", Kind: "Widget", Name: "foo"}},
		},
		{Ref: model.ResourceRef{Name: "unrelated"}},
		{Ref: model.ResourceRef{Name: "wrong-owner"}, OwnerRefs: []model.OwnerRef{{APIVersion: "example.com/v1", Kind: "Widget", Name: "other"}}},
	}
	orphans := Detect(target, candidates)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "child", orphans[0].Name)
	assert.Equal(t, "configmaps", orphans[0].GVR.Resource)
}

func TestDetectIgnoresTargetItself(t *testing.T) {
	ref := model.ResourceRef{Name: "foo", GVR: schema.GroupVersionResource{Resource: "widgets"}}
	target := model.StuckObject{Ref: ref, Kind: "Widget", APIVersion: "example.com/v1"}
	// A self-referential ownerRef must not make the target its own orphan.
	candidates := []model.StuckObject{{Ref: ref, OwnerRefs: []model.OwnerRef{{APIVersion: "example.com/v1", Kind: "Widget", Name: "foo"}}}}
	assert.Empty(t, Detect(target, candidates))
}
