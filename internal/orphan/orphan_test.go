package orphan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func nsGVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
}

func TestDetectByOwnerRef(t *testing.T) {
	target := model.ResourceRef{Name: "foo", GVR: nsGVR()}
	candidates := []model.StuckObject{
		{Ref: model.ResourceRef{Name: "child", Namespace: "foo"}, OwnerRefs: []model.ResourceRef{target}},
		{Ref: model.ResourceRef{Name: "unrelated"}},
	}
	orphans := Detect(target, candidates)
	assert.Len(t, orphans, 1)
	assert.Equal(t, "child", orphans[0].Name)
}
