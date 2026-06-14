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
		SourceStatus: map[model.Source]model.ReadStatus{
			model.SourceAPIServices: model.ReadOK,
			model.SourceWorkloads:   model.ReadOK,
			model.SourcePods:        model.ReadOK,
		},
	}
	owner := model.OwnerCandidate{Finalizer: "example.com/x", Kind: "APIService", Group: "example.com"}
	ev := For(owner, snap)
	require.NotEmpty(t, ev)
	assert.Equal(t, model.ClassDeadHard, ev[0].Class)
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

func TestProbeWorkloadAbsentNonWorkloadKindIsNil(t *testing.T) {
	snap := model.Snapshot{SourceStatus: map[model.Source]model.ReadStatus{model.SourceWorkloads: model.ReadOK}}
	assert.Nil(t, probeWorkload(model.OwnerCandidate{Kind: "APIService"}, snap))
}

func TestProbeWorkloadZeroReadyButPodsExistIsLive(t *testing.T) {
	pod := unstructured.Unstructured{}
	pod.SetName("op-pod")
	snap := model.Snapshot{
		RawDeployments: []unstructured.Unstructured{deploy("op-controller-manager", 0)},
		RawPods:        []unstructured.Unstructured{pod},
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceWorkloads: model.ReadOK, model.SourcePods: model.ReadOK},
	}
	evs := probeWorkload(model.OwnerCandidate{MatchReason: "op-controller-manager"}, snap)
	require.NotEmpty(t, evs)
	assert.Equal(t, model.ClassLive, evs[0].Class)
}
