package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerdictSafeToClear(t *testing.T) {
	assert.True(t, Verdict{State: StateDead}.IsSafeToClear())
	assert.False(t, Verdict{State: StateSlow}.IsSafeToClear())
	assert.False(t, Verdict{State: StateUnknown}.IsSafeToClear())
}
