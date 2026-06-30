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

// MainDevByRevs returns the author with the most revisions per entity.
func MainDevByRevs(commits []model.Commit, _ model.Options) []ContributorResult {
	authorRevs, totalRevs := revsPerEntityAuthor(commits)
	return pickTopContributor(authorRevs, totalRevs)
}

func FormatMainDevByRevs(results []ContributorResult, _ model.Options) [][]string {
	return formatContributor(results, "revs", "total-revs")
}

type FragmentationResult struct {
	Entity    string
	Fractal   float64
	TotalRevs int
}

// Fragmentation calculates the fractal value (author distribution) per entity.
// fractal = 1 - Σ(author_revs/total_revs)²
// 0 = single author, approaching 1 = many equal contributors.
func Fragmentation(commits []model.Commit, _ model.Options) []FragmentationResult {
	authorRevs, totalRevs := revsPerEntityAuthor(commits)

	sumSqPerEntity := map[string]float64{}
	for k, revs := range authorRevs {
		ratio := float64(revs) / float64(totalRevs[k.entity])
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

func FormatFragmentation(results []FragmentationResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "fractal-value", "total-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprintf("%.2f", r.Fractal), fmt.Sprint(r.TotalRevs)})
	}
	return out
}
