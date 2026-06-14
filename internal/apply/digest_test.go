package apply

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

func TestDigestStableAndSensitive(t *testing.T) {
	ref := model.ResourceRef{Name: "foo"}
	base := []model.Verdict{{Finalizer: "kubernetes", State: model.StateDead}}

	d1 := Digest(ref, base, "100")
	d2 := Digest(ref, base, "100")
	assert.Equal(t, d1, d2, "same inputs -> same digest")
	assert.Len(t, d1, 12)

	assert.NotEqual(t, d1, Digest(ref, base, "101"), "resourceVersion change -> new digest")
	changed := []model.Verdict{{Finalizer: "kubernetes", State: model.StateSlow}}
	assert.NotEqual(t, d1, Digest(ref, changed, "100"), "verdict change -> new digest")
}
