package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"gomaat/internal/model"
)

type EntityEffortResult struct {
	Entity     string
	Author     string
	AuthorRevs int
	TotalRevs  int
}

// EntityEffort returns revision count per (entity, author) pair.
func EntityEffort(commits []model.Commit, _ model.Options) []EntityEffortResult {
	type key struct{ entity, author string }
	// count distinct revisions per (entity, author)
	authorRevs := countDistinct(commits, func(c model.Commit) key { return key{c.Entity, c.Author} }, func(c model.Commit) string { return c.Rev })

	// total revisions per entity
	totalRevs := countDistinct(commits, func(c model.Commit) string { return c.Entity }, func(c model.Commit) string { return c.Rev })

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

type MainDevByRevsResult struct {
	Entity    string
	MainDev   string
	Added     int // revisions from main dev
	TotalRevs int
	Ownership float64
}

// MainDevByRevs returns the author with the most revisions per entity.
func MainDevByRevs(commits []model.Commit, _ model.Options) []MainDevByRevsResult {
	type key struct{ entity, author string }
	authorRevs := countDistinct(commits, func(c model.Commit) key { return key{c.Entity, c.Author} }, func(c model.Commit) string { return c.Rev })
	totalRevs := countDistinct(commits, func(c model.Commit) string { return c.Entity }, func(c model.Commit) string { return c.Rev })

	type bestEntry struct {
		author string
		revs   int
	}
	bestByEntity := map[string]bestEntry{}
	for k, revs := range authorRevs {
		cur, ok := bestByEntity[k.entity]
		if !ok || revs > cur.revs {
			bestByEntity[k.entity] = bestEntry{k.author, revs}
		}
	}

	results := make([]MainDevByRevsResult, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := totalRevs[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(best.revs) / float64(total) * 100.0
		}
		results = append(results, MainDevByRevsResult{entity, best.author, best.revs, total, ownership})
	}
	slices.SortFunc(results, func(a, b MainDevByRevsResult) int { return cmp.Compare(a.Entity, b.Entity) })

	return results
}

func FormatMainDevByRevs(results []MainDevByRevsResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "main-dev", "added", "total-added", "ownership"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity, r.MainDev,
			fmt.Sprint(r.Added), fmt.Sprint(r.TotalRevs),
			fmt.Sprintf("%.2f", r.Ownership),
		})
	}
	return out
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
	type key struct{ entity, author string }
	authorRevs := countDistinct(commits, func(c model.Commit) key { return key{c.Entity, c.Author} }, func(c model.Commit) string { return c.Rev })
	totalRevs := countDistinct(commits, func(c model.Commit) string { return c.Entity }, func(c model.Commit) string { return c.Rev })

	// collect authors per entity
	entityAuthors := map[string][]string{}
	for k := range authorRevs {
		entityAuthors[k.entity] = append(entityAuthors[k.entity], k.author)
	}

	results := make([]FragmentationResult, 0, len(entityAuthors))
	for entity, authors := range entityAuthors {
		total := totalRevs[entity]
		var sumSq float64
		if total > 0 {
			for _, author := range authors {
				ratio := float64(authorRevs[key{entity, author}]) / float64(total)
				sumSq += ratio * ratio
			}
		}
		fractal := 1.0 - sumSq
		fractal = math.Round(fractal*100) / 100
		results = append(results, FragmentationResult{entity, fractal, total})
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
