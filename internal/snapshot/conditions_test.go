package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestExtractNamespaceFields(t *testing.T) {
	o := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Namespace",
		"metadata": map[string]any{"name": "foo", "finalizers": []any{"op.example.com/x"}},
		"spec":     map[string]any{"finalizers": []any{"kubernetes"}},
		"status": map[string]any{"conditions": []any{
			map[string]any{"type": "NamespaceDeletionDiscoveryFailure", "status": "True", "reason": "DiscoveryFailed", "message": "metrics.example.com"},
		}},
	}}

	spec := extractStringSlice(o.Object, "spec", "finalizers")
	assert.Equal(t, []string{"kubernetes"}, spec)

	conds := extractConditions(o)
	require.Len(t, conds, 1)
	assert.Equal(t, "NamespaceDeletionDiscoveryFailure", conds[0].Type)
	assert.Equal(t, "True", conds[0].Status)
	assert.Equal(t, "metrics.example.com", conds[0].Message)
}
