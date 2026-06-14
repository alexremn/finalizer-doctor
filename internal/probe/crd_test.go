package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestCRDAbsentForCRDOwnerIsDeadHard(t *testing.T) {
	snap := model.Snapshot{
		RawCRDs:      nil,
		SourceStatus: map[model.Source]model.ReadStatus{model.SourceCRDs: model.ReadOK},
	}
	owner := model.OwnerCandidate{Kind: "CRD", Group: "example.com"}
	ev := probeCRD(owner, snap)
	require.NotNil(t, ev)
	assert.Equal(t, model.ClassDeadHard, ev.Class)
}

func TestCRDPresentNoSignal(t *testing.T) {
	crd := unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"group": "example.com"}}}
	snap := model.Snapshot{
		RawCRDs:      []unstructured.Unstructured{crd},
		SourceStatus: map[model.Source]model.ReadStatus{model.SourceCRDs: model.ReadOK},
	}
	assert.Nil(t, probeCRD(model.OwnerCandidate{Kind: "CRD", Group: "example.com"}, snap))
}
