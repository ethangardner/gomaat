package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type EntityEffortResult struct {
	Entity     string
	Author     string
	AuthorRevs int
	TotalRevs  int
}

// EntityEffort returns revision count per (entity, author) pair.
func EntityEffort(commits []model.Commit, _ model.Options) []EntityEffortResult {
	authorRevs, totalRevs := revsPerEntityAuthor(commits)

	results := make([]EntityEffortResult, 0, len(authorRevs))
	for k, revs := range authorRevs {
		results = append(results, EntityEffortResult{
			Entity:     k.entity,
			Author:     k.author,
			AuthorRevs: revs,
			TotalRevs:  totalRevs[k.entity],
		})
	}
	slices.SortFunc(results, func(a, b EntityEffortResult) int {
		if c := cmp.Compare(a.Entity, b.Entity); c != 0 {
			return c
		}
		return cmp.Compare(b.AuthorRevs, a.AuthorRevs)
	})

	return results
}

func FormatEntityEffort(results []EntityEffortResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "author", "author-revs", "total-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, r.Author, fmt.Sprint(r.AuthorRevs), fmt.Sprint(r.TotalRevs)})
	}
	return out
}

// MainDevByRevs returns the author with the most revisions per entity,
// decay-weighted by opts.HalfLifeDays when set.
func MainDevByRevs(commits []model.Commit, opts model.Options) []ContributorResult {
	authorRevs, totalRevs := revsPerEntityAuthorForOpts(commits, opts)
	return pickTopContributor(authorRevs, totalRevs)
}

func FormatMainDevByRevs(results []ContributorResult, opts model.Options) [][]string {
	return formatContributor(results, opts, "revs", "total-revs")
}

// FragmentationResult.TotalRevs is float64 to accommodate a decay-weighted
// (--half-life) count; it holds a whole number when decay is disabled.
type FragmentationResult struct {
	Entity    string
	Fractal   float64
	TotalRevs float64
}

// Fragmentation calculates the fractal value (author distribution) per
// entity, decay-weighted by opts.HalfLifeDays when set.
// fractal = 1 - Σ(author_revs/total_revs)²
// 0 = single author, approaching 1 = many equal contributors.
func Fragmentation(commits []model.Commit, opts model.Options) []FragmentationResult {
	authorRevs, totalRevs := revsPerEntityAuthorForOpts(commits, opts)

	sumSqPerEntity := map[string]float64{}
	for k, revs := range authorRevs {
		ratio := revs / totalRevs[k.entity]
		sumSqPerEntity[k.entity] += ratio * ratio
	}

	results := make([]FragmentationResult, 0, len(sumSqPerEntity))
	for entity, sumSq := range sumSqPerEntity {
		fractal := math.Round((1.0-sumSq)*100) / 100
		results = append(results, FragmentationResult{entity, fractal, totalRevs[entity]})
	}
	slices.SortFunc(results, func(a, b FragmentationResult) int {
		if c := cmp.Compare(b.Fractal, a.Fractal); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})

	return results
}

func FormatFragmentation(results []FragmentationResult, opts model.Options) [][]string {
	out := [][]string{{"entity", "fractal-value", "total-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprintf("%.2f", r.Fractal), formatMetric(r.TotalRevs, opts)})
	}
	return out
}
