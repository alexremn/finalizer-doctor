package cli

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexremn/finalizer-doctor/internal/apply"
	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestRunPreflightDeniedReadRefuses(t *testing.T) {
	// Non-nil empty Allowed -> the fake denies every verb.
	f := &clustertest.Fake{Allowed: map[string]bool{}}
	_, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a"})
	assert.Equal(t, 1, code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
	assert.Contains(t, err.Error(), `"get"`)
}

func TestRunPreflightDeniedMutateRefuses(t *testing.T) {
	f := &clustertest.Fake{Allowed: map[string]bool{
		clustertest.AllowKey("get", "widgets", ""): true, // can read, cannot patch
	}}
	_, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: "x"})
	assert.Equal(t, 1, code)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"patch"`)
}

func TestRunApplyWritesAuditFile(t *testing.T) {
	path := t.TempDir() + "/audit.log"
	f := deadCRFake() // nil Allowed -> permissive preflight
	digest := apply.Digest(widgetRef(), []model.Verdict{{Finalizer: "example.com/cleanup", State: model.StateDead}}, "9")

	_, code, err := Run(context.Background(), f, Options{
		Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: digest, AuditFile: path,
	})
	require.NoError(t, err)
	require.Equal(t, 0, code)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ClearFinalizer") // audit records the action kind
	assert.Contains(t, string(data), "team-a/widgets.example.com/w1")
}

func TestRootRegistersTimeoutAndAuditFlags(t *testing.T) {
	var code int
	root := newRootCmd(&code)
	assert.NotNil(t, root.Flags().Lookup("timeout"))
	assert.NotNil(t, root.Flags().Lookup("audit-file"))
}
