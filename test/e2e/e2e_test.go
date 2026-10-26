//go:build e2e

// Package e2e drives the real plugin binary against a live cluster (kind in CI).
// It needs kubectl on PATH and a cluster-admin kind-* current context; objects it
// creates are removed in t.Cleanup (best effort).
package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var bin string // plugin built by TestMain

func TestMain(m *testing.M) {
	// The tests break discovery (a dead APIService) and clear finalizers, so only
	// ever run them against a throwaway kind cluster.
	// ponytail: context-name check; add an explicit opt-in if other disposable clusters need it.
	cur, _ := exec.Command("kubectl", "config", "current-context").Output()
	if ctx := strings.TrimSpace(string(cur)); !strings.HasPrefix(ctx, "kind-") {
		fmt.Fprintf(os.Stderr, "refusing to run e2e: current context %q is not a kind cluster\n", ctx)
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "fd-e2e")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bin = filepath.Join(dir, "kubectl-finalizer_doctor")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/finalizer-doctor").CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build plugin: %v\n%s", err, out)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// fd runs the plugin the way CI/cron does: stdin is /dev/null (exec's default).
func fd(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var o, e bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout, cmd.Stderr = &o, &e
	var exit *exec.ExitError
	if err := cmd.Run(); err != nil && !errors.As(err, &exit) {
		t.Fatalf("run plugin %v: %v", args, err)
	}
	return o.String(), e.String(), cmd.ProcessState.ExitCode()
}

func kubectl(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	cmd := exec.Command("kubectl", args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl %s: %s", strings.Join(args, " "), out)
	return string(out)
}

func kubectlCleanup(t *testing.T, args ...string) {
	t.Cleanup(func() { _ = exec.Command("kubectl", args...).Run() })
}

const deadAPIService = "v1beta1.e2e.finalizer-doctor.test"

// The canonical real-world cause (hack/demo.sh): an aggregated APIService whose
// backing Service is gone breaks discovery, so namespace deletion hangs.
const deadAPIServiceYAML = `apiVersion: apiregistration.k8s.io/v1
kind: APIService
metadata: {name: v1beta1.e2e.finalizer-doctor.test}
spec:
  group: e2e.finalizer-doctor.test
  version: v1beta1
  groupPriorityMinimum: 100
  versionPriority: 100
  service: {name: nonexistent, namespace: kube-system}
  insecureSkipTLSVerify: true
`

func TestStuckNamespaceOnDeadAPIServiceIsCleared(t *testing.T) {
	const ns = "e2e-stuck"
	kubectlCleanup(t, "wait", "--for=delete", "namespace/"+ns, "--timeout=90s")
	kubectl(t, deadAPIServiceYAML, "apply", "-f", "-")
	// Cleanups run LIFO: healthy discovery lets the namespace controller finish
	// deleting ns on its own, then the wait above confirms it is gone.
	kubectlCleanup(t, "delete", "apiservice", deadAPIService, "--ignore-not-found")
	kubectl(t, "", "wait", "--for=condition=Available=False", "apiservice/"+deadAPIService, "--timeout=90s")
	kubectl(t, "", "create", "namespace", ns)
	kubectl(t, "", "delete", "namespace", ns, "--wait=false")
	kubectl(t, "", "wait", "--for=condition=NamespaceDeletionDiscoveryFailure", "ns/"+ns, "--timeout=120s")

	// --all must survive the very discovery failure it is diagnosing, and find a
	// namespace held only by spec.finalizers.
	out, errOut, code := fd(t, "--all")
	require.Equal(t, 2, code, out+errOut)
	assert.Contains(t, out, "to apply: kubectl finalizer-doctor namespaces/"+ns+" --apply --confirm=")
	assert.Contains(t, errOut, "warning: API discovery failed for e2e.finalizer-doctor.test/v1beta1")

	// Singular resource name resolves via the (partial) discovery.
	out, errOut, code = fd(t, "namespace/"+ns, "--output", "json")
	require.Equal(t, 2, code, out+errOut)
	var doc struct {
		Verdicts []struct {
			Finalizer   string `json:"finalizer"`
			State       string `json:"state"`
			SafeToClear bool   `json:"safeToClear"`
		} `json:"verdicts"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &doc), out)
	require.Len(t, doc.Verdicts, 1)
	assert.Equal(t, "kubernetes", doc.Verdicts[0].Finalizer)
	assert.Equal(t, "DEAD", doc.Verdicts[0].State)
	assert.True(t, doc.Verdicts[0].SafeToClear)

	out, errOut, code = fd(t, "ns/"+ns)
	require.Equal(t, 2, code, out+errOut)
	hint := regexp.MustCompile(`(?m)^to apply: kubectl finalizer-doctor (.+) --apply --confirm=([0-9a-f]{12})$`).FindStringSubmatch(out)
	require.NotNil(t, hint, out)
	target := strings.Fields(hint[1])

	out, errOut, code = fd(t, append(target, "--apply", "--confirm=000000000000")...)
	assert.Equal(t, 3, code, out+errOut)
	assert.Contains(t, out, "refused")
	kubectl(t, "", "get", "namespace", ns) // untouched

	// The printed command, run verbatim and non-interactively, clears it.
	out, errOut, code = fd(t, append(target, "--apply", "--confirm="+hint[2])...)
	require.Equal(t, 0, code, out+errOut)
	assert.Contains(t, out, "applied:")
	kubectl(t, "", "wait", "--for=delete", "namespace/"+ns, "--timeout=60s")
}

func TestUnattributableFinalizerIsNeverCleared(t *testing.T) {
	const ns, cm = "e2e-held", "held"
	kubectl(t, "", "create", "namespace", ns)
	kubectlCleanup(t, "delete", "namespace", ns, "--timeout=90s")
	kubectl(t, `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"held","finalizers":["nobody.invalid/hold"]}}`, "-n", ns, "apply", "-f", "-")
	kubectlCleanup(t, "-n", ns, "patch", "configmap", cm, "--type=merge", "-p", `{"metadata":{"finalizers":null}}`)

	out, errOut, code := fd(t, "cm/"+cm, "-n", ns)
	assert.Equal(t, 0, code, out+errOut)
	assert.Equal(t, "not stuck: configmaps/held -n e2e-held is not terminating on finalizers\n", out)
	out, errOut, code = fd(t, "cm/"+cm, "-n", ns, "--apply", "--confirm=000000000000")
	assert.Equal(t, 3, code, out+errOut)
	assert.Contains(t, out, "refused: configmaps/held -n e2e-held is not terminating")

	kubectl(t, "", "-n", ns, "delete", "configmap", cm, "--wait=false")

	out, errOut, code = fd(t, "configmap/"+cm, "-n", ns)
	assert.Equal(t, 2, code, out+errOut)
	assert.Contains(t, out, "verdict:          UNKNOWN")
	assert.NotContains(t, out, "to apply")

	out, errOut, code = fd(t, "configmaps/"+cm, "-n", ns, "--apply", "--confirm=000000000000")
	assert.Equal(t, 3, code, out+errOut)
	assert.Contains(t, out, "refused")
	assert.Contains(t, kubectl(t, "", "-n", ns, "get", "configmap", cm, "-o", "jsonpath={.metadata.finalizers}"), "nobody.invalid/hold")

	out, errOut, code = fd(t, "configmaps/does-not-exist", "-n", ns)
	assert.Equal(t, 1, code, out)
	assert.Contains(t, errOut, "not found")
}
