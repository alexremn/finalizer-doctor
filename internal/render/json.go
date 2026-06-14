package render

import (
	"encoding/json"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// JSON renders machine-readable output for CI/scripting.
func JSON(verdicts []model.Verdict, plan model.Plan) string {
	type outVerdict struct {
		Finalizer   string           `json:"finalizer"`
		Owner       string           `json:"owner"`
		Kind        string           `json:"kind"`
		State       string           `json:"state"`
		SafeToClear bool             `json:"safeToClear"`
		Score       *int             `json:"score,omitempty"`
		Evidence    []model.Evidence `json:"evidence"`
	}
	doc := struct {
		Verdicts []outVerdict `json:"verdicts"`
		Plan     model.Plan   `json:"plan"`
	}{Plan: plan}
	for _, v := range verdicts {
		doc.Verdicts = append(doc.Verdicts, outVerdict{
			Finalizer: v.Finalizer, Owner: v.Owner.MatchReason, Kind: v.Owner.Kind,
			State: string(v.State), SafeToClear: v.SafeToClear, Score: v.Score, Evidence: v.Evidence,
		})
	}
	out, _ := json.MarshalIndent(doc, "", "  ")
	return string(out)
}
