package discover

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
)

func TestNamespaceObjectsReturnsCandidatesWithOwnerRefs(t *testing.T) {
	cmGVR := schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	child := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "child", "namespace": "team-a"},
	}}
	child.SetKind("ConfigMap")
	child.SetAPIVersion("v1")
	child.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "example.com/v1", Kind: "Widget", Name: "w1"}})

	f := &clustertest.Fake{
		Preferred: []*metav1.APIResourceList{{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{Name: "configmaps", Namespaced: true, Kind: "ConfigMap", Verbs: metav1.Verbs{"list"}},
				{Name: "nodes", Namespaced: false, Verbs: metav1.Verbs{"list"}},      // skipped: cluster-scoped
				{Name: "pods/status", Namespaced: true, Verbs: metav1.Verbs{"list"}}, // skipped: subresource
				{Name: "secrets", Namespaced: true, Verbs: metav1.Verbs{"get"}},      // skipped: not listable
			},
		}},
		Lists: map[string]*unstructured.UnstructuredList{
			clustertest.ListKey(cmGVR, "team-a"): {Items: []unstructured.Unstructured{child}},
		},
	}

	got, err := NamespaceObjects(context.Background(), f, "team-a")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "child", got[0].Ref.Name)
	assert.Equal(t, "ConfigMap", got[0].Kind)
	require.Len(t, got[0].OwnerRefs, 1)
	assert.Equal(t, "Widget", got[0].OwnerRefs[0].Kind)
	assert.Equal(t, "w1", got[0].OwnerRefs[0].Name)
}

func TestNamespaceObjectsDiscoveryError(t *testing.T) {
	f := &clustertest.Fake{Errs: map[string]error{"ServerPreferredResources": errors.New("boom")}}
	_, err := NamespaceObjects(context.Background(), f, "team-a")
	assert.Error(t, err)
}
