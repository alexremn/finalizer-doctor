package clustertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFakeReadersReturnFixtures(t *testing.T) {
	one := []unstructured.Unstructured{{}}
	f := &Fake{
		Preferred:       []*metav1.APIResourceList{{GroupVersion: "v1"}},
		CRDObjs:         one,
		ValidatingObjs:  one,
		MutatingObjs:    one,
		DeploymentObjs:  one,
		StatefulSetObjs: one,
		DaemonSetObjs:   one,
		PodObjs:         one,
		EndpointObjs:    one,
	}
	ctx := context.Background()

	pr, err := f.ServerPreferredResources(ctx)
	require.NoError(t, err)
	assert.Len(t, pr, 1)

	for _, fn := range []func(context.Context) ([]unstructured.Unstructured, error){
		f.CRDs, f.ValidatingWebhooks, f.MutatingWebhooks,
	} {
		got, err := fn(ctx)
		require.NoError(t, err)
		assert.Len(t, got, 1)
	}
	for _, fn := range []func(context.Context, string) ([]unstructured.Unstructured, error){
		f.Deployments, f.StatefulSets, f.DaemonSets, f.Pods,
	} {
		got, err := fn(ctx, "")
		require.NoError(t, err)
		assert.Len(t, got, 1)
	}
	es, err := f.EndpointSlices(ctx, "ns", "svc")
	require.NoError(t, err)
	assert.Len(t, es, 1)
}

func TestFakeErrorInjectionEverywhere(t *testing.T) {
	boom := assert.AnError
	f := &Fake{Errs: map[string]error{
		"ServerPreferredResources": boom, "List": boom, "CRDs": boom,
		"ValidatingWebhooks": boom, "MutatingWebhooks": boom, "Deployments": boom,
		"StatefulSets": boom, "DaemonSets": boom, "Pods": boom, "EndpointSlices": boom,
		"Can": boom, "PatchFinalizers": boom, "FinalizeNamespace": boom, "Delete": boom,
	}}
	ctx := context.Background()
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}

	_, err := f.ServerPreferredResources(ctx)
	assert.Error(t, err)
	_, err = f.List(ctx, gvr, "")
	assert.Error(t, err)
	_, err = f.CRDs(ctx)
	assert.Error(t, err)
	_, err = f.ValidatingWebhooks(ctx)
	assert.Error(t, err)
	_, err = f.MutatingWebhooks(ctx)
	assert.Error(t, err)
	_, err = f.Deployments(ctx, "")
	assert.Error(t, err)
	_, err = f.StatefulSets(ctx, "")
	assert.Error(t, err)
	_, err = f.DaemonSets(ctx, "")
	assert.Error(t, err)
	_, err = f.Pods(ctx, "")
	assert.Error(t, err)
	_, err = f.EndpointSlices(ctx, "", "")
	assert.Error(t, err)
	_, err = f.Can(ctx, authv1.ResourceAttributes{})
	assert.Error(t, err)
	assert.Error(t, f.PatchFinalizers(ctx, gvr, "n", "x", nil, "1"))
	assert.Error(t, f.FinalizeNamespace(ctx, "x", nil, "1"))
	assert.Error(t, f.Delete(ctx, gvr, "n", "x", "1"))
}

func TestFakeGetMissingErrorsLoudly(t *testing.T) {
	f := &Fake{}
	gvr := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	_, err := f.Get(context.Background(), gvr, "n", "absent")
	assert.Error(t, err, "unconfigured Get must error, not return nil,nil")
}
