package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"time"

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
// Used by MainDev, RefactoringMainDev, and MainDevByRevs. Count/Total are
// float64 so they can carry a decay-weighted (--half-life) value; they hold
// whole numbers when decay is disabled.
type ContributorResult struct {
	Entity      string
	Contributor string
	Count       float64
	Total       float64
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

// resolveNow returns opts.AgeTimeNow, defaulting to the current time when
// unset — the same fallback age.go uses for its own "now" reference, reused
// here as the decay reference point so --half-life and --age-time-now
// compose predictably instead of each having its own notion of "now".
func resolveNow(opts model.Options) time.Time {
	if opts.AgeTimeNow.IsZero() {
		return time.Now()
	}
	return opts.AgeTimeNow
}

// decayWeight returns the decay weight 0.5^(ageDays/halfLifeDays) for a
// commit dated dateStr relative to now. ok is false when dateStr can't be
// parsed, so callers can skip that commit's contribution rather than
// silently mis-weighting it.
func decayWeight(dateStr string, now time.Time, halfLifeDays float64) (weight float64, ok bool) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return 0, false
	}
	ageDays := now.Sub(t).Hours() / 24
	if ageDays < 0 {
		ageDays = 0
	}
	return math.Pow(0.5, ageDays/halfLifeDays), true
}

// commitWeight returns the decay weight for a commit dated dateStr, or 1.0
// when opts.HalfLifeDays is <= 0. ok is false when decay is enabled and
// dateStr cannot be parsed.
func commitWeight(dateStr string, now time.Time, opts model.Options) (float64, bool) {
	if opts.HalfLifeDays <= 0 {
		return 1.0, true
	}
	return decayWeight(dateStr, now, opts.HalfLifeDays)
}

// countDistinctWeighted is countDistinct's decay-aware counterpart: instead
// of counting each distinct value returned by valFn once, it sums a
// per-value decay weight computed from the (first-seen) commit that
// produced that value. Used when opts.HalfLifeDays > 0; countDistinct
// remains the plain, undecayed path.
func countDistinctWeighted[K comparable](commits []model.Commit, keyFn func(model.Commit) K, valFn func(model.Commit) string, now time.Time, halfLifeDays float64) map[K]float64 {
	dates := map[K]map[string]string{} // key -> distinct value -> its commit date
	for _, c := range commits {
		k := keyFn(c)
		if dates[k] == nil {
			dates[k] = map[string]string{}
		}
		v := valFn(c)
		if _, seen := dates[k][v]; !seen {
			dates[k][v] = c.Date
		}
	}

	weights := make(map[K]float64, len(dates))
	for k, vals := range dates {
		var sum float64
		for _, date := range vals {
			if w, ok := decayWeight(date, now, halfLifeDays); ok {
				sum += w
			}
		}
		weights[k] = sum
	}
	return weights
}

// toFloatMap converts an int-valued map (as returned by countDistinct) to
// its float64-valued equivalent, for merging with decay-weighted results
// that share the same map shape.
func toFloatMap[K comparable](m map[K]int) map[K]float64 {
	out := make(map[K]float64, len(m))
	for k, v := range m {
		out[k] = float64(v)
	}
	return out
}

// countDistinctForOpts counts distinct values per key returned by valFn and
// keyFn, decay-weighted when opts.HalfLifeDays > 0, or plain distinct counts
// (converted to float64) when decay is disabled.
func countDistinctForOpts[K comparable](commits []model.Commit, opts model.Options, keyFn func(model.Commit) K, valFn func(model.Commit) string) map[K]float64 {
	if opts.HalfLifeDays > 0 {
		return countDistinctWeighted(commits, keyFn, valFn, resolveNow(opts), opts.HalfLifeDays)
	}
	return toFloatMap(countDistinct(commits, keyFn, valFn))
}

// revsPerEntityAuthorForOpts returns per-(entity,author) and per-entity
// revision counts as float64: decay-weighted when opts.HalfLifeDays > 0,
// or plain distinct revision counts (weight 1 per revision) otherwise.
// Used by MainDevByRevs and Fragmentation, which are decay-aware; EntityEffort
// intentionally keeps calling the plain, always-undecayed revsPerEntityAuthor instead.
func revsPerEntityAuthorForOpts(commits []model.Commit, opts model.Options) (map[entityAuthorKey]float64, map[string]float64) {
	authorRevs := countDistinctForOpts(commits, opts,
		func(c model.Commit) entityAuthorKey { return entityAuthorKey{c.Entity, c.Author} },
		func(c model.Commit) string { return c.Rev })
	totalRevs := countDistinctForOpts(commits, opts,
		func(c model.Commit) string { return c.Entity },
		func(c model.Commit) string { return c.Rev })
	return authorRevs, totalRevs
}

// formatMetric renders a possibly decay-weighted metric: a plain integer
// string when decay is disabled (byte-identical to the pre-decay int-typed
// output), or two decimal places when --half-life is active.
func formatMetric(v float64, opts model.Options) string {
	if opts.HalfLifeDays <= 0 {
		return fmt.Sprint(int(v))
	}
	return fmt.Sprintf("%.2f", v)
}

// aggregateChurn sums LocAdded/LocDeleted and counts distinct revisions per
// key returned by keyFn.
func aggregateChurn(commits []model.Commit, keyFn func(model.Commit) string) []ChurnResult {
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

	results := make([]ChurnResult, 0, len(byKey))
	for k, e := range byKey {
		results = append(results, ChurnResult{k, e.added, e.deleted, len(e.revs)})
	}
	return results
}

// pickTopContributor selects, per entity, the author with the highest count
// from pre-computed per-(entity,author) counts and per-entity totals, and
// computes ownership %. Counts are float64 to accommodate decay-weighted
// (--half-life) callers; undecayed callers pass whole-number values.
func pickTopContributor(byKey map[entityAuthorKey]float64, totalByEntity map[string]float64) []ContributorResult {
	type best struct {
		author string
		count  float64
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
		var ownership float64
		if total > 0 {
			ownership = b.count / total * 100.0
		}
		results = append(results, ContributorResult{entity, b.author, b.count, total, ownership})
	}
	slices.SortFunc(results, func(a, b ContributorResult) int { return cmp.Compare(a.Entity, b.Entity) })
	return results
}

// findTopContributor returns, per entity, the author with the highest
// decay-weighted value from valueFn (weight 1 per commit when
// opts.HalfLifeDays is 0), along with their count, the entity total, and
// ownership %.
func findTopContributor(commits []model.Commit, opts model.Options, valueFn func(model.Commit) int) []ContributorResult {
	byKey := map[entityAuthorKey]float64{}
	totalByEntity := map[string]float64{}
	now := resolveNow(opts)
	for _, c := range commits {
		weight, ok := commitWeight(c.Date, now, opts)
		if !ok {
			continue
		}
		k := entityAuthorKey{c.Entity, c.Author}
		v := float64(valueFn(c)) * weight
		byKey[k] += v
		totalByEntity[c.Entity] += v
	}
	return pickTopContributor(byKey, totalByEntity)
}

// formatContributor renders ContributorResult rows to CSV, with caller-supplied
// column headers for the count and total columns.
func formatContributor(results []ContributorResult, opts model.Options, countHeader, totalHeader string) [][]string {
	out := [][]string{{"entity", "main-dev", countHeader, totalHeader, "ownership"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, r.Contributor,
			formatMetric(r.Count, opts), formatMetric(r.Total, opts),
			fmt.Sprintf("%.2f", r.Ownership),
		})
	}
	return out
}
