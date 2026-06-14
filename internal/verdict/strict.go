package verdict

import "github.com/alexremn/finalizer-doctor/internal/model"

// Strict is the default, conservative strategy (verdict-engine.md §6).
type Strict struct{}

// Verdict applies, in order: unknown-owner veto, cannot-probe (unreadable
// liveness source) veto, live-signal downgrade, then the hard-dead test.
func (Strict) Verdict(owner model.OwnerCandidate, ev []model.Evidence) model.Verdict {
	if owner.Kind == "Unknown" {
		return finalize(owner, ev, model.StateUnknown, nil)
	}
	if unreadableLiveness(ev) { // cannot-probe ≠ dead
		return finalize(owner, ev, model.StateUnknown, nil)
	}
	if hasClass(ev, model.ClassLive) {
		return finalize(owner, ev, model.StateSlow, nil)
	}
	if hasClass(ev, model.ClassDeadHard) {
		return finalize(owner, ev, model.StateDead, nil)
	}
	return finalize(owner, ev, model.StateUnknown, nil)
}
