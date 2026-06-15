package model

// ActionKind enumerates the mutating actions the apply stage can perform.
type ActionKind string

// The mutating action kinds, in plan order (orphans first, finalize last).
const (
	ActionCleanOrphan       ActionKind = "CleanOrphan"
	ActionClearFinalizer    ActionKind = "ClearFinalizer"    // metadata.finalizers patch
	ActionFinalizeNamespace ActionKind = "FinalizeNamespace" // spec.finalizers via /finalize
)

// Action is one mutating step in a Plan.
type Action struct {
	Kind       ActionKind
	Target     ResourceRef
	Finalizer  string // the specific finalizer to remove (Clear/Finalize)
	Reason     string
	Reversible bool // always false in v1; deletions/clears are irreversible
}

// Plan is the ordered remediation. Refused holds verdicts that block but are
// not safe to act on (SLOW/UNKNOWN, or blockers).
type Plan struct {
	Actions []Action
	Refused []Verdict
	Notes   []string // e.g. webhook-blocker explanations, orphan warnings
}

// HasActions reports whether the plan would mutate anything.
func (p Plan) HasActions() bool { return len(p.Actions) > 0 }
