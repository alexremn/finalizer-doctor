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

func TestCombinedState(t *testing.T) {
	dead := []model.Verdict{{State: model.StateDead}, {State: model.StateDead}}
	assert.Equal(t, model.StateDead, combinedState(dead))

	assert.Equal(t, model.StateSlow, combinedState([]model.Verdict{{State: model.StateDead}, {State: model.StateSlow}}))
	assert.Equal(t, model.StateUnknown, combinedState([]model.Verdict{{State: model.StateDead}, {State: model.StateUnknown}}))
	assert.Equal(t, model.StateUnknown, combinedState(nil))
}

func TestMatchesResource(t *testing.T) {
	r := metav1.APIResource{Name: "persistentvolumeclaims", SingularName: "persistentvolumeclaim", Kind: "PersistentVolumeClaim", ShortNames: []string{"pvc"}}
	assert.True(t, matchesResource(r, "persistentvolumeclaims"))
	assert.True(t, matchesResource(r, "persistentvolumeclaim"))
	assert.True(t, matchesResource(r, "pvc"))
	assert.True(t, matchesResource(r, "PersistentVolumeClaim"))
	assert.False(t, matchesResource(r, "pods"))
}

// widgetRef is the ResourceRef the fallback resolver produces for the test CR.
func widgetRef() model.ResourceRef {
	return model.ResourceRef{
		GVR:       schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"},
		Namespace: "team-a", Name: "w1",
	}
}

func deadCRFake() *clustertest.Fake {
	cr := stuckCR() // finalizer example.com/cleanup, rv 9, ns team-a, name w1
	f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		c := cr
		return &c, nil
	}
	return f
}

func TestRunApplySucceedsWithMatchingDigest(t *testing.T) {
	f := deadCRFake()
	// The dry-run would print this digest; compute the same proof here.
	digest := apply.Digest(widgetRef(), []model.Verdict{{Finalizer: "example.com/cleanup", State: model.StateDead}}, "9")

	out, code, err := Run(context.Background(), f, Options{
		Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: digest,
	})
	require.NoError(t, err)
	assert.Equal(t, 0, code, "matching digest -> applied -> exit 0")
	assert.Contains(t, out, "applied")
	require.Len(t, f.Mutations, 1)
	assert.Contains(t, f.Mutations[0], "PatchFinalizers")
}

func TestRunDryRunPrintsApplyHintWithDigest(t *testing.T) {
	f := deadCRFake()
	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Output: "human"})
	require.NoError(t, err)
	assert.Equal(t, 2, code)
	digest := apply.Digest(widgetRef(), []model.Verdict{{Finalizer: "example.com/cleanup", State: model.StateDead}}, "9")
	assert.Contains(t, out, "--confirm="+digest)
}
