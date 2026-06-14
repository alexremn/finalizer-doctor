//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiagnoseStuckNamespace assumes a kind cluster with a CRD+controller
// installed then deleted, leaving a namespace Terminating. See
// fixtures/stuck-namespace.md for the setup commands the CI e2e job runs.
func TestDiagnoseStuckNamespace(t *testing.T) {
	out, err := exec.Command("./kubectl-finalizer_doctor", "ns/e2e-stuck", "--output", "json").CombinedOutput()
	// Exit code 2 (stuck found) is expected; CombinedOutput surfaces non-zero as err.
	require.Error(t, err, "expected non-zero exit (stuck found)")
	s := string(out)
	assert.Contains(t, s, `"state":`)
	assert.True(t, strings.Contains(s, "DEAD") || strings.Contains(s, "SLOW") || strings.Contains(s, "UNKNOWN"))
}
