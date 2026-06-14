package cluster

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	authv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GVRs for the evidence sources, read via the dynamic client as unstructured.
var (
	gvrAPIServices = schema.GroupVersionResource{Group: "apiregistration.k8s.io", Version: "v1", Resource: "apiservices"}
	gvrCRDs        = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}
	gvrValidating  = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "validatingwebhookconfigurations"}
	gvrMutating    = schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingwebhookconfigurations"}
	gvrDeployments = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	gvrStatefulSet = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
	gvrDaemonSet   = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}
	gvrPods        = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	gvrEndpointSli = schema.GroupVersionResource{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"}
)

type clientGo struct {
	dyn  dynamic.Interface
	disc discovery.DiscoveryInterface
	cs   kubernetes.Interface
}

// NewFromConfig builds a ClusterClient from a rest.Config.
func NewFromConfig(cfg *rest.Config) (ClusterClient, error) {
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("dynamic client: %w", err)
	}
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("discovery client: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("clientset: %w", err)
	}
	return &clientGo{dyn: dyn, disc: disc, cs: cs}, nil
}

func (c *clientGo) ServerPreferredResources(_ context.Context) ([]*metav1.APIResourceList, error) {
	return c.disc.ServerPreferredResources()
}

func (c *clientGo) List(ctx context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	if ns == "" {
		return c.dyn.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	return c.dyn.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{})
}

func (c *clientGo) Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	r := c.dyn.Resource(gvr)
	if ns != "" {
		return r.Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	}
	return r.Get(ctx, name, metav1.GetOptions{})
}

func (c *clientGo) listItems(ctx context.Context, gvr schema.GroupVersionResource, ns string) ([]unstructured.Unstructured, error) {
	l, err := c.List(ctx, gvr, ns)
	if err != nil {
		return nil, err
	}
	return l.Items, nil
}

func (c *clientGo) APIServices(ctx context.Context) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrAPIServices, "")
}
func (c *clientGo) CRDs(ctx context.Context) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrCRDs, "")
}
func (c *clientGo) ValidatingWebhooks(ctx context.Context) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrValidating, "")
}
func (c *clientGo) MutatingWebhooks(ctx context.Context) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrMutating, "")
}
func (c *clientGo) Deployments(ctx context.Context, ns string) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrDeployments, ns)
}
func (c *clientGo) StatefulSets(ctx context.Context, ns string) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrStatefulSet, ns)
}
func (c *clientGo) DaemonSets(ctx context.Context, ns string) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrDaemonSet, ns)
}
func (c *clientGo) Pods(ctx context.Context, ns string) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrPods, ns)
}
func (c *clientGo) EndpointSlices(ctx context.Context, ns, _ string) ([]unstructured.Unstructured, error) {
	return c.listItems(ctx, gvrEndpointSli, ns)
}

func (c *clientGo) Can(ctx context.Context, attrs authv1.ResourceAttributes) (bool, error) {
	r, err := c.cs.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx,
		&authv1.SelfSubjectAccessReview{Spec: authv1.SelfSubjectAccessReviewSpec{ResourceAttributes: &attrs}},
		metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("access review: %w", err)
	}
	return r.Status.Allowed, nil
}

func (c *clientGo) PatchFinalizers(ctx context.Context, gvr schema.GroupVersionResource, ns, name string, newFinalizers []string, rv string) error {
	patch, err := finalizersMergePatch(newFinalizers, rv)
	if err != nil {
		return err
	}
	r := c.dyn.Resource(gvr)
	if ns != "" {
		_, err = r.Namespace(ns).Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
		return err
	}
	_, err = r.Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	return err
}

func (c *clientGo) FinalizeNamespace(ctx context.Context, name string, newSpecFinalizers []string, rv string) error {
	ns, err := c.cs.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get namespace %s: %w", name, err)
	}
	if ns.ResourceVersion != rv {
		return apierrors.NewConflict(
			schema.GroupResource{Resource: "namespaces"}, name,
			fmt.Errorf("resourceVersion changed (have %s, want %s)", ns.ResourceVersion, rv))
	}
	ns.Spec.Finalizers = toFinalizerNames(newSpecFinalizers)
	_, err = c.cs.CoreV1().Namespaces().Finalize(ctx, ns, metav1.UpdateOptions{})
	return err
}

func (c *clientGo) Delete(ctx context.Context, gvr schema.GroupVersionResource, ns, name, rv string) error {
	opts := metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &rv}}
	r := c.dyn.Resource(gvr)
	if ns != "" {
		return r.Namespace(ns).Delete(ctx, name, opts)
	}
	return r.Delete(ctx, name, opts)
}

func toFinalizerNames(in []string) []corev1.FinalizerName {
	out := make([]corev1.FinalizerName, 0, len(in))
	for _, f := range in {
		out = append(out, corev1.FinalizerName(f))
	}
	return out
}

// finalizersMergePatch builds a JSON merge patch that replaces metadata.finalizers
// with the filtered set and pins resourceVersion as an optimistic-concurrency
// precondition (a mismatch yields a 409 from the API server).
func finalizersMergePatch(newFinalizers []string, rv string) ([]byte, error) {
	if newFinalizers == nil {
		newFinalizers = []string{}
	}
	body := map[string]any{
		"metadata": map[string]any{
			"finalizers":      newFinalizers,
			"resourceVersion": rv,
		},
	}
	return json.Marshal(body)
}
