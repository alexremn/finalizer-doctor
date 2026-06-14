package plan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func nsRef() model.ResourceRef {
	return model.ResourceRef{GVR: schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, Name: "foo"}
}

func TestPlanRefusesWhenAnyFinalizerNotDead(t *testing.T) {
	obj := model.StuckObject{Ref: nsRef(), SpecFinalizers: []string{"kubernetes"}, MetadataFinalizers: []string{"op/x"}}
	verdicts := []model.Verdict{
		{Finalizer: "kubernetes", State: model.StateDead},
		{Finalizer: "op/x", State: model.StateSlow},
	}
	p := Build(obj, verdicts, nil, false)
	assert.Empty(t, p.Actions, "must refuse: not all finalizers DEAD")
	assert.Len(t, p.Refused, 1)
}

func TestPlanNamespaceFinalizeOrderedLast(t *testing.T) {
	obj := model.StuckObject{Ref: nsRef(), SpecFinalizers: []string{"kubernetes"}, MetadataFinalizers: []string{"op/x"}}
	verdicts := []model.Verdict{
		{Finalizer: "op/x", State: model.StateDead},
		{Finalizer: "kubernetes", State: model.StateDead},
	}
	orphans := []model.ResourceRef{{Name: "child", Namespace: "foo"}}
	p := Build(obj, verdicts, orphans, false)

	require.Len(t, p.Actions, 3)
	assert.Equal(t, model.ActionCleanOrphan, p.Actions[0].Kind)
	assert.Equal(t, model.ActionClearFinalizer, p.Actions[1].Kind)
	assert.Equal(t, "op/x", p.Actions[1].Finalizer)
	last := p.Actions[len(p.Actions)-1]
	assert.Equal(t, model.ActionFinalizeNamespace, last.Kind)
	assert.Equal(t, "kubernetes", last.Finalizer)
}

func TestPlanWebhookBlockerRefuses(t *testing.T) {
	obj := model.StuckObject{Ref: nsRef(), SpecFinalizers: []string{"kubernetes"}}
	verdicts := []model.Verdict{{Finalizer: "kubernetes", State: model.StateDead}}
	p := Build(obj, verdicts, nil, true /* webhookBlocker */)
	assert.Empty(t, p.Actions)
	assert.NotEmpty(t, p.Notes)
}
