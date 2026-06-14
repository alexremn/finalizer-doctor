package apply

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/cluster/clustertest"
	"github.com/alexremn/finalizer-doctor/internal/model"
)

func nsRef() model.ResourceRef {
	return model.ResourceRef{GVR: schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, Name: "foo"}
}

func TestExecuteRunsActionsInOrder(t *testing.T) {
	f := &clustertest.Fake{}
	plan := model.Plan{Actions: []model.Action{
		{Kind: model.ActionCleanOrphan, Target: model.ResourceRef{GVR: schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, Namespace: "foo", Name: "child"}},
		{Kind: model.ActionFinalizeNamespace, Target: nsRef(), Finalizer: "kubernetes"},
	}}
	reverify := func(model.ResourceRef) (model.State, string, error) { return model.StateDead, "100", nil }

	res, err := Execute(context.Background(), f, plan, reverify)
	require.NoError(t, err)
	assert.Equal(t, 2, res.Completed)
	require.Len(t, f.Mutations, 2)
	assert.Contains(t, f.Mutations[0], "Delete")
	assert.Contains(t, f.Mutations[1], "FinalizeNamespace foo")
}

func TestExecuteAbortsWhenReverifyNoLongerDead(t *testing.T) {
	f := &clustertest.Fake{}
	plan := model.Plan{Actions: []model.Action{{Kind: model.ActionFinalizeNamespace, Target: nsRef(), Finalizer: "kubernetes"}}}
	reverify := func(model.ResourceRef) (model.State, string, error) { return model.StateSlow, "100", nil }

	_, err := Execute(context.Background(), f, plan, reverify)
	assert.ErrorIs(t, err, ErrReverifyChanged)
	assert.Empty(t, f.Mutations, "must not mutate when re-verify is no longer DEAD")
}

func TestExecuteStopsOnFirstError(t *testing.T) {
	f := &clustertest.Fake{Errs: map[string]error{"PatchFinalizers": assert.AnError}}
	plan := model.Plan{Actions: []model.Action{
		{Kind: model.ActionClearFinalizer, Target: model.ResourceRef{GVR: schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, Namespace: "foo", Name: "c"}, Finalizer: "op/x"},
	}}
	reverify := func(model.ResourceRef) (model.State, string, error) { return model.StateDead, "5", nil }
	res, err := Execute(context.Background(), f, plan, reverify)
	require.Error(t, err)
	assert.Equal(t, 0, res.Completed)
}
