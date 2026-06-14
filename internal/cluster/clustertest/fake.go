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

// Fake implements cluster.ClusterClient from in-memory fixtures.
type Fake struct {
	Preferred       []*metav1.APIResourceList
	Lists           map[string]*unstructured.UnstructuredList // key: gvr.String()+"/"+namespace
	APIServiceObjs  []unstructured.Unstructured
	CRDObjs         []unstructured.Unstructured
	ValidatingObjs  []unstructured.Unstructured
	MutatingObjs    []unstructured.Unstructured
	DeploymentObjs  []unstructured.Unstructured
	StatefulSetObjs []unstructured.Unstructured
	DaemonSetObjs   []unstructured.Unstructured
	PodObjs         []unstructured.Unstructured
	EndpointObjs    []unstructured.Unstructured
	Allowed         map[string]bool  // key: verb:resource:subresource
	Errs            map[string]error // inject errors by method name
	Mutations       []string         // ordered record of mutating calls
}

func (f *Fake) err(name string) error { return f.Errs[name] }

func (f *Fake) ServerPreferredResources(_ context.Context) ([]*metav1.APIResourceList, error) {
	return f.Preferred, f.err("ServerPreferredResources")
}
func (f *Fake) List(_ context.Context, gvr schema.GroupVersionResource, ns string) (*unstructured.UnstructuredList, error) {
	if e := f.err("List"); e != nil {
		return nil, e
	}
	l := f.Lists[gvr.String()+"/"+ns]
	if l == nil {
		l = &unstructured.UnstructuredList{}
	}
	return l, nil
}
func (f *Fake) Get(_ context.Context, _ schema.GroupVersionResource, _, _ string) (*unstructured.Unstructured, error) {
	return nil, f.err("Get")
}
func (f *Fake) APIServices(_ context.Context) ([]unstructured.Unstructured, error) {
	return f.APIServiceObjs, f.err("APIServices")
}
func (f *Fake) CRDs(_ context.Context) ([]unstructured.Unstructured, error) {
	return f.CRDObjs, f.err("CRDs")
}
func (f *Fake) ValidatingWebhooks(_ context.Context) ([]unstructured.Unstructured, error) {
	return f.ValidatingObjs, f.err("ValidatingWebhooks")
}
func (f *Fake) MutatingWebhooks(_ context.Context) ([]unstructured.Unstructured, error) {
	return f.MutatingObjs, f.err("MutatingWebhooks")
}
func (f *Fake) Deployments(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.DeploymentObjs, f.err("Deployments")
}
func (f *Fake) StatefulSets(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.StatefulSetObjs, f.err("StatefulSets")
}
func (f *Fake) DaemonSets(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.DaemonSetObjs, f.err("DaemonSets")
}
func (f *Fake) Pods(_ context.Context, _ string) ([]unstructured.Unstructured, error) {
	return f.PodObjs, f.err("Pods")
}
func (f *Fake) EndpointSlices(_ context.Context, _, _ string) ([]unstructured.Unstructured, error) {
	return f.EndpointObjs, f.err("EndpointSlices")
}
func (f *Fake) Can(_ context.Context, a authv1.ResourceAttributes) (bool, error) {
	if e := f.err("Can"); e != nil {
		return false, e
	}
	return f.Allowed[fmt.Sprintf("%s:%s:%s", a.Verb, a.Resource, a.Subresource)], nil
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
