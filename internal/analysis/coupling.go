package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
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

type pairKey struct{ a, b string }

// Coupling detects modules that tend to change together (temporal coupling).
func Coupling(commits []model.Commit, opts model.Options) []CouplingResult {
	pairShared := map[pairKey]int{}
	moduleRevs := map[string]int{}

	for _, entities := range filteredChangesets(commits, opts) {
		for i, a := range entities {
			moduleRevs[a]++
			for _, b := range entities[i+1:] {
				pairShared[pairKey{a, b}]++
			}
		}
	}

	var results []CouplingResult
	for key, shared := range pairShared {
		revsA, revsB := moduleRevs[key.a], moduleRevs[key.b]
		avg := (float64(revsA) + float64(revsB)) / 2.0
		degree := (float64(shared) / avg) * 100.0

		if avg < float64(opts.MinRevs) ||
			shared < opts.MinSharedRevs ||
			degree < opts.MinCoupling ||
			math.Floor(degree) > opts.MaxCoupling {
			continue
		}

		results = append(results, CouplingResult{
			Entity:  key.a,
			Coupled: key.b,
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
	headers := []string{"entity", "coupled", "degree", "average-revs"}
	if opts.VerboseResults {
		headers = append(headers, "first-entity-revisions", "second-entity-revisions", "shared-revisions")
	}

	out := make([][]string, 0, len(results)+1)
	out = append(out, headers)
	for _, r := range results {
		row := []string{r.Entity, r.Coupled, fmt.Sprint(r.Degree), fmt.Sprint(r.AvgRevs)}
		if opts.VerboseResults {
			row = append(row, fmt.Sprint(r.RevA), fmt.Sprint(r.RevB), fmt.Sprint(r.Shared))
		}
		out = append(out, row)
	}
	return out
}

type SumOfCouplingResult struct {
	Entity string
	Soc    int
}

// SumOfCoupling aggregates coupling counts per entity (how many co-changes it participates in).
func SumOfCoupling(commits []model.Commit, opts model.Options) []SumOfCouplingResult {
	soc := map[string]int{}
	for _, dedupedEntities := range filteredChangesets(commits, opts) {
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

func filteredChangesets(commits []model.Commit, opts model.Options) [][]string {
	revEntities := map[string][]string{}
	for _, c := range commits {
		revEntities[c.Rev] = append(revEntities[c.Rev], c.Entity)
	}
	var out [][]string
	for _, entities := range revEntities {
		deduped := dedupe(entities)
		if len(deduped) > opts.MaxChangesetSize {
			continue
		}
		out = append(out, deduped)
	}
	return out
}

func dedupe(ss []string) []string {
	slices.Sort(ss)
	return slices.Compact(ss)
}
