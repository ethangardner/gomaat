package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"gomaat/internal/model"
)

// AbsChurn returns lines added/deleted aggregated by date.
func AbsChurn(commits []model.Commit, _ model.Options) []ChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Date })
	slices.SortFunc(aggs, func(a, b churnAgg) int { return cmp.Compare(a.key, b.key) })

	return aggsToChurnResults(aggs)
}

func FormatAbsChurn(results []ChurnResult, _ model.Options) [][]string {
	return formatChurn(results, "date")
}

// AuthorChurn returns lines added/deleted aggregated by author.
func AuthorChurn(commits []model.Commit, _ model.Options) []ChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Author })
	slices.SortFunc(aggs, func(a, b churnAgg) int { return cmp.Compare(a.key, b.key) })

	return aggsToChurnResults(aggs)
}

func FormatAuthorChurn(results []ChurnResult, _ model.Options) [][]string {
	return formatChurn(results, "author")
}

// EntityChurn returns lines added/deleted aggregated by entity, sorted by added desc.
func EntityChurn(commits []model.Commit, _ model.Options) []ChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Entity })
	slices.SortFunc(aggs, func(a, b churnAgg) int {
		if c := cmp.Compare(b.added, a.added); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})

	return aggsToChurnResults(aggs)
}

func FormatEntityChurn(results []ChurnResult, _ model.Options) [][]string {
	return formatChurn(results, "entity")
}

func formatChurn(results []ChurnResult, keyHeader string) [][]string {
	out := [][]string{{keyHeader, "added", "deleted", "commits"}}
	for _, r := range results {
		out = append(out, []string{r.Key, fmt.Sprint(r.Added), fmt.Sprint(r.Deleted), fmt.Sprint(r.Commits)})
	}
	return out
}

type EntityOwnershipResult struct {
	Entity  string
	Author  string
	Added   int
	Deleted int
}

// EntityOwnership returns churn per (entity, author) pair.
func EntityOwnership(commits []model.Commit, _ model.Options) []EntityOwnershipResult {
	type entry struct{ added, deleted int }
	byKey := map[entityAuthorKey]*entry{}
	for _, c := range commits {
		k := entityAuthorKey{c.Entity, c.Author}
		e, ok := byKey[k]
		if !ok {
			e = &entry{}
			byKey[k] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
	}

	results := make([]EntityOwnershipResult, 0, len(byKey))
	for k, e := range byKey {
		results = append(results, EntityOwnershipResult{k.entity, k.author, e.added, e.deleted})
	}
	slices.SortFunc(results, func(a, b EntityOwnershipResult) int {
		if c := cmp.Compare(a.Entity, b.Entity); c != 0 {
			return c
		}
		return cmp.Compare(a.Author, b.Author)
	})

	return results
}

func FormatEntityOwnership(results []EntityOwnershipResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "author", "added", "deleted"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, r.Author, fmt.Sprint(r.Added), fmt.Sprint(r.Deleted)})
	}
	return out
}

// MainDev returns the author with the most lines added per entity.
func MainDev(commits []model.Commit, _ model.Options) []ContributorResult {
	return findTopContributor(commits, func(c model.Commit) int { return c.LocAdded })
}

func FormatMainDev(results []ContributorResult, _ model.Options) [][]string {
	return formatContributor(results, "added", "total-added")
}

// RefactoringMainDev returns the author with the most lines deleted per entity.
func RefactoringMainDev(commits []model.Commit, _ model.Options) []ContributorResult {
	return findTopContributor(commits, func(c model.Commit) int { return c.LocDeleted })
}

func FormatRefactoringMainDev(results []ContributorResult, _ model.Options) [][]string {
	return formatContributor(results, "removed", "total-removed")
}
