package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"gomaat/internal/model"
)

type AbsChurnResult struct {
	Date    string
	Added   int
	Deleted int
	Commits int
}

// AbsChurn returns lines added/deleted aggregated by date.
func AbsChurn(commits []model.Commit, _ model.Options) []AbsChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Date })
	slices.SortFunc(aggs, func(a, b churnAgg) int { return cmp.Compare(a.key, b.key) })

	results := make([]AbsChurnResult, len(aggs))
	for i, a := range aggs {
		results[i] = AbsChurnResult{a.key, a.added, a.deleted, a.commits}
	}
	return results
}

func FormatAbsChurn(results []AbsChurnResult, _ model.Options) [][]string {
	out := [][]string{{"date", "added", "deleted", "commits"}}
	for _, r := range results {
		out = append(out, []string{r.Date, fmt.Sprint(r.Added), fmt.Sprint(r.Deleted), fmt.Sprint(r.Commits)})
	}
	return out
}

type AuthorChurnResult struct {
	Author  string
	Added   int
	Deleted int
	Commits int
}

// AuthorChurn returns lines added/deleted aggregated by author.
func AuthorChurn(commits []model.Commit, _ model.Options) []AuthorChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Author })
	slices.SortFunc(aggs, func(a, b churnAgg) int { return cmp.Compare(a.key, b.key) })

	results := make([]AuthorChurnResult, len(aggs))
	for i, a := range aggs {
		results[i] = AuthorChurnResult{a.key, a.added, a.deleted, a.commits}
	}
	return results
}

func FormatAuthorChurn(results []AuthorChurnResult, _ model.Options) [][]string {
	out := [][]string{{"author", "added", "deleted", "commits"}}
	for _, r := range results {
		out = append(out, []string{r.Author, fmt.Sprint(r.Added), fmt.Sprint(r.Deleted), fmt.Sprint(r.Commits)})
	}
	return out
}

type EntityChurnResult struct {
	Entity  string
	Added   int
	Deleted int
	Commits int
}

// EntityChurn returns lines added/deleted aggregated by entity, sorted by added desc.
func EntityChurn(commits []model.Commit, _ model.Options) []EntityChurnResult {
	aggs := aggregateChurn(commits, func(c model.Commit) string { return c.Entity })
	slices.SortFunc(aggs, func(a, b churnAgg) int {
		if c := cmp.Compare(b.added, a.added); c != 0 {
			return c
		}
		return cmp.Compare(a.key, b.key)
	})

	results := make([]EntityChurnResult, len(aggs))
	for i, a := range aggs {
		results[i] = EntityChurnResult{a.key, a.added, a.deleted, a.commits}
	}
	return results
}

func FormatEntityChurn(results []EntityChurnResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "added", "deleted", "commits"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Added), fmt.Sprint(r.Deleted), fmt.Sprint(r.Commits)})
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
	type key struct{ entity, author string }
	type entry struct{ added, deleted int }
	byKey := map[key]*entry{}
	for _, c := range commits {
		k := key{c.Entity, c.Author}
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

type MainDevResult struct {
	Entity     string
	MainDev    string
	Added      int
	TotalAdded int
	Ownership  float64
}

// MainDev returns the author with the most lines added per entity.
func MainDev(commits []model.Commit, _ model.Options) []MainDevResult {
	entries := findTopContributor(commits, func(c model.Commit) int { return c.LocAdded })
	results := make([]MainDevResult, len(entries))
	for i, e := range entries {
		results[i] = MainDevResult{e.entity, e.author, e.count, e.total, e.ownership}
	}
	return results
}

func FormatMainDev(results []MainDevResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "main-dev", "added", "total-added", "ownership"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, r.MainDev,
			fmt.Sprint(r.Added), fmt.Sprint(r.TotalAdded),
			fmt.Sprintf("%.2f", r.Ownership),
		})
	}
	return out
}

type RefactoringMainDevResult struct {
	Entity       string
	MainDev      string
	Removed      int
	TotalRemoved int
	Ownership    float64
}

// RefactoringMainDev returns the author with the most lines deleted per entity.
func RefactoringMainDev(commits []model.Commit, _ model.Options) []RefactoringMainDevResult {
	entries := findTopContributor(commits, func(c model.Commit) int { return c.LocDeleted })
	results := make([]RefactoringMainDevResult, len(entries))
	for i, e := range entries {
		results[i] = RefactoringMainDevResult{e.entity, e.author, e.count, e.total, e.ownership}
	}
	return results
}

func FormatRefactoringMainDev(results []RefactoringMainDevResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "main-dev", "removed", "total-removed", "ownership"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, r.MainDev,
			fmt.Sprint(r.Removed), fmt.Sprint(r.TotalRemoved),
			fmt.Sprintf("%.2f", r.Ownership),
		})
	}
	return out
}
