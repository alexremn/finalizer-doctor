package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
)

func failWebhookConfig() unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{
		"webhooks": []any{map[string]any{
			"name":          "v.example.com",
			"failurePolicy": "Fail",
			"rules": []any{map[string]any{
				"operations": []any{"UPDATE"},
				"apiGroups":  []any{"example.com"},
				"resources":  []any{"widgets"},
			}},
			"clientConfig": map[string]any{"service": map[string]any{"name": "hook", "namespace": "kube-system"}},
		}},
	}}
}

func notReadyEndpointSlice() unstructured.Unstructured {
	es := unstructured.Unstructured{Object: map[string]any{
		"endpoints": []any{map[string]any{"conditions": map[string]any{"ready": false}}},
	}}
	es.SetNamespace("kube-system")
	es.SetLabels(map[string]string{"kubernetes.io/service-name": "hook"})
	return es
}

func blockedFake() *clustertest.Fake {
	cr := stuckCR()
	f := &clustertest.Fake{
		APIServiceObjs: []unstructured.Unstructured{deadAPIService()},
		ValidatingObjs: []unstructured.Unstructured{failWebhookConfig()},
		EndpointObjs:   []unstructured.Unstructured{notReadyEndpointSlice()},
	}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		c := cr
		return &c, nil
	}
	return f
}

func TestRunDryRunShowsWebhookBlocked(t *testing.T) {
	out, code, err := Run(context.Background(), blockedFake(), Options{Target: "widgets.example.com/w1", Namespace: "team-a"})
	require.NoError(t, err)
	assert.Equal(t, 2, code)
	assert.Contains(t, out, "blocked")
	assert.NotContains(t, out, "to apply:", "no apply hint when blocked")
}

func TestRunApplyWebhookBlockedRefuses(t *testing.T) {
	out, code, err := Run(context.Background(), blockedFake(), Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: "anytoken0000"})
	require.NoError(t, err)
	assert.Equal(t, 3, code)
	assert.Contains(t, out, "refused")
	assert.Empty(t, blockedFake().Mutations)
}
