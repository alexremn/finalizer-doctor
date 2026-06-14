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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

var widgetGVR = schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}

func newUnstruct(apiVersion, kind, name, ns string) *unstructured.Unstructured {
	o := &unstructured.Unstructured{}
	o.SetAPIVersion(apiVersion)
	o.SetKind(kind)
	o.SetName(name)
	if ns != "" {
		o.SetNamespace(ns)
	}
	return o
}

func fakeDynClient(objs ...runtime.Object) *clientGo {
	scheme := runtime.NewScheme()
	listKinds := map[schema.GroupVersionResource]string{
		gvrAPIServices: "APIServiceList",
		gvrCRDs:        "CustomResourceDefinitionList",
		widgetGVR:      "WidgetList",
	}
	dyn := dynfake.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, objs...)
	return &clientGo{dyn: dyn}
}

func TestClientGoDynamicReadAndMutate(t *testing.T) {
	ctx := context.Background()
	as := newUnstruct("apiregistration.k8s.io/v1", "APIService", "v1.example.com", "")
	widget := newUnstruct("example.com/v1", "Widget", "w1", "team-a")
	widget.SetFinalizers([]string{"example.com/cleanup"})
	c := fakeDynClient(as, widget)

	// APIServices read.
	list, err := c.APIServices(ctx)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	// Get.
	got, err := c.Get(ctx, widgetGVR, "team-a", "w1")
	require.NoError(t, err)
	assert.Equal(t, "w1", got.GetName())

	// List (namespaced).
	l, err := c.List(ctx, widgetGVR, "team-a")
	require.NoError(t, err)
	assert.Len(t, l.Items, 1)

	// PatchFinalizers clears the finalizer.
	require.NoError(t, c.PatchFinalizers(ctx, widgetGVR, "team-a", "w1", nil, got.GetResourceVersion()))
	after, err := c.Get(ctx, widgetGVR, "team-a", "w1")
	require.NoError(t, err)
	assert.Empty(t, after.GetFinalizers())

	// Delete.
	require.NoError(t, c.Delete(ctx, widgetGVR, "team-a", "w1", after.GetResourceVersion()))
	_, err = c.Get(ctx, widgetGVR, "team-a", "w1")
	assert.True(t, apierrors.IsNotFound(err))
}

func TestClientGoCan(t *testing.T) {
	cs := k8sfake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, &authv1.SelfSubjectAccessReview{Status: authv1.SubjectAccessReviewStatus{Allowed: true}}, nil
	})
	c := &clientGo{cs: cs}
	ok, err := c.Can(context.Background(), authv1.ResourceAttributes{Verb: "patch", Resource: "namespaces"})
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestClientGoFinalizeNamespaceResourceVersionMismatch(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foo", ResourceVersion: "5"}}
	cs := k8sfake.NewSimpleClientset(ns)
	c := &clientGo{cs: cs}

	// rv mismatch -> conflict, no finalize.
	err := c.FinalizeNamespace(context.Background(), "foo", nil, "4")
	assert.True(t, apierrors.IsConflict(err), "stale resourceVersion must conflict")

	// rv match -> finalize succeeds.
	require.NoError(t, c.FinalizeNamespace(context.Background(), "foo", nil, "5"))
}
