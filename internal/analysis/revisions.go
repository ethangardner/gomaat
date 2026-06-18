package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"gomaat/internal/model"
)

type RevisionsResult struct {
	Entity string
	Revs   int
}

func Revisions(commits []model.Commit, _ model.Options) []RevisionsResult {
	revsByEntity := countDistinct(commits, func(c model.Commit) string { return c.Entity }, func(c model.Commit) string { return c.Rev })

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

func FormatRevisions(results []RevisionsResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "n-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Revs)})
	}
	return out
}
