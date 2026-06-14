package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestNewFromConfigBuildsClient(t *testing.T) {
	c, err := NewFromConfig(&rest.Config{Host: "https://example.invalid"})
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestFinalizersMergePatchPinsResourceVersion(t *testing.T) {
	patch, err := finalizersMergePatch(nil, "42")
	require.NoError(t, err)
	assert.Contains(t, string(patch), `"resourceVersion":"42"`)
	assert.Contains(t, string(patch), `"finalizers":[]`)
}
