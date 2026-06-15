package model

// State is the per-finalizer verdict.
type State string

// The per-finalizer verdict states.
const (
	StateDead    State = "DEAD"
	StateSlow    State = "SLOW"
	StateUnknown State = "UNKNOWN"
)

// OwnerCandidate is a finalizer's mapped owning controller/API/webhook.
type OwnerCandidate struct {
	Finalizer   string
	Kind        string      // "APIService" | "CRD" | "Workload" | "Webhook" | "Builtin" | "Unknown"
	Group       string      // concrete API group the mapping resolved (used by the probe); "" if none
	Ref         ResourceRef // the owner object, when identifiable
	MatchReason string      // how the mapping matched, for operator sanity-check
	Rank        int         // higher = stronger match (exact group > suffix > name)
}

// Verdict is the engine's per-finalizer output.
type Verdict struct {
	Finalizer   string
	Owner       OwnerCandidate
	State       State
	Evidence    []Evidence
	SafeToClear bool
	Score       *int // set only in score mode
}

// IsSafeToClear is true iff the state is DEAD.
func (v Verdict) IsSafeToClear() bool { return v.State == StateDead }
