// Package snapshot converts a ClusterClient's reads into an immutable
// model.Snapshot. It is the only consumer of ClusterClient besides apply.
package snapshot

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/cluster"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func extractStringSlice(obj map[string]any, fields ...string) []string {
	v, found, err := unstructured.NestedStringSlice(obj, fields...)
	if err != nil || !found {
		return nil
	}
	return v
}

func extractConditions(o *unstructured.Unstructured) []model.Condition {
	raw, found, err := unstructured.NestedSlice(o.Object, "status", "conditions")
	if err != nil || !found {
		return nil
	}
	out := make([]model.Condition, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		c := model.Condition{}
		c.Type, _, _ = unstructured.NestedString(m, "type")
		c.Status, _, _ = unstructured.NestedString(m, "status")
		c.Reason, _, _ = unstructured.NestedString(m, "reason")
		c.Message, _, _ = unstructured.NestedString(m, "message")
		out = append(out, c)
	}
	return out
}

// Build reads every evidence source and assembles an immutable Snapshot.
// A read error on any source marks that source Unreadable (never fatal) so the
// verdict engine can apply its cannot-probe-≠-dead veto.
func Build(ctx context.Context, c cluster.ClusterClient, refs []model.ResourceRef, now time.Time) (model.Snapshot, error) {
	s := model.Snapshot{Now: now, SourceStatus: map[model.Source]model.ReadStatus{}}

	for _, ref := range refs {
		o, err := c.Get(ctx, ref.GVR, ref.Namespace, ref.Name)
		if err != nil {
			s.SourceStatus[model.SourceTargets] = model.ReadUnreadable
			continue
		}
		s.Targets = append(s.Targets, toStuckObject(ref, o))
	}
	if _, ok := s.SourceStatus[model.SourceTargets]; !ok {
		s.SourceStatus[model.SourceTargets] = model.ReadOK
	}

	// Evidence sources: store raw unstructured; probe interprets them.
	s.RawAPIServices = readSource(ctx, &s, model.SourceAPIServices, c.APIServices)
	s.RawCRDs = readSource(ctx, &s, model.SourceCRDs, c.CRDs)
	s.RawValidating = readSource(ctx, &s, model.SourceWebhooks, c.ValidatingWebhooks)
	s.RawMutating = readSourceSilent(ctx, c.MutatingWebhooks) // webhook status already tracked above
	s.RawDeployments = readSource(ctx, &s, model.SourceWorkloads, func(ctx context.Context) ([]unstructured.Unstructured, error) {
		return c.Deployments(ctx, "")
	})
	s.RawPods = readSource(ctx, &s, model.SourcePods, func(ctx context.Context) ([]unstructured.Unstructured, error) {
		return c.Pods(ctx, "")
	})
	s.RawEndpointSlices = readSource(ctx, &s, model.SourceEndpointSlices, func(ctx context.Context) ([]unstructured.Unstructured, error) {
		return c.EndpointSlices(ctx, "", "")
	})
	return s, nil
}

func readSource(ctx context.Context, s *model.Snapshot, src model.Source, fn func(context.Context) ([]unstructured.Unstructured, error)) []unstructured.Unstructured {
	objs, err := fn(ctx)
	if err != nil {
		s.SourceStatus[src] = model.ReadUnreadable
		return nil
	}
	s.SourceStatus[src] = model.ReadOK
	return objs
}

func readSourceSilent(ctx context.Context, fn func(context.Context) ([]unstructured.Unstructured, error)) []unstructured.Unstructured {
	objs, _ := fn(ctx)
	return objs
}

func toStuckObject(ref model.ResourceRef, o *unstructured.Unstructured) model.StuckObject {
	so := model.StuckObject{
		Ref:                ref,
		MetadataFinalizers: o.GetFinalizers(),
		SpecFinalizers:     extractStringSlice(o.Object, "spec", "finalizers"),
		ResourceVersion:    o.GetResourceVersion(),
	}
	if dt := o.GetDeletionTimestamp(); dt != nil {
		t := dt.Time
		so.DeletionTimestamp = &t
	}
	if ref.GVR.Resource == "namespaces" {
		so.NamespaceConditions = extractConditions(o)
	}
	return so
}
