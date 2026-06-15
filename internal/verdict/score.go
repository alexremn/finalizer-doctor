package verdict

import "github.com/alexremn/finalizer-doctor/internal/model"

// Score is the opt-in strategy. It applies the same three vetoes as Strict, then
// requires >=1 hard signal AND score >= threshold for DEAD, so soft signals can
// never produce DEAD alone (verdict-engine.md §7).
type Score struct{}

// Verdict applies the same three vetoes as Strict, then requires >=1 hard signal
// AND score >= threshold for DEAD (verdict-engine.md §7).
func (Score) Verdict(owner model.OwnerCandidate, ev []model.Evidence) model.Verdict {
	total := scoreOf(ev)
	if owner.Kind == "Unknown" {
		return finalize(owner, ev, model.StateUnknown, &total)
	}
	if unreadableLiveness(ev) {
		return finalize(owner, ev, model.StateUnknown, &total)
	}
	if hasClass(ev, model.ClassLive) {
		return finalize(owner, ev, model.StateSlow, &total)
	}
	if hasClass(ev, model.ClassDeadHard) && total >= ScoreThreshold {
		return finalize(owner, ev, model.StateDead, &total)
	}
	if anyDeadward(ev) {
		return finalize(owner, ev, model.StateSlow, &total)
	}
	return finalize(owner, ev, model.StateUnknown, &total)
}

func scoreOf(ev []model.Evidence) int {
	total := 0
	for _, e := range ev {
		if e.Class.IsDeadward() {
			total += e.Weight
		}
	}
	return total
}

func anyDeadward(ev []model.Evidence) bool {
	for _, e := range ev {
		if e.Class.IsDeadward() {
			return true
		}
	}
	return false
}
