package analysis

import (
	"cmp"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

// RevisionsResult.Revs is float64 to accommodate a decay-weighted
// (--half-life) count; it holds a whole number when decay is disabled.
type RevisionsResult struct {
	Entity string
	Revs   float64
}

// Revisions counts the number of revisions for each entity, decay-weighted
// by opts.HalfLifeDays when set.
func Revisions(commits []model.Commit, opts model.Options) []RevisionsResult {
	var revsByEntity map[string]float64
	if opts.HalfLifeDays > 0 {
		revsByEntity = countDistinctWeighted(commits,
			func(c model.Commit) string { return c.Entity },
			func(c model.Commit) string { return c.Rev },
			resolveNow(opts), opts.HalfLifeDays)
	} else {
		revsByEntity = toFloatMap(countDistinct(commits,
			func(c model.Commit) string { return c.Entity },
			func(c model.Commit) string { return c.Rev }))
	}

	results := make([]RevisionsResult, 0, len(revsByEntity))
	for entity, revs := range revsByEntity {
		results = append(results, RevisionsResult{entity, revs})
	}
	slices.SortFunc(results, func(a, b RevisionsResult) int {
		if c := cmp.Compare(b.Revs, a.Revs); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})

	return results
}

func FormatRevisions(results []RevisionsResult, opts model.Options) [][]string {
	out := [][]string{{"entity", "n-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, formatMetric(r.Revs, opts)})
	}
	return out
}
