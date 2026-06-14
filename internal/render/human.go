// Package render produces human and JSON output from verdicts and a plan.
package render

import (
	"fmt"
	"strings"

	"github.com/alexremn/finalizer-doctor/internal/model"
)

// Human renders the explain block (verdict-engine.md §10).
func Human(verdicts []model.Verdict, plan model.Plan) string {
	var b strings.Builder
	for _, v := range verdicts {
		fmt.Fprintf(&b, "blocking finalizer: %s\n", v.Finalizer)
		fmt.Fprintf(&b, "  attributed to:    %s [%s]\n", v.Owner.MatchReason, v.Owner.Kind)
		fmt.Fprintf(&b, "  verdict:          %s\n", v.State)
		fmt.Fprintln(&b, "  evidence:")
		for _, e := range v.Evidence {
			fmt.Fprintf(&b, "    %s %s\n", e.Class.Tag(), e.Observed)
		}
	}
	if len(plan.Actions) > 0 {
		fmt.Fprintln(&b, "  plan:")
		for i, a := range plan.Actions {
			fmt.Fprintf(&b, "    %d. %s %s (%s)\n", i+1, a.Kind, a.Finalizer, a.Reason)
		}
	}
	for _, n := range plan.Notes {
		fmt.Fprintf(&b, "  note: %s\n", n)
	}
	return b.String()
}
