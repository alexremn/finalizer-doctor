// Package discover resolves CLI targets and scans the cluster for stuck objects.
package discover

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

// Target is a partially-resolved reference parsed from the CLI. The resource is
// a discovery-time string (singular/plural/short); snapshot.Build resolves it to
// a GVR via discovery. "ns" is normalized to "namespaces".
type Target struct {
	Group     string
	Resource  string
	Namespace string
	Name      string
}

// ParseTarget parses "<resource>[.<group>]/<name>", with "ns" as an alias for
// "namespaces". defaultNS supplies the namespace for namespaced resources.
func ParseTarget(in, defaultNS string) (Target, error) {
	slash := strings.IndexByte(in, '/')
	if slash <= 0 || slash == len(in)-1 {
		return Target{}, fmt.Errorf("invalid target %q: want <resource>[.<group>]/<name>", in)
	}
	left, name := in[:slash], in[slash+1:]
	if name == "" {
		return Target{}, fmt.Errorf("invalid target %q: empty name", in)
	}

	res, group := left, ""
	if dot := strings.IndexByte(left, '.'); dot >= 0 {
		res, group = left[:dot], left[dot+1:]
	}
	if res == "ns" {
		res = "namespaces"
	}

	t := Target{Group: group, Resource: res, Name: name}
	if res != "namespaces" { // namespaces are cluster-scoped
		t.Namespace = defaultNS
	}
	return t, nil
}

// Scan lists every preferred, listable resource and returns refs for objects
// that have a deletionTimestamp and a non-empty finalizer set.
func Scan(ctx context.Context, c cluster.ClusterClient) ([]model.ResourceRef, error) {
	lists, err := c.ServerPreferredResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}
	var refs []model.ResourceRef
	for _, rl := range lists {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rl.APIResources {
			if !canList(r.Verbs) || strings.Contains(r.Name, "/") {
				continue
			}
			gvr := gv.WithResource(r.Name)
			objs, err := c.List(ctx, gvr, "")
			if err != nil {
				continue // unreadable group: skip, do not fail the whole scan
			}
			for i := range objs.Items {
				o := objs.Items[i]
				if o.GetDeletionTimestamp() != nil && len(o.GetFinalizers()) > 0 {
					refs = append(refs, model.ResourceRef{GVR: gvr, Namespace: o.GetNamespace(), Name: o.GetName()})
				}
			}
		}
	}
	return refs, nil
}

func canList(verbs []string) bool {
	for _, v := range verbs {
		if v == "list" {
			return true
		}
	}
	return false
}

// NamespaceObjects lists every namespaced, listable resource in the namespace and
// returns minimal StuckObjects (ref + kind + ownerRefs) as orphan candidates.
// A failed list on any group is skipped, never fatal.
func NamespaceObjects(ctx context.Context, c cluster.ClusterClient, namespace string) ([]model.StuckObject, error) {
	lists, err := c.ServerPreferredResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovery failed: %w", err)
	}
	var out []model.StuckObject
	for _, rl := range lists {
		gv, err := schema.ParseGroupVersion(rl.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range rl.APIResources {
			if !r.Namespaced || !canList(r.Verbs) || strings.Contains(r.Name, "/") {
				continue
			}
			gvr := gv.WithResource(r.Name)
			objs, err := c.List(ctx, gvr, namespace)
			if err != nil {
				continue
			}
			for i := range objs.Items {
				o := objs.Items[i]
				out = append(out, model.StuckObject{
					Ref:        model.ResourceRef{GVR: gvr, Namespace: o.GetNamespace(), Name: o.GetName()},
					Kind:       o.GetKind(),
					APIVersion: o.GetAPIVersion(),
					OwnerRefs:  ownerRefsOf(&o),
				})
			}
		}
	}
	return out, nil
}

func ownerRefsOf(o *unstructured.Unstructured) []model.OwnerRef {
	refs := o.GetOwnerReferences()
	if len(refs) == 0 {
		return nil
	}
	out := make([]model.OwnerRef, 0, len(refs))
	for _, r := range refs {
		out = append(out, model.OwnerRef{APIVersion: r.APIVersion, Kind: r.Kind, Name: r.Name})
	}
	return out
}
