package mapping

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// Map resolves a finalizer string to its best owner candidate.
func Map(finalizer string, snap model.Snapshot) model.OwnerCandidate {
	if e, ok := builtin(finalizer); ok {
		return model.OwnerCandidate{Finalizer: finalizer, Kind: e.Kind, MatchReason: "built-in: " + e.Owner, Rank: 100}
	}
	left := DomainOf(finalizer)
	if left == "" {
		return model.OwnerCandidate{Finalizer: finalizer, Kind: "Unknown", MatchReason: "no parseable domain", Rank: 0}
	}
	// 1. CRD group match (highest-confidence non-builtin).
	for _, crd := range snap.RawCRDs {
		g, _, _ := unstructured.NestedString(crd.Object, "spec", "group")
		if groupMatches(g, left) {
			return model.OwnerCandidate{Finalizer: finalizer, Kind: "CRD", Group: g, MatchReason: "CRD group " + g, Rank: rankFor(g, left)}
		}
	}
	// 2. APIService group match.
	for _, as := range snap.RawAPIServices {
		g, _, _ := unstructured.NestedString(as.Object, "spec", "group")
		if groupMatches(g, left) {
			return model.OwnerCandidate{Finalizer: finalizer, Kind: "APIService", Group: g, MatchReason: "APIService group " + g, Rank: rankFor(g, left)}
		}
	}
	return model.OwnerCandidate{Finalizer: finalizer, Kind: "Unknown", MatchReason: "domain " + left + " matched no CRD/APIService", Rank: 0}
}

// groupMatches reports whether an API group g owns the finalizer's left-hand
// part. The left part is either "<name>.<group>" or "<group>"; both resolve to g
// when g equals left or left ends with ".<g>".
func groupMatches(g, left string) bool {
	return g != "" && (g == left || strings.HasSuffix(left, "."+g))
}

// ForceStateForUnknown returns UNKNOWN when the owner could not be identified.
func ForceStateForUnknown(c model.OwnerCandidate) model.State {
	if c.Kind == "Unknown" {
		return model.StateUnknown
	}
	return ""
}

// DomainOf returns the group/domain portion of a finalizer: everything before
// the first '/'. The result is either "<name>.<group>" or "<group>"; the
// concrete group is resolved by groupMatches against known CRD/APIService groups.
func DomainOf(finalizer string) string {
	if slash := strings.IndexByte(finalizer, '/'); slash >= 0 {
		return finalizer[:slash]
	}
	return finalizer
}

func rankFor(group, domain string) int {
	if group == domain {
		return 80 // exact group
	}
	return 50 // domain suffix
}
