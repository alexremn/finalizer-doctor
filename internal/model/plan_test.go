package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanHasActions(t *testing.T) {
	p := Plan{Actions: []Action{{Kind: ActionClearFinalizer}}}
	assert.True(t, p.HasActions())
	assert.False(t, Plan{}.HasActions())
}
