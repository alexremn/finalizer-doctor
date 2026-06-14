package verdict

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestScoreSoftAloneNeverDead(t *testing.T) {
	ev := []model.Evidence{{Class: model.ClassDeadSoft, Weight: 15, Source: model.SourcePods}}
	v := Score{}.Verdict(owner(), ev)
	assert.NotEqual(t, model.StateDead, v.State, "no hard signal -> never DEAD")
}

func TestScoreSingleHardSignalBelowThresholdIsSlow(t *testing.T) {
	ev := []model.Evidence{{Class: model.ClassDeadHard, Weight: 40, Source: model.SourceAPIServices}}
	v := Score{}.Verdict(owner(), ev)
	require.NotNil(t, v.Score)
	assert.Equal(t, 40, *v.Score)
	assert.Equal(t, model.StateSlow, v.State, "single +40 hard signal < 60 threshold")
}

func TestScoreHardPlusCorroborationIsDead(t *testing.T) {
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Weight: 40, Source: model.SourceAPIServices},
		{Class: model.ClassDeadHard, Weight: 35, Source: model.SourceCRDs},
	}
	v := Score{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateDead, v.State)
	assert.True(t, v.SafeToClear)
}

func TestScoreLiveVetoes(t *testing.T) {
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Weight: 40, Source: model.SourceAPIServices},
		{Class: model.ClassDeadHard, Weight: 35, Source: model.SourceCRDs},
		{Class: model.ClassLive, Source: model.SourceWorkloads},
	}
	v := Score{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateSlow, v.State)
}

func TestScoreUnreadableLivenessVetoes(t *testing.T) {
	ev := []model.Evidence{
		{Class: model.ClassDeadHard, Weight: 40, Source: model.SourceAPIServices},
		{Class: model.ClassDeadHard, Weight: 35, Source: model.SourceCRDs},
		{Class: model.ClassUnreadable, Source: model.SourceWorkloads},
	}
	v := Score{}.Verdict(owner(), ev)
	assert.Equal(t, model.StateUnknown, v.State)
}
