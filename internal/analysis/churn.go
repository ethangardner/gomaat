package analysis

import (
	"fmt"
	"sort"

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
	sort.Slice(aggs, func(i, j int) bool { return aggs[i].key < aggs[j].key })

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
	sort.Slice(aggs, func(i, j int) bool { return aggs[i].key < aggs[j].key })

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
	sort.Slice(aggs, func(i, j int) bool {
		if aggs[i].added != aggs[j].added {
			return aggs[i].added > aggs[j].added
		}
		return aggs[i].key < aggs[j].key
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
	sort.Slice(results, func(i, j int) bool {
		if results[i].Entity != results[j].Entity {
			return results[i].Entity < results[j].Entity
		}
		return results[i].Author < results[j].Author
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
	type key struct{ entity, author string }
	addedByKey := map[key]int{}
	totalAddedByEntity := map[string]int{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		addedByKey[k] += c.LocAdded
		totalAddedByEntity[c.Entity] += c.LocAdded
	}

	// find main dev per entity
	type bestEntry struct {
		author string
		added  int
	}
	bestByEntity := map[string]bestEntry{}
	for k, added := range addedByKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || added > cur.added {
			bestByEntity[k.entity] = bestEntry{k.author, added}
		}
	}

	results := make([]MainDevResult, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := totalAddedByEntity[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(best.added) / float64(total) * 100.0
		}
		results = append(results, MainDevResult{entity, best.author, best.added, total, ownership})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Entity < results[j].Entity })

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
	type key struct{ entity, author string }
	deletedByKey := map[key]int{}
	totalDeletedByEntity := map[string]int{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		deletedByKey[k] += c.LocDeleted
		totalDeletedByEntity[c.Entity] += c.LocDeleted
	}

	type bestEntry struct {
		author  string
		deleted int
	}
	bestByEntity := map[string]bestEntry{}
	for k, deleted := range deletedByKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || deleted > cur.deleted {
			bestByEntity[k.entity] = bestEntry{k.author, deleted}
		}
	}

	results := make([]RefactoringMainDevResult, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := totalDeletedByEntity[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(best.deleted) / float64(total) * 100.0
		}
		results = append(results, RefactoringMainDevResult{entity, best.author, best.deleted, total, ownership})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Entity < results[j].Entity })

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
