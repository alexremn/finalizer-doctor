package clustertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authv1 "k8s.io/api/authorization/v1"
)

func TestFakeCanRespectsAllowSet(t *testing.T) {
	f := &Fake{Allowed: map[string]bool{"patch:namespaces:": true}}
	ok, err := f.Can(context.Background(), authv1.ResourceAttributes{Verb: "patch", Resource: "namespaces"})
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = f.Can(context.Background(), authv1.ResourceAttributes{Verb: "delete", Resource: "pods"})
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestFakeRecordsMutations(t *testing.T) {
	f := &Fake{}
	require.NoError(t, f.FinalizeNamespace(context.Background(), "foo", nil, "42"))
	require.Len(t, f.Mutations, 1)
	assert.Equal(t, "FinalizeNamespace foo rv=42", f.Mutations[0])
}
