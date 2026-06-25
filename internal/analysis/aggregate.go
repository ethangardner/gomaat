package analysis

import (
	"cmp"
	"slices"

	"gomaat/internal/model"
)

type entityAuthorKey struct{ entity, author string }

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

type topContributorEntry struct {
	entity    string
	author    string
	count     int
	total     int
	ownership float64
}

// pickTopContributor selects, per entity, the author with the highest count
// from pre-computed per-(entity,author) counts and per-entity totals, and
// computes ownership %.
func pickTopContributor(byKey map[entityAuthorKey]int, totalByEntity map[string]int) []topContributorEntry {
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

	results := make([]topContributorEntry, 0, len(bestByEntity))
	for entity, b := range bestByEntity {
		total := totalByEntity[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(b.count) / float64(total) * 100.0
		}
		results = append(results, topContributorEntry{entity, b.author, b.count, total, ownership})
	}
	slices.SortFunc(results, func(a, b topContributorEntry) int { return cmp.Compare(a.entity, b.entity) })
	return results
}

// findTopContributor returns, per entity, the author with the highest value
// from valueFn, along with their count, the entity total, and ownership %.
func findTopContributor(commits []model.Commit, valueFn func(model.Commit) int) []topContributorEntry {
	byKey := map[entityAuthorKey]int{}
	totalByEntity := map[string]int{}
	for _, c := range commits {
		k := entityAuthorKey{c.Entity, c.Author}
		v := valueFn(c)
		byKey[k] += v
		totalByEntity[c.Entity] += v
	}
	return pickTopContributor(byKey, totalByEntity)
}
