package analysis

import (
	"fmt"
	"math"
	"sort"

	"godemaat/internal/model"
)

// EntityEffort returns revision count per (entity, author) pair.
func EntityEffort(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	// count distinct revisions per (entity, author)
	revsByKey := map[key]map[string]struct{}{}
	for _, c := range commits {
		k := key{c.Entity, c.Author}
		if _, ok := revsByKey[k]; !ok {
			revsByKey[k] = map[string]struct{}{}
		}
		revsByKey[k][c.Rev] = struct{}{}
	}

	// total revisions per entity
	totalRevsByEntity := map[string]map[string]struct{}{}
	for _, c := range commits {
		if _, ok := totalRevsByEntity[c.Entity]; !ok {
			totalRevsByEntity[c.Entity] = map[string]struct{}{}
		}
		totalRevsByEntity[c.Entity][c.Rev] = struct{}{}
	}

	type row struct {
		entity     string
		author     string
		authorRevs int
		totalRevs  int
	}
	rows := make([]row, 0, len(revsByKey))
	for k, revs := range revsByKey {
		rows = append(rows, row{
			entity:     k.entity,
			author:     k.author,
			authorRevs: len(revs),
			totalRevs:  len(totalRevsByEntity[k.entity]),
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].entity != rows[j].entity {
			return rows[i].entity < rows[j].entity
		}
		return rows[i].authorRevs > rows[j].authorRevs
	})

	out := [][]string{{"entity", "author", "author-revs", "total-revs"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, r.author, fmt.Sprint(r.authorRevs), fmt.Sprint(r.totalRevs)})
	}
	return out
}

// MainDevByRevs returns the author with the most revisions per entity.
func MainDevByRevs(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	revsByKey := map[key]map[string]struct{}{}
	totalRevsByEntity := map[string]map[string]struct{}{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		if _, ok := revsByKey[k]; !ok {
			revsByKey[k] = map[string]struct{}{}
		}
		revsByKey[k][c.Rev] = struct{}{}
		if _, ok := totalRevsByEntity[c.Entity]; !ok {
			totalRevsByEntity[c.Entity] = map[string]struct{}{}
		}
		totalRevsByEntity[c.Entity][c.Rev] = struct{}{}
	}

	type bestEntry struct {
		author string
		revs   int
	}
	bestByEntity := map[string]bestEntry{}
	for k, revs := range revsByKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || len(revs) > cur.revs {
			bestByEntity[k.entity] = bestEntry{k.author, len(revs)}
		}
	}

	type row struct {
		entity     string
		mainDev    string
		added      int // "added" col = revisions from main dev
		totalAdded int
		ownership  float64
	}
	rows := make([]row, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := len(totalRevsByEntity[entity])
		var ownership float64
		if total > 0 {
			ownership = float64(best.revs) / float64(total) * 100.0
		}
		rows = append(rows, row{entity, best.author, best.revs, total, ownership})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].entity < rows[j].entity })

	out := [][]string{{"entity", "main-dev", "added", "total-added", "ownership"}}
	for _, r := range rows {
		out = append(out, []string{
			r.entity, r.mainDev,
			fmt.Sprint(r.added), fmt.Sprint(r.totalAdded),
			fmt.Sprintf("%.2f", r.ownership),
		})
	}
	return out
}

// Fragmentation calculates the fractal value (author distribution) per entity.
// fractal = 1 - Σ(author_revs/total_revs)²
// 0 = single author, approaching 1 = many equal contributors.
func Fragmentation(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	revsByKey := map[key]map[string]struct{}{}
	totalRevsByEntity := map[string]map[string]struct{}{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		if _, ok := revsByKey[k]; !ok {
			revsByKey[k] = map[string]struct{}{}
		}
		revsByKey[k][c.Rev] = struct{}{}
		if _, ok := totalRevsByEntity[c.Entity]; !ok {
			totalRevsByEntity[c.Entity] = map[string]struct{}{}
		}
		totalRevsByEntity[c.Entity][c.Rev] = struct{}{}
	}

	// collect authors per entity
	entityAuthors := map[string][]string{}
	for k := range revsByKey {
		entityAuthors[k.entity] = append(entityAuthors[k.entity], k.author)
	}

	type row struct {
		entity    string
		fractal   float64
		totalRevs int
	}
	rows := make([]row, 0, len(entityAuthors))
	for entity, authors := range entityAuthors {
		total := len(totalRevsByEntity[entity])
		var sumSq float64
		for _, author := range authors {
			authorRevs := len(revsByKey[key{entity, author}])
			ratio := float64(authorRevs) / float64(total)
			sumSq += ratio * ratio
		}
		fractal := 1.0 - sumSq
		fractal = math.Round(fractal*100) / 100
		rows = append(rows, row{entity, fractal, total})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].fractal != rows[j].fractal {
			return rows[i].fractal > rows[j].fractal
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "fractal-value", "total-revs"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprintf("%.2f", r.fractal), fmt.Sprint(r.totalRevs)})
	}
	return out
}
