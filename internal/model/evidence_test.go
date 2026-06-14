package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassTag(t *testing.T) {
	assert.Equal(t, "[hard]", ClassDeadHard.Tag())
	assert.Equal(t, "[soft]", ClassDeadSoft.Tag())
	assert.Equal(t, "[live]", ClassLive.Tag())
	assert.Equal(t, "[ ok ]", ClassNeutral.Tag())
	assert.Equal(t, "[????]", ClassUnreadable.Tag())
}

func TestClassIsDeadward(t *testing.T) {
	assert.True(t, ClassDeadHard.IsDeadward())
	assert.True(t, ClassDeadSoft.IsDeadward())
	assert.False(t, ClassLive.IsDeadward())
	assert.False(t, ClassNeutral.IsDeadward())
	assert.False(t, ClassUnreadable.IsDeadward())
}
