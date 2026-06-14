package mapping

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltin(t *testing.T) {
	b, ok := builtin("kubernetes")
	assert.True(t, ok)
	assert.Equal(t, "Builtin", b.Kind)
	assert.True(t, b.RarelyDead)

	_, ok = builtin("example.com/cleanup")
	assert.False(t, ok)
}
