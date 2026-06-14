// Package webhook detects admission webhooks that would reject the mutating
// request finalizer-doctor is about to send (safety-model.md §6, verdict-engine
// §9.1). A failurePolicy=Fail webhook backed by a dead service is a refusal gate.
package webhook

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

const serviceNameLabel = "kubernetes.io/service-name"

// Blocks reports whether a failurePolicy=Fail webhook with a dead backing service
// would intercept an UPDATE/PATCH to ref, and a human-readable reason. It is
// conservative against false refusals: if a backing service's readiness cannot be
// determined, it does not treat the webhook as a blocker.
func Blocks(snap model.Snapshot, ref model.ResourceRef) (bool, string) {
	configs := make([]unstructured.Unstructured, 0, len(snap.RawValidating)+len(snap.RawMutating))
	configs = append(configs, snap.RawValidating...)
	configs = append(configs, snap.RawMutating...)

	for _, cfg := range configs {
		whs, _, _ := unstructured.NestedSlice(cfg.Object, "webhooks")
		for _, w := range whs {
			m, ok := w.(map[string]any)
			if !ok {
				continue
			}
			if fp, _, _ := unstructured.NestedString(m, "failurePolicy"); fp == "Ignore" {
				continue // default is Fail; only an explicit Ignore is non-blocking
			}
			if !rulesMatch(m, ref) {
				continue
			}
			ns, name, ok := serviceRef(m)
			if !ok {
				continue // URL-backed webhook: backing readiness not assessable here
			}
			if serviceDeadByEndpoints(snap, ns, name) {
				whName, _, _ := unstructured.NestedString(m, "name")
				return true, fmt.Sprintf("webhook %q (failurePolicy=Fail) backed by dead service %s/%s would reject the request", whName, ns, name)
			}
		}
	}
	return false, ""
}

func rulesMatch(webhook map[string]any, ref model.ResourceRef) bool {
	rules, _, _ := unstructured.NestedSlice(webhook, "rules")
	for _, r := range rules {
		rm, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if !sliceMatches(rm, "operations", "UPDATE") {
			continue
		}
		if !sliceMatches(rm, "apiGroups", ref.GVR.Group) {
			continue
		}
		if !sliceMatches(rm, "resources", ref.GVR.Resource) {
			continue
		}
		return true
	}
	return false
}

// sliceMatches reports whether the named string slice contains "*" or want.
func sliceMatches(rule map[string]any, field, want string) bool {
	vals, _, _ := unstructured.NestedStringSlice(rule, field)
	for _, v := range vals {
		if v == "*" || v == want {
			return true
		}
	}
	return false
}

func serviceRef(webhook map[string]any) (namespace, name string, ok bool) {
	name, _, _ = unstructured.NestedString(webhook, "clientConfig", "service", "name")
	namespace, _, _ = unstructured.NestedString(webhook, "clientConfig", "service", "namespace")
	return namespace, name, name != "" && namespace != ""
}

// serviceDeadByEndpoints returns true only when EndpointSlices for the service
// exist but none have a ready endpoint. Absence of slices is treated as
// indeterminate (not a blocker) to avoid false refusals.
func serviceDeadByEndpoints(snap model.Snapshot, ns, name string) bool {
	if !snap.Readable(model.SourceEndpointSlices) {
		return false
	}
	var sawSlice, sawReady bool
	for _, es := range snap.RawEndpointSlices {
		if es.GetNamespace() != ns || es.GetLabels()[serviceNameLabel] != name {
			continue
		}
		sawSlice = true
		if endpointSliceHasReady(es) {
			sawReady = true
		}
	}
	return sawSlice && !sawReady
}

func endpointSliceHasReady(es unstructured.Unstructured) bool {
	eps, _, _ := unstructured.NestedSlice(es.Object, "endpoints")
	for _, e := range eps {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if ready, found, _ := unstructured.NestedBool(em, "conditions", "ready"); found && ready {
			return true
		}
	}
	return false
}
