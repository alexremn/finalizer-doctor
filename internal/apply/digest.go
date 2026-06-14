// Package apply holds the gate, confirm digest, and gated executor.
package apply

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// Digest binds the confirmation token to the proof: target + sorted
// (finalizer:state) pairs + observed resourceVersion (safety-model.md §3).
func Digest(ref model.ResourceRef, verdicts []model.Verdict, resourceVersion string) string {
	pairs := make([]string, 0, len(verdicts))
	for _, v := range verdicts {
		pairs = append(pairs, v.Finalizer+":"+string(v.State))
	}
	sort.Strings(pairs)
	canonical := strings.Join([]string{ref.String(), strings.Join(pairs, ","), resourceVersion}, "|")
	sum := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(sum[:])[:12]
}
