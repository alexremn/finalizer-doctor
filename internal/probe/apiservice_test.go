package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func apiservice(name, group string, hasService bool, available string) unstructured.Unstructured {
	spec := map[string]any{"group": group}
	if hasService {
		spec["service"] = map[string]any{"name": "svc", "namespace": "ns"}
	}
	o := unstructured.Unstructured{Object: map[string]any{
		"spec": spec,
		"status": map[string]any{"conditions": []any{
			map[string]any{"type": "Available", "status": available, "reason": "ServiceNotFound"},
		}},
	}}
	o.SetName(name)
	return o
}

func TestAPIServiceUnavailableAggregatedIsDeadHard(t *testing.T) {
	snap := model.Snapshot{
		RawAPIServices: []unstructured.Unstructured{apiservice("v1beta1.metrics.example.com", "metrics.example.com", true, "False")},
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadOK},
	}
	owner := model.OwnerCandidate{Finalizer: "x", Kind: "APIService", Group: "metrics.example.com"}
	ev := probeAPIService(owner, snap)
	require.NotNil(t, ev)
	assert.Equal(t, model.ClassDeadHard, ev.Class)
	assert.Contains(t, ev.Observed, "ServiceNotFound")
}

func TestAPIServiceUnavailableLocalIsNotDead(t *testing.T) {
	// Local APIService (no spec.service): never inferred dead from these reasons.
	snap := model.Snapshot{
		RawAPIServices: []unstructured.Unstructured{apiservice("v1.apps", "apps", false, "False")},
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadOK},
	}
	owner := model.OwnerCandidate{Finalizer: "x", Kind: "APIService", Group: "apps"}
	ev := probeAPIService(owner, snap)
	assert.Nil(t, ev, "local APIService must not produce a dead signal")
}

func TestAPIServiceUnreadableIsUnreadable(t *testing.T) {
	snap := model.Snapshot{SourceStatus: map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadUnreadable}}
	owner := model.OwnerCandidate{Kind: "APIService", Group: "metrics.example.com"}
	ev := probeAPIService(owner, snap)
	require.NotNil(t, ev)
	assert.Equal(t, model.ClassUnreadable, ev.Class)
}
