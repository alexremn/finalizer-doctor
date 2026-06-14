// Package verdict decides, per finalizer, whether the owner is DEAD/SLOW/UNKNOWN.
package verdict

import "github.com/alexremn/finalizer-doctor/internal/model"

// Verdicter is the pluggable verdict strategy.
type Verdicter interface {
	Verdict(owner model.OwnerCandidate, ev []model.Evidence) model.Verdict
}

// livenessSources are the sources whose unreadability vetoes a DEAD verdict.
var livenessSources = map[model.Source]bool{
	model.SourceWorkloads:      true,
	model.SourcePods:           true,
	model.SourceEndpointSlices: true,
}

func isLivenessSource(s model.Source) bool { return livenessSources[s] }

func hasClass(ev []model.Evidence, c model.Class) bool {
	for _, e := range ev {
		if e.Class == c {
			return true
		}
	}
	return false
}

func unreadableLiveness(ev []model.Evidence) bool {
	for _, e := range ev {
		if e.Class == model.ClassUnreadable && isLivenessSource(e.Source) {
			return true
		}
	}
	return false
}

func finalize(owner model.OwnerCandidate, ev []model.Evidence, state model.State, score *int) model.Verdict {
	return model.Verdict{
		Finalizer:   owner.Finalizer,
		Owner:       owner,
		State:       state,
		Evidence:    ev,
		SafeToClear: state == model.StateDead,
		Score:       score,
	}
}
