package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
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

// EntityOwnershipResult's Added/Deleted are float64 to accommodate
// decay-weighted (--half-life) line counts; they hold whole numbers when
// decay is disabled.
type EntityOwnershipResult struct {
	Entity  string
	Author  string
	Added   float64
	Deleted float64
}

// EntityOwnership returns churn per (entity, author) pair, decay-weighted by
// opts.HalfLifeDays when set.
func EntityOwnership(commits []model.Commit, opts model.Options) []EntityOwnershipResult {
	type entry struct{ added, deleted float64 }
	byKey := map[entityAuthorKey]*entry{}
	now := resolveNow(opts)
	for _, c := range commits {
		weight := 1.0
		if opts.HalfLifeDays > 0 {
			w, ok := decayWeight(c.Date, now, opts.HalfLifeDays)
			if !ok {
				continue
			}
			weight = w
		}
		k := entityAuthorKey{c.Entity, c.Author}
		e, ok := byKey[k]
		if !ok {
			e = &entry{}
			byKey[k] = e
		}
		e.added += float64(c.LocAdded) * weight
		e.deleted += float64(c.LocDeleted) * weight
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

func FormatEntityOwnership(results []EntityOwnershipResult, opts model.Options) [][]string {
	out := [][]string{{"entity", "author", "added", "deleted"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, r.Author, formatMetric(r.Added, opts), formatMetric(r.Deleted, opts)})
	}
	return out
}

// MainDev returns the author with the most lines added per entity,
// decay-weighted by opts.HalfLifeDays when set.
func MainDev(commits []model.Commit, opts model.Options) []ContributorResult {
	return findTopContributor(commits, opts, func(c model.Commit) int { return c.LocAdded })
}

func FormatMainDev(results []ContributorResult, opts model.Options) [][]string {
	return formatContributor(results, opts, "added", "total-added")
}

// RefactoringMainDev returns the author with the most lines deleted per
// entity, decay-weighted by opts.HalfLifeDays when set.
func RefactoringMainDev(commits []model.Commit, opts model.Options) []ContributorResult {
	return findTopContributor(commits, opts, func(c model.Commit) int { return c.LocDeleted })
}

func FormatRefactoringMainDev(results []ContributorResult, opts model.Options) [][]string {
	return formatContributor(results, opts, "removed", "total-removed")
}
