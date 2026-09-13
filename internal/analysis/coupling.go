package analysis

import (
	"cmp"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

// CouplingResult's numeric fields are float64 to accommodate decay-weighted
// (--half-life) revision counts; they hold whole numbers when decay is
// disabled.
type CouplingResult struct {
	Entity  string
	Coupled string
	Degree  float64
	AvgRevs float64
	RevA    float64
	RevB    float64
	Shared  float64
}

type pairKey struct{ a, b string }

// Coupling detects modules that tend to change together (temporal coupling).
func Coupling(commits []model.Commit, opts model.Options) []CouplingResult {
	pairShared := map[pairKey]float64{}
	moduleRevs := map[string]float64{}
	now := resolveNow(opts)

	for _, cs := range filteredChangesets(commits, opts) {
		weight, ok := commitWeight(cs.Date, now, opts)
		if !ok {
			continue
		}
		for i, a := range cs.Entities {
			moduleRevs[a] += weight
			for _, b := range cs.Entities[i+1:] {
				pairShared[pairKey{a, b}] += weight
			}
		}
	}

	var results []CouplingResult
	for key, shared := range pairShared {
		revsA, revsB := moduleRevs[key.a], moduleRevs[key.b]
		avg := (revsA + revsB) / 2.0
		degree := (shared / avg) * 100.0

		if avg < float64(opts.MinRevs) ||
			shared < float64(opts.MinSharedRevs) ||
			degree < opts.MinCoupling ||
			math.Floor(degree) > opts.MaxCoupling {
			continue
		}

		results = append(results, CouplingResult{
			Entity:  key.a,
			Coupled: key.b,
			Degree:  math.Floor(degree),
			AvgRevs: math.Ceil(avg),
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
		row := []string{r.Entity, r.Coupled, formatMetric(r.Degree, opts), formatMetric(r.AvgRevs, opts)}
		if opts.VerboseResults {
			row = append(row, formatMetric(r.RevA, opts), formatMetric(r.RevB, opts), formatMetric(r.Shared, opts))
		}
		out = append(out, row)
	}
	return out
}

// SumOfCouplingResult.Soc is float64 to accommodate a decay-weighted
// (--half-life) count; it holds a whole number when decay is disabled.
type SumOfCouplingResult struct {
	Entity string
	Soc    float64
}

// SumOfCoupling aggregates coupling counts per entity (how many co-changes it participates in).
func SumOfCoupling(commits []model.Commit, opts model.Options) []SumOfCouplingResult {
	soc := map[string]float64{}
	now := resolveNow(opts)
	for _, cs := range filteredChangesets(commits, opts) {
		weight, ok := commitWeight(cs.Date, now, opts)
		if !ok {
			continue
		}
		for _, e := range cs.Entities {
			soc[e] += float64(len(cs.Entities)-1) * weight
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

func FormatSumOfCoupling(results []SumOfCouplingResult, opts model.Options) [][]string {
	out := [][]string{{"entity", "soc"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, formatMetric(r.Soc, opts)})
	}
	return out
}

// changeset is one commit's deduplicated entity list, with the commit's own
// date attached so callers can decay-weight the whole changeset consistently
// (a shared revision between two files uses the same per-commit weight).
type changeset struct {
	Date     string
	Entities []string
}

func filteredChangesets(commits []model.Commit, opts model.Options) []changeset {
	type revData struct {
		date     string
		entities []string
	}
	revEntities := map[string]*revData{}
	for _, c := range commits {
		rd, ok := revEntities[c.Rev]
		if !ok {
			rd = &revData{date: c.Date}
			revEntities[c.Rev] = rd
		}
		rd.entities = append(rd.entities, c.Entity)
	}
	var out []changeset
	for _, rd := range revEntities {
		deduped := dedupe(rd.entities)
		if len(deduped) > opts.MaxChangesetSize {
			continue
		}
		out = append(out, changeset{Date: rd.date, Entities: deduped})
	}
	return out
}

func dedupe(ss []string) []string {
	slices.Sort(ss)
	return slices.Compact(ss)
}
