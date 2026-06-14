package verdict

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func deadAS(group string) unstructured.Unstructured {
	as := unstructured.Unstructured{Object: map[string]any{
		"spec":   map[string]any{"group": group, "service": map[string]any{"name": "m", "namespace": "kube-system"}},
		"status": map[string]any{"conditions": []any{map[string]any{"type": "Available", "status": "False", "reason": "ServiceNotFound"}}},
	}}
	as.SetName("v1beta1." + group)
	return as
}

func crdObj(group string) unstructured.Unstructured {
	return unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"group": group}}}
}

func bothReadable() map[model.Source]model.ReadStatus {
	return map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadOK, model.SourceCRDs: model.ReadOK}
}

func nsObj(conds ...model.Condition) model.StuckObject {
	return model.StuckObject{SpecFinalizers: []string{"kubernetes"}, NamespaceConditions: conds}
}

func TestNamespaceKubernetesDeadClassicCase(t *testing.T) {
	obj := nsObj(model.Condition{Type: "NamespaceDeletionDiscoveryFailure", Status: "True"})
	snap := model.Snapshot{RawAPIServices: []unstructured.Unstructured{deadAS("metrics.example.com")}, SourceStatus: bothReadable()}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateDead, v.State)
	assert.True(t, v.SafeToClear)
}

func TestNamespaceKubernetesCRDExistsRefuses(t *testing.T) {
	obj := nsObj(model.Condition{Type: "NamespaceDeletionDiscoveryFailure", Status: "True"})
	snap := model.Snapshot{
		RawAPIServices: []unstructured.Unstructured{deadAS("example.com")},
		RawCRDs:        []unstructured.Unstructured{crdObj("example.com")},
		SourceStatus:   bothReadable(),
	}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateUnknown, v.State, "CRD-backed group with broken discovery -> content unprovable -> refuse")
}

func TestNamespaceKubernetesContentRemainingIsSlow(t *testing.T) {
	obj := nsObj(
		model.Condition{Type: "NamespaceDeletionDiscoveryFailure", Status: "True"},
		model.Condition{Type: "NamespaceContentRemaining", Status: "True"},
	)
	snap := model.Snapshot{RawAPIServices: []unstructured.Unstructured{deadAS("metrics.example.com")}, SourceStatus: bothReadable()}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateSlow, v.State)
}

func TestNamespaceKubernetesNoDiscoveryFailureIsUnknown(t *testing.T) {
	obj := nsObj()
	snap := model.Snapshot{SourceStatus: bothReadable()}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateUnknown, v.State)
}

func TestNamespaceKubernetesNoDeadAPIServiceIsSlow(t *testing.T) {
	obj := nsObj(model.Condition{Type: "NamespaceDeletionDiscoveryFailure", Status: "True"})
	snap := model.Snapshot{SourceStatus: bothReadable()}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateSlow, v.State)
}

func TestNamespaceKubernetesUnreadableSourcesIsUnknown(t *testing.T) {
	obj := nsObj(model.Condition{Type: "NamespaceDeletionDiscoveryFailure", Status: "True"})
	snap := model.Snapshot{SourceStatus: map[model.Source]model.ReadStatus{model.SourceAPIServices: model.ReadUnreadable, model.SourceCRDs: model.ReadOK}}
	v := NamespaceKubernetes(obj, snap)
	assert.Equal(t, model.StateUnknown, v.State)
}
