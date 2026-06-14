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
	domain := DomainOf(finalizer)
	if domain == "" {
		return model.OwnerCandidate{Finalizer: finalizer, Kind: "Unknown", MatchReason: "no parseable domain", Rank: 0}
	}
	// 1. CRD group match (highest-confidence non-builtin).
	for _, crd := range snap.RawCRDs {
		g, _, _ := unstructured.NestedString(crd.Object, "spec", "group")
		if g != "" && (g == domain || strings.HasSuffix(domain, g)) {
			return model.OwnerCandidate{Finalizer: finalizer, Kind: "CRD", MatchReason: "CRD group " + g, Rank: rankFor(g, domain)}
		}
	}
	// 2. APIService group match.
	for _, as := range snap.RawAPIServices {
		g, _, _ := unstructured.NestedString(as.Object, "spec", "group")
		if g != "" && (g == domain || strings.HasSuffix(domain, g)) {
			return model.OwnerCandidate{Finalizer: finalizer, Kind: "APIService", MatchReason: "APIService group " + g, Rank: rankFor(g, domain)}
		}
	}
	return model.OwnerCandidate{Finalizer: finalizer, Kind: "Unknown", MatchReason: "domain " + domain + " matched no CRD/APIService", Rank: 0}
}

// ForceStateForUnknown returns UNKNOWN when the owner could not be identified.
func ForceStateForUnknown(c model.OwnerCandidate) model.State {
	if c.Kind == "Unknown" {
		return model.StateUnknown
	}
	return ""
}

// DomainOf extracts the domain from "<name>.<domain>/<x>" or "<domain>/<x>".
// Exported because the CLI run pipeline uses it to pass the domain to the probe.
func DomainOf(finalizer string) string {
	left := finalizer
	if slash := strings.IndexByte(finalizer, '/'); slash >= 0 {
		left = finalizer[:slash]
	}
	if dot := strings.IndexByte(left, '.'); dot >= 0 {
		return left[dot+1:]
	}
	return left
}

func rankFor(group, domain string) int {
	if group == domain {
		return 80 // exact group
	}
	return 50 // domain suffix
}
