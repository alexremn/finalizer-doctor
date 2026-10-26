package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/discover"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func asMetaTime() *metav1.Time { t := metav1.Now(); return &t }

func deadAPIService() unstructured.Unstructured {
	as := unstructured.Unstructured{Object: map[string]any{
		"spec":   map[string]any{"group": "example.com", "service": map[string]any{"name": "m", "namespace": "kube-system"}},
		"status": map[string]any{"conditions": []any{map[string]any{"type": "Available", "status": "False", "reason": "ServiceNotFound"}}},
	}}
	as.SetName("v1.example.com")
	return as
}

func stuckCR() unstructured.Unstructured {
	cr := unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "w1", "namespace": "team-a", "resourceVersion": "9"},
	}}
	cr.SetFinalizers([]string{"example.com/cleanup"})
	cr.SetDeletionTimestamp(asMetaTime())
	return cr
}

func TestRunDryRunReportsDeadViaAPIService(t *testing.T) {
	cr := stuckCR()
	f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		return &cr, nil
	}

	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Output: "json", Verdict: "strict"})
	require.NoError(t, err)
	assert.Equal(t, 2, code, "stuck found -> exit 2 in dry-run")
	assert.Contains(t, out, `"state": "DEAD"`)
}

func TestRunApplyAllIsInvalidInvocation(t *testing.T) {
	f := &clustertest.Fake{}
	_, code, err := Run(context.Background(), f, Options{All: true, Apply: true})
	assert.Equal(t, 1, code)
	var inv *InvalidInvocation
	assert.ErrorAs(t, err, &inv)
}

func TestRunApplyNonInteractiveBadDigestRefused(t *testing.T) {
	cr := stuckCR()
	f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
	f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
		return &cr, nil
	}
	out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: true, Confirm: "wrongtoken00"})
	require.NoError(t, err)
	assert.Equal(t, 3, code, "bad digest -> refused")
	assert.Contains(t, out, "refused")
	_ = model.StateDead
}

func notFound(_ context.Context, gvr schema.GroupVersionResource, _, name string) (*unstructured.Unstructured, error) {
	return nil, apierrors.NewNotFound(gvr.GroupResource(), name)
}

func TestRunMissingTargetIsError(t *testing.T) {
	for _, apply := range []bool{false, true} {
		f := &clustertest.Fake{GetFn: notFound}
		out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/nope", Namespace: "team-a", Apply: apply, Confirm: "x"})
		assert.Equal(t, 1, code, "apply=%v: a missing target is an operational error, not stuck/cleared", apply)
		assert.ErrorContains(t, err, "not found")
		assert.Empty(t, out)
		assert.Empty(t, f.Mutations)
	}
}

func TestRunNonTerminatingTargetIsNotStuck(t *testing.T) {
	live := stuckCR()
	live.SetDeletionTimestamp(nil) // finalizer present, but nobody is deleting it
	for _, tc := range []struct {
		apply bool
		code  int
		out   string
	}{
		{false, 0, "not stuck: widgets.example.com/w1 -n team-a is not terminating on finalizers\n"},
		// Under --apply the operator confirmed clearing a stuck object; a live one
		// (wrong name, or recreated since the dry-run) is a refusal, not "cleared".
		{true, 3, "refused: widgets.example.com/w1 -n team-a is not terminating on finalizers; nothing to clear\n"},
	} {
		f := &clustertest.Fake{APIServiceObjs: []unstructured.Unstructured{deadAPIService()}}
		f.GetFn = func(context.Context, schema.GroupVersionResource, string, string) (*unstructured.Unstructured, error) {
			c := live
			return &c, nil
		}
		out, code, err := Run(context.Background(), f, Options{Target: "widgets.example.com/w1", Namespace: "team-a", Apply: tc.apply, Confirm: "x"})
		require.NoError(t, err)
		assert.Equal(t, tc.code, code, "apply=%v", tc.apply)
		assert.Equal(t, tc.out, out)
		assert.Empty(t, f.Mutations, "never strip a finalizer from a live object")
	}
}

func TestRunAllWarnsAboutGroupsItCouldNotScan(t *testing.T) {
	f := &clustertest.Fake{Errs: map[string]error{"ServerPreferredResources": &discovery.ErrGroupDiscoveryFailed{
		Groups: map[schema.GroupVersion]error{{Group: "metrics.example.com", Version: "v1beta1"}: errors.New("stale GroupVersion discovery")},
	}}}
	var warn bytes.Buffer
	out, code, err := Run(context.Background(), f, Options{All: true, Warn: &warn})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "no stuck resources found\n", out, "the warning goes to stderr, never into (JSON) stdout")
	assert.Equal(t, "warning: API discovery failed for metrics.example.com/v1beta1; objects there were not scanned\n", warn.String())
}

func TestRunAllTargetVanishedReportsNothingStuck(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "example.com", Version: "v1", Resource: "widgets"}
	f := &clustertest.Fake{
		Preferred: []*metav1.APIResourceList{{
			GroupVersion: "example.com/v1",
			APIResources: []metav1.APIResource{{Name: "widgets", Namespaced: true, Verbs: metav1.Verbs{"list"}}},
		}},
		Lists: map[string]*unstructured.UnstructuredList{clustertest.ListKey(gvr, ""): {Items: []unstructured.Unstructured{stuckCR()}}},
		GetFn: notFound, // finished deleting between the scan and the snapshot
	}
	out, code, err := Run(context.Background(), f, Options{All: true})
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "no stuck resources found\n", out)
}

func TestCLITargetRoundTripsThroughParseTarget(t *testing.T) {
	for want, ref := range map[string]model.ResourceRef{
		"namespaces/demo":                  {GVR: schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, Name: "demo"},
		"configmaps/c -n default":          {GVR: schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, Namespace: "default", Name: "c"},
		"widgets.example.com/w1 -n team-a": widgetRef(),
	} {
		assert.Equal(t, want, cliTarget(ref))
		target, ns, _ := strings.Cut(want, " -n ")
		parsed, err := discover.ParseTarget(target, ns)
		require.NoError(t, err)
		assert.Equal(t, discover.Target{Group: ref.GVR.Group, Resource: ref.GVR.Resource, Namespace: ref.Namespace, Name: ref.Name}, parsed)
	}
}
