package probe

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// probeAPIService reports a hard dead signal only for an AGGREGATED APIService
// (spec.service set) whose group matches the owner and whose Available condition
// is False. Local APIServices (spec.service nil) are served by kube-apiserver and
// never inferred dead (verdict-engine.md §3).
func probeAPIService(owner model.OwnerCandidate, snap model.Snapshot) *model.Evidence {
	if owner.Group == "" {
		return nil
	}
	if !snap.Readable(model.SourceAPIServices) {
		return &model.Evidence{Signal: "APIService availability", Source: model.SourceAPIServices, Observed: "could not read APIServices", Class: model.ClassUnreadable}
	}
	for _, as := range snap.RawAPIServices {
		g, _, _ := unstructured.NestedString(as.Object, "spec", "group")
		if g != owner.Group {
			continue
		}
		if _, hasService, _ := unstructured.NestedMap(as.Object, "spec", "service"); !hasService {
			return nil // local APIService: never inferred dead
		}
		status, reason := availableCondition(as)
		if status == "False" {
			return &model.Evidence{
				Signal:   "APIService availability",
				Source:   model.SourceAPIServices,
				Observed: as.GetName() + " Available=False (" + reason + ")",
				Class:    model.ClassDeadHard,
				Weight:   40,
			}
		}
	}
	return nil
}

func availableCondition(as unstructured.Unstructured) (status, reason string) {
	conds, _, _ := unstructured.NestedSlice(as.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _, _ := unstructured.NestedString(m, "type"); strings.EqualFold(t, "Available") {
			status, _, _ = unstructured.NestedString(m, "status")
			reason, _, _ = unstructured.NestedString(m, "reason")
			return status, reason
		}
	}
	return "", ""
}
