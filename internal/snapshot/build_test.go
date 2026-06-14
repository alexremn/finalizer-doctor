package snapshot

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestBuildMarksUnreadableSource(t *testing.T) {
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	tgt := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "foo", "resourceVersion": "7"},
		"spec":     map[string]any{"finalizers": []any{"kubernetes"}},
	}}
	ts := metav1.NewTime(time.Unix(1000, 0))
	tgt.SetDeletionTimestamp(&ts)
	tgt.SetFinalizers(nil)

	ref := model.ResourceRef{GVR: gvr, Name: "foo"}

	f := &clustertest.Fake{
		Gets: map[string]*unstructured.Unstructured{
			clustertest.GetKey(gvr, ref.Namespace, ref.Name): &tgt,
		},
		Errs: map[string]error{"Deployments": assert.AnError},
	}

	snap, err := Build(context.Background(), f, []model.ResourceRef{ref}, time.Unix(100, 0))
	require.NoError(t, err)
	require.Len(t, snap.Targets, 1)
	assert.Equal(t, []string{"kubernetes"}, snap.Targets[0].SpecFinalizers)
	assert.Equal(t, "7", snap.Targets[0].ResourceVersion)
	assert.False(t, snap.Readable(model.SourceWorkloads), "errored source must be Unreadable")
	assert.True(t, snap.Readable(model.SourceAPIServices))
}
