// Package model holds the immutable domain types passed between pipeline
// stages. It must not import any Kubernetes client package.
package model

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ResourceRef uniquely identifies a Kubernetes object by GVR + namespace + name.
type ResourceRef struct {
	GVR       schema.GroupVersionResource
	Namespace string
	Name      string
}

// String renders a stable, human-readable reference.
func (r ResourceRef) String() string {
	res := r.GVR.Resource
	if r.GVR.Group != "" {
		res = res + "." + r.GVR.Group
	}
	if r.Namespace != "" {
		return strings.Join([]string{r.Namespace, res, r.Name}, "/")
	}
	return res + "/" + r.Name
}
