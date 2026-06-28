package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"gomaat/internal/model"
)

type CouplingResult struct {
	Entity  string
	Coupled string
	Degree  int
	AvgRevs int
	RevA    int
	RevB    int
	Shared  int
}

// Coupling detects modules that tend to change together (temporal coupling).
func Coupling(commits []model.Commit, opts model.Options) []CouplingResult {
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
		for i := range len(dedupedEntities) {
			for j := i + 1; j < len(dedupedEntities); j++ {
				a, b := dedupedEntities[i], dedupedEntities[j]
				if a > b {
					a, b = b, a
				}
				pairShared[a+"\x00"+b]++
			}
		}
	}

	var results []CouplingResult
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

		results = append(results, CouplingResult{
			Entity:  a,
			Coupled: b,
			Degree:  int(degree),
			AvgRevs: int(math.Ceil(avg)),
			RevA:    revsA,
			RevB:    revsB,
			Shared:  shared,
		})
	}

	slices.SortFunc(results, func(a, b CouplingResult) int {
		if c := cmp.Compare(b.Degree, a.Degree); c != 0 {
			return c
		}
		return cmp.Compare(b.AvgRevs, a.AvgRevs)
	})

	return results
}

func FormatCoupling(results []CouplingResult, opts model.Options) [][]string {
	var headers []string
	if opts.VerboseResults {
		headers = []string{"entity", "coupled", "degree", "average-revs", "first-entity-revisions", "second-entity-revisions", "shared-revisions"}
	} else {
		headers = []string{"entity", "coupled", "degree", "average-revs"}
	}

	out := [][]string{headers}
	for _, r := range results {
		if opts.VerboseResults {
			out = append(out, []string{
				r.Entity, r.Coupled,
				fmt.Sprint(r.Degree), fmt.Sprint(r.AvgRevs),
				fmt.Sprint(r.RevA), fmt.Sprint(r.RevB), fmt.Sprint(r.Shared),
			})
		} else {
			out = append(out, []string{r.Entity, r.Coupled, fmt.Sprint(r.Degree), fmt.Sprint(r.AvgRevs)})
		}
	}
	return out
}

type SumOfCouplingResult struct {
	Entity string
	Soc    int
}

// SumOfCoupling aggregates coupling counts per entity (how many co-changes it participates in).
func SumOfCoupling(commits []model.Commit, opts model.Options) []SumOfCouplingResult {
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

	results := make([]SumOfCouplingResult, 0, len(soc))
	for entity, count := range soc {
		results = append(results, SumOfCouplingResult{entity, count})
	}
	slices.SortFunc(results, func(a, b SumOfCouplingResult) int {
		if c := cmp.Compare(b.Soc, a.Soc); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})

	return results
}

func FormatSumOfCoupling(results []SumOfCouplingResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "soc"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Soc)})
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
	a, b, _ := strings.Cut(key, "\x00")
	return [2]string{a, b}
}
