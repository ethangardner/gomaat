package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type RevisionsResult struct {
	Entity string
	Revs   int
}

// Revisions counts the number of revisions for each entity.
func Revisions(commits []model.Commit, _ model.Options) []RevisionsResult {
	revsByEntity := countDistinct(commits, func(c model.Commit) string { return c.Entity }, func(c model.Commit) string { return c.Rev })

	results := make([]RevisionsResult, 0, len(revsByEntity))
	for entity, revs := range revsByEntity {
		results = append(results, RevisionsResult{entity, revs})
	}
	slices.SortFunc(results, func(a, b RevisionsResult) int {
		return cmp.Or(cmp.Compare(b.Revs, a.Revs), cmp.Compare(a.Entity, b.Entity))
	})

	return results
}

func FormatRevisions(results []RevisionsResult, _ model.Options) [][]string {
	return formatRows([]string{"entity", "n-revs"}, results, func(r RevisionsResult) []string {
		return []string{r.Entity, fmt.Sprint(r.Revs)}
	})
}
