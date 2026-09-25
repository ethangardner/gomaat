package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type entityAuthorKey struct{ entity, author string }

// ChurnResult holds added/deleted line totals and distinct commit count for one key.
// Used by AbsChurn (key=date), AuthorChurn (key=author), and EntityChurn (key=entity).
type ChurnResult struct {
	Key     string
	Added   int
	Deleted int
	Commits int
}

// ContributorResult holds the top contributor per entity and their ownership percentage.
// Used by MainDev, RefactoringMainDev, and MainDevByRevs.
type ContributorResult struct {
	Entity      string
	Contributor string
	Count       int
	Total       int
	Ownership   float64
}

// countDistinct counts, for each key returned by keyFn, the number of
// distinct values returned by valFn across commits.
func countDistinct[K comparable](commits []model.Commit, keyFn func(model.Commit) K, valFn func(model.Commit) string) map[K]int {
	sets := map[K]map[string]struct{}{}
	for _, c := range commits {
		k := keyFn(c)
		if sets[k] == nil {
			sets[k] = map[string]struct{}{}
		}
		sets[k][valFn(c)] = struct{}{}
	}

	counts := make(map[K]int, len(sets))
	for k, s := range sets {
		counts[k] = len(s)
	}
	return counts
}

// revsPerEntityAuthor returns per-(entity,author) distinct revision counts and
// per-entity distinct revision counts, computed in one pass each via countDistinct.
func revsPerEntityAuthor(commits []model.Commit) (map[entityAuthorKey]int, map[string]int) {
	authorRevs := countDistinct(commits,
		func(c model.Commit) entityAuthorKey { return entityAuthorKey{c.Entity, c.Author} },
		func(c model.Commit) string { return c.Rev })
	totalRevs := countDistinct(commits,
		func(c model.Commit) string { return c.Entity },
		func(c model.Commit) string { return c.Rev })
	return authorRevs, totalRevs
}

// churnAgg holds added/deleted line totals and a distinct revision count for
// one key, shared by AbsChurn, AuthorChurn, and EntityChurn.
type churnAgg struct {
	key     string
	added   int
	deleted int
	commits int
}

// aggregateChurn sums LocAdded/LocDeleted and counts distinct revisions per
// key returned by keyFn.
func aggregateChurn(commits []model.Commit, keyFn func(model.Commit) string) []churnAgg {
	type entry struct {
		added, deleted int
		revs           map[string]struct{}
	}
	byKey := map[string]*entry{}
	for _, c := range commits {
		k := keyFn(c)
		e, ok := byKey[k]
		if !ok {
			e = &entry{revs: map[string]struct{}{}}
			byKey[k] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
		e.revs[c.Rev] = struct{}{}
	}

	aggs := make([]churnAgg, 0, len(byKey))
	for k, e := range byKey {
		aggs = append(aggs, churnAgg{k, e.added, e.deleted, len(e.revs)})
	}
	return aggs
}

func aggsToChurnResults(aggs []churnAgg) []ChurnResult {
	results := make([]ChurnResult, len(aggs))
	for i, a := range aggs {
		results[i] = ChurnResult{a.key, a.added, a.deleted, a.commits}
	}
	return results
}

// pickTopContributor selects, per entity, the author with the highest count
// from pre-computed per-(entity,author) counts and per-entity totals, and
// computes ownership %.
func pickTopContributor(byKey map[entityAuthorKey]int, totalByEntity map[string]int) []ContributorResult {
	type best struct {
		author string
		count  int
	}
	bestByEntity := map[string]best{}
	for k, count := range byKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || count > cur.count || (count == cur.count && k.author < cur.author) {
			bestByEntity[k.entity] = best{k.author, count}
		}
	}

	results := make([]ContributorResult, 0, len(bestByEntity))
	for entity, b := range bestByEntity {
		total := totalByEntity[entity]
		results = append(results, ContributorResult{entity, b.author, b.count, total, percent(b.count, total)})
	}
	slices.SortFunc(results, func(a, b ContributorResult) int { return cmp.Compare(a.Entity, b.Entity) })
	return results
}

// sumPerEntityAuthor sums valueFn per (entity, author) pair and per entity.
func sumPerEntityAuthor(commits []model.Commit, valueFn func(model.Commit) int) (map[entityAuthorKey]int, map[string]int) {
	byKey := map[entityAuthorKey]int{}
	totalByEntity := map[string]int{}
	for _, c := range commits {
		v := valueFn(c)
		byKey[entityAuthorKey{c.Entity, c.Author}] += v
		totalByEntity[c.Entity] += v
	}
	return byKey, totalByEntity
}

// percent returns part/total as a percentage, or 0 when total is 0.
func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}

// linesAdded is the ownership measure shared by MainDev, BusFactor, and
// KnowledgeLoss: an author owns the lines they added.
func linesAdded(c model.Commit) int { return c.LocAdded }

// findTopContributor returns, per entity, the author with the highest value
// from valueFn, along with their count, the entity total, and ownership %.
func findTopContributor(commits []model.Commit, valueFn func(model.Commit) int) []ContributorResult {
	return pickTopContributor(sumPerEntityAuthor(commits, valueFn))
}

// formatRows renders a header row and data rows produced by rowFn for each item.
func formatRows[T any](header []string, items []T, rowFn func(T) []string) [][]string {
	out := make([][]string, len(items)+1)
	out[0] = header
	for i, item := range items {
		out[i+1] = rowFn(item)
	}
	return out
}

// formatContributor renders ContributorResult rows to CSV, with caller-supplied
// column headers for the count and total columns.
func formatContributor(results []ContributorResult, countHeader, totalHeader string) [][]string {
	return formatRows([]string{"entity", "main-dev", countHeader, totalHeader, "ownership"}, results, func(r ContributorResult) []string {
		return []string{
			r.Entity, r.Contributor,
			fmt.Sprint(r.Count), fmt.Sprint(r.Total),
			fmt.Sprintf("%.2f", r.Ownership),
		}
	})
}
