package clustertest

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestFakeCanRespectsAllowSet(t *testing.T) {
	f := &Fake{Allowed: map[string]bool{"patch:namespaces:": true}}
	ok, err := f.Can(context.Background(), authv1.ResourceAttributes{Verb: "patch", Resource: "namespaces"})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = f.Can(context.Background(), authv1.ResourceAttributes{Verb: "delete", Resource: "pods"})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFakeRecordsMutations(t *testing.T) {
	f := &Fake{}
	require.NoError(t, f.FinalizeNamespace(context.Background(), "foo", nil, "42"))
	require.Len(t, f.Mutations, 1)
	assert.Equal(t, "FinalizeNamespace foo rv=42", f.Mutations[0])
}

func TestFakeGetReturnsObjectFromGetsMap(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	want := &unstructured.Unstructured{}
	want.SetName("my-deploy")
	f := &Fake{
		Gets: map[string]*unstructured.Unstructured{
			GetKey(gvr, "default", "my-deploy"): want,
		},
	}
	got, err := f.Get(context.Background(), gvr, "default", "my-deploy")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFakeGetCallsGetFn(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	sentinel := &unstructured.Unstructured{}
	sentinel.SetName("sentinel")
	f := &Fake{
		GetFn: func(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
			return sentinel, nil
		},
	}
	got, err := f.Get(context.Background(), gvr, "kube-system", "some-pod")
	require.NoError(t, err)
	assert.Equal(t, sentinel, got)
}

func TestFakeAPIServicesInjectedErrorReturnsNilData(t *testing.T) {
	injected := errors.New("api-services unavailable")
	f := &Fake{
		APIServiceObjs: []unstructured.Unstructured{{}, {}}, // would be non-nil on success
		Errs:           map[string]error{"APIServices": injected},
	}
	got, err := f.APIServices(context.Background())
	require.ErrorIs(t, err, injected)
	assert.Nil(t, got, "expected nil data when error is injected")
}
