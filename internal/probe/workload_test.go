package probe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func deploy(name string, ready int64) unstructured.Unstructured {
	o := unstructured.Unstructured{Object: map[string]any{
		"status": map[string]any{"readyReplicas": ready},
	}}
	o.SetName(name)
	return o
}

func TestWorkloadZeroReadyNoPodsIsDeadHard(t *testing.T) {
	snap := model.Snapshot{
		RawDeployments: []unstructured.Unstructured{deploy("op-controller-manager", 0)},
		RawPods:        nil,
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceWorkloads: model.ReadOK, model.SourcePods: model.ReadOK},
	}
	owner := model.OwnerCandidate{MatchReason: "name heuristic op-controller-manager"}
	evs := probeWorkload(owner, snap)
	assert.Equal(t, model.ClassDeadHard, evs[0].Class)
}

func TestWorkloadReadyIsLive(t *testing.T) {
	snap := model.Snapshot{
		RawDeployments: []unstructured.Unstructured{deploy("op-controller-manager", 2)},
		SourceStatus:   map[model.Source]model.ReadStatus{model.SourceWorkloads: model.ReadOK, model.SourcePods: model.ReadOK},
	}
	owner := model.OwnerCandidate{MatchReason: "name heuristic op-controller-manager"}
	evs := probeWorkload(owner, snap)
	assert.Equal(t, model.ClassLive, evs[0].Class)
}

func TestWorkloadUnreadableIsUnreadable(t *testing.T) {
	snap := model.Snapshot{SourceStatus: map[model.Source]model.ReadStatus{model.SourceWorkloads: model.ReadUnreadable}}
	evs := probeWorkload(model.OwnerCandidate{}, snap)
	assert.Equal(t, model.ClassUnreadable, evs[0].Class)
	assert.Equal(t, model.SourceWorkloads, evs[0].Source)
}
