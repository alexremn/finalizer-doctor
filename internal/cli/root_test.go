package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/discover"
)

func TestVersionSubcommand(t *testing.T) {
	var code int
	root := newRootCmd(&code)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	require.NoError(t, root.Execute())
	assert.Contains(t, buf.String(), "finalizer-doctor")
}

// /dev/null is a character device, so a mode-bit check misreads it as a TTY and
// sends `--apply --confirm` (cron/CI, stdin </dev/null) down the typed-name path.
func TestIsTTYFalseForDevNull(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)
	defer func() { _ = devNull.Close() }()
	orig := os.Stdin
	os.Stdin = devNull
	defer func() { os.Stdin = orig }()

	assert.False(t, isTTY())
}

// Usage errors must surface as usage errors, not be masked by a kubeconfig or
// cluster failure that happens to come first.
func TestRootRejectsInvalidFlagsBeforeLoadingKubeconfig(t *testing.T) {
	for _, args := range [][]string{
		{"--apply", "--all"},
		{"ns/x", "--verdict", "strcit"},
		{"ns/x", "--output", "yaml"},
	} {
		var code int
		root := newRootCmd(&code)
		root.SetOut(io.Discard)
		root.SetArgs(append(args, "--kubeconfig", "/nonexistent/kubeconfig"))
		err := root.Execute()
		var inv *InvalidInvocation
		assert.ErrorAs(t, err, &inv, "%v", args)
		assert.Equal(t, 1, code, "%v", args)
	}
}

func TestResolveGVRViaDiscovery(t *testing.T) {
	f := &clustertest.Fake{Preferred: []*metav1.APIResourceList{{
		GroupVersion: "example.com/v1",
		APIResources: []metav1.APIResource{{Name: "widgets", SingularName: "widget", Kind: "Widget"}},
	}}}
	ref, err := resolveGVR(context.Background(), f, discover.Target{Group: "example.com", Resource: "widget", Namespace: "team-a", Name: "w1"})
	require.NoError(t, err)
	assert.Equal(t, schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}, ref.GVR)
	assert.Equal(t, "w1", ref.Name)
}

func TestRunApplyRefusesNonDeadTarget(t *testing.T) {
	cr := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "w1", "namespace": "team-a", "resourceVersion": "3"},
	}}
	cr.SetFinalizers([]string{"mystery.io/x"})
	cr.SetDeletionTimestamp(asMetaTime())
	f := &clustertest.Fake{}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		return &cr, nil
	}
	out, code, err := Run(context.Background(), f, Options{Target: "things.mystery.io/w1", Namespace: "team-a", Apply: true, Confirm: "whatever0000"})
	require.NoError(t, err)
	assert.Equal(t, 3, code)
	assert.Contains(t, out, "refused")
}
