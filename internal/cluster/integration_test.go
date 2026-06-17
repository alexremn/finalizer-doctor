//go:build integration

// Integration tests against a real kube-apiserver via envtest (no nodes/kubelet).
// Run: KUBEBUILDER_ASSETS=$(hack/setup-envtest.sh) go test -tags integration ./internal/cluster/...
// These cover the real-API behaviors the fake client can't: merge-patch finalizer
// removal, the namespace /finalize subresource, and SelfSubjectAccessReview.
package cluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cmGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

func startEnv(t *testing.T) *rest.Config {
	t.Helper()
	env := &envtest.Environment{}
	cfg, err := env.Start()
	require.NoError(t, err, "envtest start failed — set KUBEBUILDER_ASSETS (hack/setup-envtest.sh)")
	t.Cleanup(func() { _ = env.Stop() })
	return cfg
}

func TestIntegrationPatchFinalizersSurgical(t *testing.T) {
	cfg := startEnv(t)
	c, err := NewFromConfig(cfg)
	require.NoError(t, err)
	dyn, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	ctx := context.Background()

	cm := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "stuck", "namespace": "default", "finalizers": []any{"a/x", "b/y"}},
	}}
	_, err = dyn.Resource(cmGVR).Namespace("default").Create(ctx, cm, metav1.CreateOptions{})
	require.NoError(t, err)
	// Delete -> deletionTimestamp set; object persists because finalizers remain
	// (no GC controller in envtest, just like a stuck object in the wild).
	require.NoError(t, dyn.Resource(cmGVR).Namespace("default").Delete(ctx, "stuck", metav1.DeleteOptions{}))

	// Surgical removal of a/x must leave b/y intact (real merge-patch semantics).
	cur, err := c.Get(ctx, cmGVR, "default", "stuck")
	require.NoError(t, err)
	require.NoError(t, c.PatchFinalizers(ctx, cmGVR, "default", "stuck", []string{"b/y"}, cur.GetResourceVersion()))
	cur, err = c.Get(ctx, cmGVR, "default", "stuck")
	require.NoError(t, err)
	assert.Equal(t, []string{"b/y"}, cur.GetFinalizers())

	// Removing the last finalizer lets the apiserver complete deletion.
	require.NoError(t, c.PatchFinalizers(ctx, cmGVR, "default", "stuck", []string{}, cur.GetResourceVersion()))
	_, err = c.Get(ctx, cmGVR, "default", "stuck")
	assert.True(t, apierrors.IsNotFound(err))
}

func TestIntegrationCan(t *testing.T) {
	c, err := NewFromConfig(startEnv(t))
	require.NoError(t, err)
	ok, err := c.Can(context.Background(), authv1.ResourceAttributes{Verb: "patch", Resource: "configmaps", Namespace: "default"})
	require.NoError(t, err)
	assert.True(t, ok) // the envtest client is cluster-admin
}

func TestIntegrationFinalizeNamespace(t *testing.T) {
	cfg := startEnv(t)
	c, err := NewFromConfig(cfg)
	require.NoError(t, err)
	cs, err := kubernetes.NewForConfig(cfg)
	require.NoError(t, err)
	ctx := context.Background()

	_, err = cs.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "stuck-ns"}}, metav1.CreateOptions{})
	require.NoError(t, err)
	// Delete -> Terminating, holding the built-in `kubernetes` spec finalizer.
	require.NoError(t, cs.CoreV1().Namespaces().Delete(ctx, "stuck-ns", metav1.DeleteOptions{}))
	got, err := cs.CoreV1().Namespaces().Get(ctx, "stuck-ns", metav1.GetOptions{})
	require.NoError(t, err)

	// /finalize with empty spec.finalizers lets the apiserver finish deletion.
	require.NoError(t, c.FinalizeNamespace(ctx, "stuck-ns", nil, got.ResourceVersion))
	_, err = cs.CoreV1().Namespaces().Get(ctx, "stuck-ns", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err))
}
