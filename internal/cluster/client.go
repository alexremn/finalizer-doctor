// Package cluster is the only package allowed to touch the Kubernetes API.
// Everything else consumes model types via this interface.
package cluster

import (
	"context"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ClusterClient is the narrow boundary over client-go. Implementations return
// plain API objects; all interpretation happens in pure stages.
type ClusterClient interface {
	// Discovery & read.
	ServerPreferredResources(ctx context.Context) ([]*metav1.APIResourceList, error)
	List(ctx context.Context, gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error)
	Get(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error)

	// Evidence sources (each returns its own typed list; see snapshot.Build).
	APIServices(ctx context.Context) ([]unstructured.Unstructured, error)
	CRDs(ctx context.Context) ([]unstructured.Unstructured, error)
	ValidatingWebhooks(ctx context.Context) ([]unstructured.Unstructured, error)
	MutatingWebhooks(ctx context.Context) ([]unstructured.Unstructured, error)
	Deployments(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	StatefulSets(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	DaemonSets(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	Pods(ctx context.Context, namespace string) ([]unstructured.Unstructured, error)
	EndpointSlices(ctx context.Context, namespace, serviceName string) ([]unstructured.Unstructured, error)

	// Authorization preflight.
	Can(ctx context.Context, attrs authv1.ResourceAttributes) (bool, error)

	// Mutations (only invoked under --apply, after the gate).
	PatchFinalizers(ctx context.Context, gvr schema.GroupVersionResource, namespace, name string, newFinalizers []string, resourceVersion string) error
	FinalizeNamespace(ctx context.Context, name string, newSpecFinalizers []string, resourceVersion string) error
	Delete(ctx context.Context, gvr schema.GroupVersionResource, namespace, name, resourceVersion string) error
}
