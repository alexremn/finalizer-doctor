package model

// Class classifies a single piece of Evidence. It drives both the verdict
// algorithm (verdict-engine.md §6) and the render tags (§10).
type Class string

const (
	ClassDeadHard   Class = "DeadHard"   // hard proof the owner is gone
	ClassDeadSoft   Class = "DeadSoft"   // suggestive only (time/staleness); score mode only
	ClassLive       Class = "Live"       // positive proof the owner is alive
	ClassNeutral    Class = "Neutral"    // confirmatory non-deadward fact
	ClassUnreadable Class = "Unreadable" // a relevant source could not be read
)

// Tag is the short label shown in human output.
func (c Class) Tag() string {
	switch c {
	case ClassDeadHard:
		return "[hard]"
	case ClassDeadSoft:
		return "[soft]"
	case ClassLive:
		return "[live]"
	case ClassNeutral:
		return "[ ok ]"
	default:
		return "[????]"
	}
}

// IsDeadward reports whether this class points toward "owner is dead".
func (c Class) IsDeadward() bool { return c == ClassDeadHard || c == ClassDeadSoft }

// Evidence is one observed fact about an owner's liveness.
type Evidence struct {
	Signal   string // short signal name, e.g. "APIService availability"
	Source   Source // which evidence source produced it
	Observed string // human-readable observation, surfaced verbatim
	Class    Class
	Weight   int // score-mode weight; 0 in strict-only evidence
}
