package verdict

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func owner() model.OwnerCandidate { return model.OwnerCandidate{Finalizer: "f", Kind: "APIService"} }

func TestStrictUnknownOwnerIsUnknown(t *testing.T) {
	v := Strict{}.Verdict(model.OwnerCandidate{Kind: "Unknown"}, nil)
	assert.Equal(t, model.StateUnknown, v.State)
	assert.False(t, v.SafeToClear)
}

func TestStrictHardDeadNoLiveIsDead(t *testing.T) {
	ev := []model.Evidence{{Class: model.ClassDeadHard, Source: model.SourceAPIServices}}
	v := Strict{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateDead, v.State)
	assert.True(t, v.SafeToClear)
}

func TestStrictUnreadableLivenessSourceVetoesDead(t *testing.T) {
	// Hard dead signal present, but a liveness-relevant source is unreadable.
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Source: model.SourceAPIServices},
		{Class: model.ClassUnreadable, Source: model.SourceWorkloads},
	}
	v := Strict{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateUnknown, v.State, "unreadable liveness source must veto DEAD")
}

func TestStrictUnreadableNonLivenessSourceDoesNotVeto(t *testing.T) {
	// An unreadable NON-liveness source (APIServices) must not block DEAD.
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Source: model.SourceCRDs},
		{Class: model.ClassUnreadable, Source: model.SourceAPIServices},
	}
	v := Strict{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateDead, v.State)
}

func TestStrictLiveDowngradesToSlow(t *testing.T) {
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Source: model.SourceAPIServices},
		{Class: model.ClassLive, Source: model.SourceWorkloads},
	}
	v := Strict{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateSlow, v.State)
}

func TestStrictSoftOnlyIsUnknown(t *testing.T) {
	ev := []model.Evidence{{Class: model.ClassDeadSoft, Source: model.SourcePods}}
	v := Strict{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateUnknown, v.State, "time/soft alone never DEAD")
}
