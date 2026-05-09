package analysis

import (
	"fmt"
	"math"
	"sort"

	"godemaat/internal/model"
)

// Coupling detects modules that tend to change together (temporal coupling).
func Coupling(commits []model.Commit, opts model.Options) [][]string {
	// group entities by revision
	revEntities := map[string][]string{}
	for _, c := range commits {
		revEntities[c.Rev] = append(revEntities[c.Rev], c.Entity)
	}

	// count co-occurrences and per-module revision totals
	// pairKey: canonical sorted pair "A\x00B"
	pairShared := map[string]int{}
	moduleRevs := map[string]int{}

	for _, entities := range revEntities {
		dedupedEntities := dedupe(entities)
		if len(dedupedEntities) > opts.MaxChangesetSize {
			continue
		}
		// track each module's appearance (for total revisions)
		seenInRev := map[string]struct{}{}
		for _, e := range dedupedEntities {
			if _, seen := seenInRev[e]; !seen {
				moduleRevs[e]++
				seenInRev[e] = struct{}{}
			}
		}
		// generate all pairs
		for i := 0; i < len(dedupedEntities); i++ {
			for j := i + 1; j < len(dedupedEntities); j++ {
				a, b := dedupedEntities[i], dedupedEntities[j]
				if a > b {
					a, b = b, a
				}
				pairShared[a+"\x00"+b]++
			}
		}
	}

	type couplingRow struct {
		entity  string
		coupled string
		degree  int
		avgRevs int
		revA    int
		revB    int
		shared  int
	}

	var rows []couplingRow
	for key, shared := range pairShared {
		parts := splitPairKey(key)
		a, b := parts[0], parts[1]
		revsA := moduleRevs[a]
		revsB := moduleRevs[b]
		avg := (float64(revsA) + float64(revsB)) / 2.0
		degree := (float64(shared) / avg) * 100.0

		if avg < float64(opts.MinRevs) {
			continue
		}
		if shared < opts.MinSharedRevs {
			continue
		}
		if degree < opts.MinCoupling {
			continue
		}
		if math.Floor(degree) > opts.MaxCoupling {
			continue
		}

		rows = append(rows, couplingRow{
			entity:  a,
			coupled: b,
			degree:  int(degree),
			avgRevs: int(math.Ceil(avg)),
			revA:    revsA,
			revB:    revsB,
			shared:  shared,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].degree != rows[j].degree {
			return rows[i].degree > rows[j].degree
		}
		return rows[i].avgRevs > rows[j].avgRevs
	})

	var headers []string
	if opts.VerboseResults {
		headers = []string{"entity", "coupled", "degree", "average-revs", "first-entity-revisions", "second-entity-revisions", "shared-revisions"}
	} else {
		headers = []string{"entity", "coupled", "degree", "average-revs"}
	}

	out := [][]string{headers}
	for _, r := range rows {
		if opts.VerboseResults {
			out = append(out, []string{
				r.entity, r.coupled,
				fmt.Sprint(r.degree), fmt.Sprint(r.avgRevs),
				fmt.Sprint(r.revA), fmt.Sprint(r.revB), fmt.Sprint(r.shared),
			})
		} else {
			out = append(out, []string{r.entity, r.coupled, fmt.Sprint(r.degree), fmt.Sprint(r.avgRevs)})
		}
	}
	return out
}

// SumOfCoupling aggregates coupling counts per entity (how many co-changes it participates in).
func SumOfCoupling(commits []model.Commit, opts model.Options) [][]string {
	revEntities := map[string][]string{}
	for _, c := range commits {
		revEntities[c.Rev] = append(revEntities[c.Rev], c.Entity)
	}

	soc := map[string]int{}
	for _, entities := range revEntities {
		dedupedEntities := dedupe(entities)
		if len(dedupedEntities) > opts.MaxChangesetSize {
			continue
		}
		for _, e := range dedupedEntities {
			soc[e] += len(dedupedEntities) - 1
		}
	}

	type row struct {
		entity string
		soc    int
	}
	rows := make([]row, 0, len(soc))
	for entity, count := range soc {
		rows = append(rows, row{entity, count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].soc != rows[j].soc {
			return rows[i].soc > rows[j].soc
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "soc"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprint(r.soc)})
	}
	return out
}

func dedupe(ss []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}

func splitPairKey(key string) [2]string {
	idx := 0
	for i, b := range key {
		if b == '\x00' {
			idx = i
			break
		}
	}
	return [2]string{key[:idx], key[idx+1:]}
}
