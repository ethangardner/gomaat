package analysis

import (
	"fmt"

	"github.com/ethangardner/gomaat/internal/model"
)

type OrphanResult struct {
	Entity string
	Added  int
	Date   string
	Author string
}

// Orphans returns entities touched by exactly one commit in the log that
// added at least opts.MinLines lines: code dropped in once and never revised.
// It's a heuristic, not proof of dead code; stable, vendored, or generated
// files match too. Results keep EntityChurn's order: added lines descending,
// then entity.
func Orphans(commits []model.Commit, opts model.Options) []OrphanResult {
	// An orphan has one commit, so any commit for the entity is that one.
	commitFor := make(map[string]model.Commit, len(commits))
	for _, c := range commits {
		commitFor[c.Entity] = c
	}

	var results []OrphanResult
	for _, churn := range EntityChurn(commits, opts) {
		if churn.Commits != 1 || churn.Added < opts.MinLines {
			continue
		}
		c := commitFor[churn.Key]
		results = append(results, OrphanResult{churn.Key, churn.Added, c.Date, c.Author})
	}
	return results
}

func FormatOrphans(results []OrphanResult, _ model.Options) [][]string {
	return formatRows([]string{"entity", "added-lines", "date", "author"}, results, func(r OrphanResult) []string {
		return []string{r.Entity, fmt.Sprint(r.Added), r.Date, r.Author}
	})
}
