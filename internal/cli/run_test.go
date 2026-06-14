package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func asMetaTime() *metav1.Time { t := metav1.Now(); return &t }

func deadAPIService() unstructured.Unstructured {
	as := unstructured.Unstructured{Object: map[string]any{
		"spec":   map[string]any{"group": "example.com", "service": map[string]any{"name": "m", "namespace": "kube-system"}},
		"status": map[string]any{"conditions": []any{map[string]any{"type": "Available", "status": "False", "reason": "ServiceNotFound"}}},
	}}
	as.SetName("v1.example.com")
	return as
}

func stuckCR() unstructured.Unstructured {
	cr := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "w1", "namespace": "team-a", "resourceVersion": "9"},
	}}
	cr.SetFinalizers([]string{"example.com/cleanup"})
	cr.SetDeletionTimestamp(asMetaTime())
	return cr
}

func TestRunDryRunReportsDeadViaAPIService(t *testing.T) {
	cr := stuckCR()
	f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		return &cr, nil
	}

	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Output: "json", Verdict: "strict"})
	require.NoError(t, err)
	assert.Equal(t, 2, code, "stuck found -> exit 2 in dry-run")
	assert.Contains(t, out, `"state": "DEAD"`)
}

func TestRunApplyAllIsInvalidInvocation(t *testing.T) {
	f := &clustertest.Fake{}
	_, code, err := Run(context.Background(), f, Options{All: true, Apply: true})
	assert.Equal(t, 1, code)
	var inv *InvalidInvocation
	assert.ErrorAs(t, err, &inv)
}

func TestRunApplyNonInteractiveBadDigestRefused(t *testing.T) {
	cr := stuckCR()
	f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		return &cr, nil
	}
	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: "wrongtoken00"})
	require.NoError(t, err)
	assert.Equal(t, 3, code, "bad digest -> refused")
	assert.Contains(t, out, "refused")
	_ = model.StateDead
}
