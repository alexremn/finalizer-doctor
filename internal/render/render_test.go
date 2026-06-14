package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func sampleVerdict() model.Verdict {
	return model.Verdict{
		Finalizer: "kubernetes",
		Owner:     model.OwnerCandidate{Kind: "APIService", MatchReason: "APIService group metrics.example.com"},
		State:     model.StateDead,
		Evidence:  []model.Evidence{{Observed: "Available=False (ServiceNotFound)", Class: model.ClassDeadHard}},
	}
}

func TestHumanShowsVerdictAndTags(t *testing.T) {
	out := Human([]model.Verdict{sampleVerdict()}, model.Plan{})
	assert.Contains(t, out, "DEAD")
	assert.Contains(t, out, "[hard]")
	assert.Contains(t, out, "kubernetes")
}

func TestJSONIsValidAndStructured(t *testing.T) {
	out := JSON([]model.Verdict{sampleVerdict()}, model.Plan{})
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Contains(t, out, `"state": "DEAD"`)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "{"))
}
