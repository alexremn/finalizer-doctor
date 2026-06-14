package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestGateNonInteractiveRequiresMatchingDigest(t *testing.T) {
	ref := model.ResourceRef{Name: "foo"}
	verdicts := []model.Verdict{{Finalizer: "kubernetes", State: model.StateDead}}
	want := Digest(ref, verdicts, "100")

	g := Gate{Interactive: false, Confirm: want}
	assert.NoError(t, g.Authorize(ref, verdicts, "100"))

	g2 := Gate{Interactive: false, Confirm: "deadbeef0000"}
	assert.ErrorIs(t, g2.Authorize(ref, verdicts, "100"), ErrGateRefused)

	g3 := Gate{Interactive: false, Confirm: ""}
	assert.ErrorIs(t, g3.Authorize(ref, verdicts, "100"), ErrGateRefused)
}

func TestGateNonInteractiveStaleDigestRejected(t *testing.T) {
	ref := model.ResourceRef{Name: "foo"}
	verdicts := []model.Verdict{{Finalizer: "kubernetes", State: model.StateDead}}
	stale := Digest(ref, verdicts, "100")
	g := Gate{Interactive: false, Confirm: stale}
	// resourceVersion moved on -> the same token no longer matches.
	assert.ErrorIs(t, g.Authorize(ref, verdicts, "101"), ErrGateRefused)
}

func TestGateInteractiveTypedNameMatches(t *testing.T) {
	ref := model.ResourceRef{Name: "foo"}
	verdicts := []model.Verdict{{Finalizer: "kubernetes", State: model.StateDead}}
	g := Gate{Interactive: true}.WithTypedName("foo")
	assert.NoError(t, g.Authorize(ref, verdicts, "100"))

	g2 := Gate{Interactive: true}.WithTypedName("wrong")
	assert.ErrorIs(t, g2.Authorize(ref, verdicts, "100"), ErrGateRefused)
}
