package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type RiskResult struct {
	ChangedEntity string
	MissingEntity string
	Coupling      int
}

// Risk cross-references changedFiles against a precomputed coupling table,
// flagging historically-coupled files that are not part of the change.
// Coupling stores each pair once (Entity alphabetically before Coupled), so
// both directions are checked here; a pair is skipped when its other side is
// also part of the change.
func Risk(coupling []CouplingResult, changedFiles []string) []RiskResult {
	changed := make(map[string]struct{}, len(changedFiles))
	for _, f := range changedFiles {
		changed[f] = struct{}{}
	}

	var results []RiskResult
	for _, c := range coupling {
		_, entityChanged := changed[c.Entity]
		_, coupledChanged := changed[c.Coupled]
		switch {
		case entityChanged && !coupledChanged:
			results = append(results, RiskResult{ChangedEntity: c.Entity, MissingEntity: c.Coupled, Coupling: c.Degree})
		case coupledChanged && !entityChanged:
			results = append(results, RiskResult{ChangedEntity: c.Coupled, MissingEntity: c.Entity, Coupling: c.Degree})
		}
	}

	slices.SortFunc(results, func(a, b RiskResult) int {
		if r := cmp.Compare(b.Coupling, a.Coupling); r != 0 {
			return r
		}
		if r := cmp.Compare(a.ChangedEntity, b.ChangedEntity); r != 0 {
			return r
		}
		return cmp.Compare(a.MissingEntity, b.MissingEntity)
	})

	return results
}

func FormatRisk(results []RiskResult, _ model.Options) [][]string {
	out := [][]string{{"changed-entity", "missing-entity", "coupling-percent"}}
	for _, r := range results {
		out = append(out, []string{r.ChangedEntity, r.MissingEntity, fmt.Sprint(r.Coupling)})
	}
	return out
}
