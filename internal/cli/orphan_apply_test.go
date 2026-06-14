package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/apply"
	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func widgetCR() unstructured.Unstructured {
	cr := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "w1", "namespace": "team-a", "resourceVersion": "9"},
	}}
	cr.SetAPIVersion("example.com/v1")
	cr.SetKind("Widget")
	cr.SetFinalizers([]string{"example.com/cleanup"})
	cr.SetDeletionTimestamp(asMetaTime())
	return cr
}

func childConfigMap() unstructured.Unstructured {
	cm := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "child", "namespace": "team-a", "resourceVersion": "2"},
	}}
	cm.SetAPIVersion("v1")
	cm.SetKind("ConfigMap")
	cm.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "example.com/v1", Kind: "Widget", Name: "w1"}})
	return cm
}

func TestRunApplyDeletesOrphanThenClears(t *testing.T) {
	cm := childConfigMap()
	cr := widgetCR()
	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

	f := &clustertest.Fake{
		APIServiceObjs: []unstructured.Unstructured{deadAPIService()}, // dead example.com group
		Preferred: []*metav1.APIResourceList{{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{{Name: "configmaps", Namespaced: true, Kind: "ConfigMap", Verbs: metav1.Verbs{"list"}}},
		}},
		Lists: map[string]*unstructured.UnstructuredList{
			clustertest.ListKey(cmGVR, "team-a"): {Items: []unstructured.Unstructured{cm}},
		},
	}
	f.GetFn = func(_ context.Context, _ schema.GroupVersionResource, _, name string) (*unstructured.Unstructured, error) {
		switch name {
		case "child":
			c := cm
			return &c, nil
		default:
			c := cr
			return &c, nil
		}
	}

	digest := apply.Digest(widgetRef(), []model.Verdict{{Finalizer: "example.com/cleanup", State: model.StateDead}}, "9")
	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: digest})

	require.NoError(t, err)
	assert.Equal(t, 0, code)
	require.Len(t, f.Mutations, 2, "orphan delete then finalizer clear")
	assert.Contains(t, f.Mutations[0], "Delete")
	assert.Contains(t, f.Mutations[0], "child")
	assert.Contains(t, f.Mutations[1], "PatchFinalizers")
	assert.Contains(t, out, "applied")
}
