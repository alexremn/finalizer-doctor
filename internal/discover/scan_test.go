package discover

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
)

func stuckObj(name string) unstructured.Unstructured {
	o := unstructured.Unstructured{}
	o.SetName(name)
	ts := metav1.Now()
	o.SetDeletionTimestamp(&ts)
	o.SetFinalizers([]string{"example.com/cleanup"})
	return o
}

func TestScanFindsOnlyStuck(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	notStuck := unstructured.Unstructured{}
	notStuck.SetName("healthy")

	f := &clustertest.Fake{
		Preferred: []*metav1.APIResourceList{{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{{Name: "widgets", Namespaced: true, Verbs: metav1.Verbs{"list"}}},
		}},
		Lists: map[string]*unstructured.UnstructuredList{
			clustertest.ListKey(gvr, ""): {Items: []unstructured.Unstructured{stuckObj("w1"), notStuck}},
		},
	}

	refs, err := Scan(context.Background(), f)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "w1", refs[0].Name)
	assert.Equal(t, "widgets", refs[0].GVR.Resource)
}

// stuckNamespace mirrors a real stuck namespace: the `kubernetes` finalizer lives
// in spec.finalizers and metadata.finalizers is empty.
func stuckNamespace(name string) unstructured.Unstructured {
	o := unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"finalizers": []any{"kubernetes"}}}}
	o.SetName(name)
	ts := metav1.Now()
	o.SetDeletionTimestamp(&ts)
	return o
}

func TestScanFindsNamespaceStuckOnSpecFinalizer(t *testing.T) {
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	f := &clustertest.Fake{
		Preferred: []*metav1.APIResourceList{{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{{Name: "namespaces", Verbs: metav1.Verbs{"list"}}},
		}},
		Lists: map[string]*unstructured.UnstructuredList{
			clustertest.ListKey(gvr, ""): {Items: []unstructured.Unstructured{stuckNamespace("demo-stuck")}},
		},
	}

	refs, err := Scan(context.Background(), f)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "demo-stuck", refs[0].Name)
}

func TestIsStuck(t *testing.T) {
	live := unstructured.Unstructured{}
	live.SetFinalizers([]string{"example.com/cleanup"})
	deletingNoFinalizers := unstructured.Unstructured{}
	ts := metav1.Now()
	deletingNoFinalizers.SetDeletionTimestamp(&ts)
	ns, cr := stuckNamespace("ns"), stuckObj("w1")

	assert.True(t, IsStuck(&cr), "deleting + metadata finalizer")
	assert.True(t, IsStuck(&ns), "deleting + spec finalizer")
	assert.False(t, IsStuck(&live), "finalizer but not deleting")
	assert.False(t, IsStuck(&deletingNoFinalizers), "deleting with nothing holding it")
}
