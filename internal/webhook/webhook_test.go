package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func whConfig(failurePolicy, group, resource, svcNS, svcName string) unstructured.Unstructured {
	wh := map[string]any{
		"name":          "v.example.com",
		"failurePolicy": failurePolicy,
		"rules": []any{map[string]any{
			"operations": []any{"UPDATE"},
			"apiGroups":  []any{group},
			"resources":  []any{resource},
		}},
		"clientConfig": map[string]any{"service": map[string]any{"name": svcName, "namespace": svcNS}},
	}
	cfg := unstructured.Unstructured{Object: map[string]any{"webhooks": []any{wh}}}
	return cfg
}

func endpointSlice(ns, svc string, ready bool) unstructured.Unstructured {
	es := unstructured.Unstructured{Object: map[string]any{
		"endpoints": []any{map[string]any{"conditions": map[string]any{"ready": ready}}},
	}}
	es.SetNamespace(ns)
	es.SetLabels(map[string]string{serviceNameLabel: svc})
	return es
}

func widgetRef() model.ResourceRef {
	return model.ResourceRef{GVR: schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}, Name: "w1"}
}

func TestBlocksWhenFailWebhookHasDeadBacking(t *testing.T) {
	snap := model.Snapshot{
		RawValidating:     []unstructured.Unstructured{whConfig("Fail", "example.com", "widgets", "kube-system", "hook")},
		RawEndpointSlices: []unstructured.Unstructured{endpointSlice("kube-system", "hook", false)},
		SourceStatus:      map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadOK},
	}
	blocked, reason := Blocks(snap, widgetRef())
	assert.True(t, blocked)
	assert.Contains(t, reason, "dead service kube-system/hook")
}

func TestNoBlockWhenWebhookBackingReady(t *testing.T) {
	snap := model.Snapshot{
		RawValidating:     []unstructured.Unstructured{whConfig("Fail", "example.com", "widgets", "kube-system", "hook")},
		RawEndpointSlices: []unstructured.Unstructured{endpointSlice("kube-system", "hook", true)},
		SourceStatus:      map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadOK},
	}
	blocked, _ := Blocks(snap, widgetRef())
	assert.False(t, blocked)
}

func TestNoBlockWhenFailurePolicyIgnore(t *testing.T) {
	snap := model.Snapshot{
		RawValidating:     []unstructured.Unstructured{whConfig("Ignore", "example.com", "widgets", "kube-system", "hook")},
		RawEndpointSlices: []unstructured.Unstructured{endpointSlice("kube-system", "hook", false)},
		SourceStatus:      map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadOK},
	}
	blocked, _ := Blocks(snap, widgetRef())
	assert.False(t, blocked)
}

func TestNoBlockWhenRulesDoNotMatch(t *testing.T) {
	snap := model.Snapshot{
		RawValidating:     []unstructured.Unstructured{whConfig("Fail", "other.io", "things", "kube-system", "hook")},
		RawEndpointSlices: []unstructured.Unstructured{endpointSlice("kube-system", "hook", false)},
		SourceStatus:      map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadOK},
	}
	blocked, _ := Blocks(snap, widgetRef())
	assert.False(t, blocked)
}

func TestWildcardRuleMatches(t *testing.T) {
	snap := model.Snapshot{
		RawMutating:       []unstructured.Unstructured{whConfig("Fail", "*", "*", "kube-system", "hook")},
		RawEndpointSlices: []unstructured.Unstructured{endpointSlice("kube-system", "hook", false)},
		SourceStatus:      map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadOK},
	}
	blocked, _ := Blocks(snap, widgetRef())
	assert.True(t, blocked)
}

func TestNoBlockWhenEndpointsUnreadable(t *testing.T) {
	snap := model.Snapshot{
		RawValidating: []unstructured.Unstructured{whConfig("Fail", "example.com", "widgets", "kube-system", "hook")},
		SourceStatus:  map[model.Source]model.ReadStatus{model.SourceEndpointSlices: model.ReadUnreadable},
	}
	blocked, _ := Blocks(snap, widgetRef())
	assert.False(t, blocked, "indeterminate backing must not cause a false refusal")
}
