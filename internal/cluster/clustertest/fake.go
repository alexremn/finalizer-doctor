// Package clustertest provides a hand-written fake ClusterClient for unit tests.
package clustertest

import (
	"context"
	"fmt"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ListKey returns the map key used in Fake.Lists for a given GVR and namespace.
// Using this helper avoids raw string concatenation at every call site.
func ListKey(gvr schema.GroupVersionResource, ns string) string {
	return gvr.String() + "/" + ns
}

// GetKey returns the map key used in Fake.Gets for a given GVR, namespace, and name.
func GetKey(gvr schema.GroupVersionResource, ns, name string) string {
	return gvr.String() + "/" + ns + "/" + name
}

// AllowKey returns the map key used in Fake.Allowed for a given verb, resource, and subresource.
func AllowKey(verb, resource, subresource string) string {
	return fmt.Sprintf("%s:%s:%s", verb, resource, subresource)
}

// Fake implements cluster.ClusterClient from in-memory fixtures.
type Fake struct {
	Preferred       []*metav1.APIResourceList
	Lists           map[string]*unstructured.UnstructuredList // key: ListKey(gvr, namespace)
	Gets            map[string]*unstructured.Unstructured     // key: GetKey(gvr, namespace, name)
	GetFn           func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error)
	APIServiceObjs  []unstructured.Unstructured
	CRDObjs         []unstructured.Unstructured
	ValidatingObjs  []unstructured.Unstructured
	MutatingObjs    []unstructured.Unstructured
	DeploymentObjs  []unstructured.Unstructured
	StatefulSetObjs []unstructured.Unstructured
	DaemonSetObjs   []unstructured.Unstructured
	PodObjs         []unstructured.Unstructured
	EndpointObjs    []unstructured.Unstructured
	Allowed         map[string]bool  // key: AllowKey(verb, resource, subresource)
	Errs            map[string]error // inject errors by method name
	Mutations       []string         // ordered record of mutating calls
}

func (f *Fake) err(name string) error { return f.Errs[name] }

func (f *Fake) ServerPreferredResources(_ context.Context) ([]*metav1.APIResourceList, error) {
	if e := f.err("ServerPreferredResources"); e != nil {
		return nil, e
	}
	return f.Preferred, nil
}
func (f *Fake) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	if e := f.err("List"); e != nil {
		return nil, e
	}
	l := f.Lists[ListKey(gvr, ns)]
	if l == nil {
		l = &unstructured.UnstructuredList{}
	}
	return l, nil
}
func (f *Fake) Get(ctx context.Context, gvr schema.GroupVersionResource, ns, name string) (*unstructured.Unstructured, error) {
	if f.GetFn != nil {
		return f.GetFn(ctx, gvr, ns, name)
	}
	if obj, ok := f.Gets[GetKey(gvr, ns, name)]; ok {
		return obj, nil
	}
	if e := f.err("Get"); e != nil {
		return nil, e
	}
	return nil, fmt.Errorf("clustertest.Fake: no object registered for Get(%s, %q, %q)", gvr.String(), ns, name)
}
func (f *Fake) APIServices(_ context.Context) ([]unstructured.Unstructured, error) {
	if e := f.err("APIServices"); e != nil {
		return nil, e
	}
	return f.APIServiceObjs, nil
}
func (f *Fake) CRDs(_ context.Context) ([]unstructured.Unstructured, error) {
	if e := f.err("CRDs"); e != nil {
		return nil, e
	}
	return f.CRDObjs, nil
}
func (f *Fake) ValidatingWebhooks(_ context.Context) ([]unstructured.Unstructured, error) {
	if e := f.err("ValidatingWebhooks"); e != nil {
		return nil, e
	}
	return f.ValidatingObjs, nil
}
func (f *Fake) MutatingWebhooks(_ context.Context) ([]unstructured.Unstructured, error) {
	if e := f.err("MutatingWebhooks"); e != nil {
		return nil, e
	}
	return f.MutatingObjs, nil
}
func (f *Fake) Deployments(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	if e := f.err("Deployments"); e != nil {
		return nil, e
	}
	return f.DeploymentObjs, nil
}
func (f *Fake) StatefulSets(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	if e := f.err("StatefulSets"); e != nil {
		return nil, e
	}
	return f.StatefulSetObjs, nil
}
func (f *Fake) DaemonSets(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	if e := f.err("DaemonSets"); e != nil {
		return nil, e
	}
	return f.DaemonSetObjs, nil
}
func (f *Fake) Pods(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	if e := f.err("Pods"); e != nil {
		return nil, e
	}
	return f.PodObjs, nil
}
func (f *Fake) EndpointSlices(_ context.Context, _, _ string) ([]unstructured.Unstructured, error) {
	if e := f.err("EndpointSlices"); e != nil {
		return nil, e
	}
	return f.EndpointObjs, nil
}
func (f *Fake) Can(_ context.Context, a authv1.ResourceAttributes) (bool, error) {
	if e := f.err("Can"); e != nil {
		return false, e
	}
	if f.Allowed == nil {
		return true, nil // permissive default: set Allowed (non-nil) to test denials
	}
	return f.Allowed[AllowKey(a.Verb, a.Resource, a.Subresource)], nil
}
func (f *Fake) PatchFinalizers(_ context.Context, gvr schema.GroupVersionResource, ns, name string, _ []string, rv string) error {
	if e := f.err("PatchFinalizers"); e != nil {
		return e
	}
	f.Mutations = append(f.Mutations, fmt.Sprintf("PatchFinalizers %s/%s/%s rv=%s", gvr.Resource, ns, name, rv))
	return nil
}
func (f *Fake) FinalizeNamespace(_ context.Context, name string, _ []string, rv string) error {
	if e := f.err("FinalizeNamespace"); e != nil {
		return e
	}
	f.Mutations = append(f.Mutations, fmt.Sprintf("FinalizeNamespace %s rv=%s", name, rv))
	return nil
}
func (f *Fake) Delete(_ context.Context, gvr schema.GroupVersionResource, ns, name, rv string) error {
	if e := f.err("Delete"); e != nil {
		return e
	}
	f.Mutations = append(f.Mutations, fmt.Sprintf("Delete %s/%s/%s rv=%s", gvr.Resource, ns, name, rv))
	return nil
}
