package verdict

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// NSKubernetesFinalizer is the namespace spec.finalizer cleared via /finalize.
const NSKubernetesFinalizer = "kubernetes"

// NamespaceKubernetes verdicts the namespace `kubernetes` spec finalizer
// (verdict-engine.md §4). It is evidence-based binary regardless of mode: the
// namespace controller is essentially never itself dead, so the verdict is
// attributed to a failing dependency (an aggregated APIService) and is only DEAD
// when (1) discovery is failing, (2) a backing aggregated APIService is
// Available=False, (3) that group has no CRD (proving no stored content), and
// (4) no namespace content remains.
func NamespaceKubernetes(obj model.StuckObject, snap model.Snapshot) model.Verdict {
	owner := model.OwnerCandidate{
		Finalizer:   NSKubernetesFinalizer,
		Kind:        "Builtin",
		MatchReason: "namespace controller; attributed to failing dependency",
	}
	var ev []model.Evidence

	if !conditionTrue(obj.NamespaceConditions, "NamespaceDeletionDiscoveryFailure") &&
		!conditionTrue(obj.NamespaceConditions, "NamespaceDeletionGroupVersionParsingFailure") {
		// Not the classic dead-API signature; this special case can't prove it.
		ev = append(ev, model.Evidence{Signal: "namespace condition", Source: model.SourceTargets, Observed: "no discovery/parsing failure; stuck for another reason", Class: model.ClassNeutral})
		return finalize(owner, ev, model.StateUnknown, nil)
	}
	ev = append(ev, model.Evidence{Signal: "namespace condition", Source: model.SourceTargets, Observed: "discovery/group-version failure present", Class: model.ClassDeadHard, Weight: 35})

	// Either of these means the namespace will re-stick or content removal is
	// failing — clearing `kubernetes` would orphan live content. Refuse.
	for _, c := range []string{"NamespaceFinalizersRemaining", "NamespaceDeletionContentFailure"} {
		if conditionTrue(obj.NamespaceConditions, c) {
			ev = append(ev, model.Evidence{Signal: "namespace condition", Source: model.SourceTargets, Observed: c + "=True (downstream still holding the namespace)", Class: model.ClassLive})
			return finalize(owner, ev, model.StateSlow, nil)
		}
	}

	if !snap.Readable(model.SourceAPIServices) || !snap.Readable(model.SourceCRDs) {
		ev = append(ev, model.Evidence{Signal: "evidence sources", Source: model.SourceAPIServices, Observed: "could not read APIServices/CRDs", Class: model.ClassUnreadable})
		return finalize(owner, ev, model.StateUnknown, nil)
	}

	name, group, reason, found := deadAggregatedAPIService(snap)
	if !found {
		ev = append(ev, model.Evidence{Signal: "APIService availability", Source: model.SourceAPIServices, Observed: "no Available=False aggregated APIService found", Class: model.ClassNeutral})
		return finalize(owner, ev, model.StateSlow, nil)
	}
	ev = append(ev, model.Evidence{Signal: "APIService availability", Source: model.SourceAPIServices, Observed: name + " Available=False (" + reason + ") [aggregated]", Class: model.ClassDeadHard, Weight: 40})

	if crdExistsForGroup(snap, group) {
		// Discovery is broken, so the group's CRs cannot be listed to prove the
		// namespace is empty of them. Never infer "no content" → refuse.
		ev = append(ev, model.Evidence{Signal: "CRD presence", Source: model.SourceCRDs, Observed: "CRD exists for group " + group + "; stored content unprovable while discovery is broken", Class: model.ClassUnreadable})
		return finalize(owner, ev, model.StateUnknown, nil)
	}
	ev = append(ev, model.Evidence{Signal: "CRD presence", Source: model.SourceCRDs, Observed: "no CRD for group " + group + " (aggregated → no stored content)", Class: model.ClassNeutral})

	if conditionTrue(obj.NamespaceConditions, "NamespaceContentRemaining") {
		ev = append(ev, model.Evidence{Signal: "namespace condition", Source: model.SourceTargets, Observed: "NamespaceContentRemaining=True (content still present)", Class: model.ClassLive})
		return finalize(owner, ev, model.StateSlow, nil)
	}
	ev = append(ev, model.Evidence{Signal: "namespace condition", Source: model.SourceTargets, Observed: "no content remaining", Class: model.ClassNeutral})
	return finalize(owner, ev, model.StateDead, nil)
}

func conditionTrue(conds []model.Condition, typ string) bool {
	for _, c := range conds {
		if c.Type == typ && c.Status == "True" {
			return true
		}
	}
	return false
}

// deadAggregatedAPIService returns the first aggregated (spec.service set)
// APIService whose Available condition is False.
func deadAggregatedAPIService(snap model.Snapshot) (name, group, reason string, found bool) {
	for _, as := range snap.RawAPIServices {
		if _, hasSvc, _ := unstructured.NestedMap(as.Object, "spec", "service"); !hasSvc {
			continue
		}
		st, rs := nsAvailableCondition(as)
		if st == "False" {
			g, _, _ := unstructured.NestedString(as.Object, "spec", "group")
			return as.GetName(), g, rs, true
		}
	}
	return "", "", "", false
}

func crdExistsForGroup(snap model.Snapshot, group string) bool {
	for _, crd := range snap.RawCRDs {
		if g, _, _ := unstructured.NestedString(crd.Object, "spec", "group"); g == group {
			return true
		}
	}
	return false
}

func nsAvailableCondition(as unstructured.Unstructured) (status, reason string) {
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
