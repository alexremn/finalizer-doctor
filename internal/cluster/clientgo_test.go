package cluster

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// stubDisc returns canned ServerPreferredResources results; other methods panic.
type stubDisc struct {
	discovery.DiscoveryInterfaceWithContext
	lists []*metav1.APIResourceList
	err   error
}

func (d stubDisc) ServerPreferredResourcesWithContext(context.Context) ([]*metav1.APIResourceList, error) {
	return d.lists, d.err
}

func TestPreferredResourcesKeepsPartialAndNamesFailedGroups(t *testing.T) {
	core := []*metav1.APIResourceList{{GroupVersion: "v1", APIResources: []metav1.APIResource{{Name: "configmaps"}}}}
	dead := &discovery.ErrGroupDiscoveryFailed{Groups: map[schema.GroupVersion]error{
		{Group: "metrics.example.com", Version: "v1beta1"}: errors.New("stale GroupVersion discovery"),
		{Group: "a.example.com", Version: "v1"}:            errors.New("stale GroupVersion discovery"),
	}}
	c := &clientGo{disc: stubDisc{lists: core, err: dead}}

	got, failed, err := PreferredResources(context.Background(), c)
	require.NoError(t, err, "a dead aggregated group must not fail discovery of the healthy ones")
	assert.Equal(t, core, got)
	assert.Equal(t, []string{"a.example.com/v1", "metrics.example.com/v1beta1"}, failed, "sorted for stable output")
}

func TestPreferredResourcesPropagatesTotalFailure(t *testing.T) {
	c := &clientGo{disc: stubDisc{err: errors.New("connection refused")}}
	_, failed, err := PreferredResources(context.Background(), c)
	assert.ErrorContains(t, err, "connection refused")
	assert.Nil(t, failed)
}

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
