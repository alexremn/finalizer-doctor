package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func crd(group string) unstructured.Unstructured {
	o := unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{"group": group},
	}}
	o.SetName("widgets." + group)
	return o
}

func TestMapBuiltin(t *testing.T) {
	c := Map("kubernetes", model.Snapshot{})
	assert.Equal(t, "Builtin", c.Kind)
	assert.Equal(t, "kubernetes", c.Finalizer)
}

func TestMapDomainToCRD(t *testing.T) {
	snap := model.Snapshot{RawCRDs: []unstructured.Unstructured{crd("example.com")}}
	c := Map("widgets.example.com/cleanup", snap)
	assert.Equal(t, "CRD", c.Kind)
	assert.Contains(t, c.MatchReason, "example.com")
	assert.Greater(t, c.Rank, 0)
}

func TestMapUnknown(t *testing.T) {
	c := Map("mystery.io/x", model.Snapshot{})
	assert.Equal(t, "Unknown", c.Kind)
	require.Equal(t, model.StateUnknown, ForceStateForUnknown(c))
}

func TestDomainOf(t *testing.T) {
	assert.Equal(t, "example.com", DomainOf("widgets.example.com/cleanup"))
	assert.Equal(t, "example.com", DomainOf("example.com/finalizer"))
	assert.Equal(t, "kubernetes", DomainOf("kubernetes"))
}
