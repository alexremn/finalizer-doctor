package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestForAggregatesEvidence(t *testing.T) {
	snap := model.Snapshot{
		RawAPIServices: []unstructured.Unstructured{apiservice("v1.example.com", "example.com", true, "False")},
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadOK},
	}
	owner := model.OwnerCandidate{Finalizer: "example.com/x", Kind: "APIService", Group: "example.com"}
	ev := For(owner, snap)
	require.NotEmpty(t, ev)
	assert.Equal(t, model.ClassDeadHard, ev[0].Class)
}

func TestForBuiltinIsLive(t *testing.T) {
	// Built-in finalizers must not match arbitrary operator Deployments; they are
	// attributed to their core control-plane owner and reported live (-> SLOW).
	owner := model.OwnerCandidate{Finalizer: "kubernetes.io/pv-protection", Kind: "Builtin", MatchReason: "built-in: pv-protection-controller (KCM)"}
	ev := For(owner, model.Snapshot{})
	require.Len(t, ev, 1)
	assert.Equal(t, model.ClassLive, ev[0].Class)
}

func TestProbeAPIServiceEmptyGroupIsNil(t *testing.T) {
	assert.Nil(t, probeAPIService(model.OwnerCandidate{Kind: "APIService"}, model.Snapshot{}))
}

func TestProbeCRDUnreadable(t *testing.T) {
	snap := model.Snapshot{SourceStatus: map[model.Source]model.ReadStatus{model.SourceCRDs: model.ReadUnreadable}}
	ev := probeCRD(model.OwnerCandidate{Kind: "CRD", Group: "example.com"}, snap)
	require.NotNil(t, ev)
	assert.Equal(t, model.ClassUnreadable, ev.Class)
}
