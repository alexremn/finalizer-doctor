package apply

import (
	"errors"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// ErrGateRefused is returned when authorization fails (exit code 3).
var ErrGateRefused = errors.New("safety gate not satisfied")

// Gate authorizes a mutation. Interactive runs require the typed resource name;
// non-interactive runs require a --confirm digest matching the current proof.
type Gate struct {
	Interactive bool
	Confirm     string // value of --confirm (digest), non-interactive path
	typedName   string // captured from the prompt on the interactive path
}

// WithTypedName returns a copy of the gate carrying the operator's typed input
// (interactive path). Keeping the field unexported keeps Authorize deterministic.
func (g Gate) WithTypedName(name string) Gate {
	g.typedName = name
	return g
}

// Authorize checks the gate against the current verdicts and resourceVersion.
func (g Gate) Authorize(ref model.ResourceRef, verdicts []model.Verdict, resourceVersion string) error {
	if g.Interactive {
		if g.typedName == ref.Name {
			return nil
		}
		return ErrGateRefused
	}
	if g.Confirm != "" && g.Confirm == Digest(ref, verdicts, resourceVersion) {
		return nil
	}
	return ErrGateRefused
}
