package analysis

import (
	"fmt"
	"sort"

	"godemaat/internal/model"
)

// AbsChurn returns lines added/deleted aggregated by date.
func AbsChurn(commits []model.Commit, _ model.Options) [][]string {
	type entry struct {
		added   int
		deleted int
		commits map[string]struct{}
	}
	byDate := map[string]*entry{}
	for _, c := range commits {
		e, ok := byDate[c.Date]
		if !ok {
			e = &entry{commits: map[string]struct{}{}}
			byDate[c.Date] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
		e.commits[c.Rev] = struct{}{}
	}

	type row struct {
		date                    string
		added, deleted, commits int
	}
	rows := make([]row, 0, len(byDate))
	for date, e := range byDate {
		rows = append(rows, row{date, e.added, e.deleted, len(e.commits)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].date < rows[j].date })

	out := [][]string{{"date", "added", "deleted", "commits"}}
	for _, r := range rows {
		out = append(out, []string{r.date, fmt.Sprint(r.added), fmt.Sprint(r.deleted), fmt.Sprint(r.commits)})
	}
	return out
}

// AuthorChurn returns lines added/deleted aggregated by author.
func AuthorChurn(commits []model.Commit, _ model.Options) [][]string {
	type entry struct {
		added   int
		deleted int
		commits map[string]struct{}
	}
	byAuthor := map[string]*entry{}
	for _, c := range commits {
		e, ok := byAuthor[c.Author]
		if !ok {
			e = &entry{commits: map[string]struct{}{}}
			byAuthor[c.Author] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
		e.commits[c.Rev] = struct{}{}
	}

	type row struct {
		author                  string
		added, deleted, commits int
	}
	rows := make([]row, 0, len(byAuthor))
	for author, e := range byAuthor {
		rows = append(rows, row{author, e.added, e.deleted, len(e.commits)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].author < rows[j].author })

	out := [][]string{{"author", "added", "deleted", "commits"}}
	for _, r := range rows {
		out = append(out, []string{r.author, fmt.Sprint(r.added), fmt.Sprint(r.deleted), fmt.Sprint(r.commits)})
	}
	return out
}

// EntityChurn returns lines added/deleted aggregated by entity, sorted by added desc.
func EntityChurn(commits []model.Commit, _ model.Options) [][]string {
	type entry struct {
		added   int
		deleted int
		commits map[string]struct{}
	}
	byEntity := map[string]*entry{}
	for _, c := range commits {
		e, ok := byEntity[c.Entity]
		if !ok {
			e = &entry{commits: map[string]struct{}{}}
			byEntity[c.Entity] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
		e.commits[c.Rev] = struct{}{}
	}

	type row struct {
		entity                  string
		added, deleted, commits int
	}
	rows := make([]row, 0, len(byEntity))
	for entity, e := range byEntity {
		rows = append(rows, row{entity, e.added, e.deleted, len(e.commits)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].added != rows[j].added {
			return rows[i].added > rows[j].added
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "added", "deleted", "commits"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprint(r.added), fmt.Sprint(r.deleted), fmt.Sprint(r.commits)})
	}
	return out
}

// EntityOwnership returns churn per (entity, author) pair.
func EntityOwnership(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	type entry struct{ added, deleted int }
	byKey := map[key]*entry{}
	for _, c := range commits {
		k := key{c.Entity, c.Author}
		e, ok := byKey[k]
		if !ok {
			e = &entry{}
			byKey[k] = e
		}
		e.added += c.LocAdded
		e.deleted += c.LocDeleted
	}

	type row struct {
		entity, author string
		added, deleted int
	}
	rows := make([]row, 0, len(byKey))
	for k, e := range byKey {
		rows = append(rows, row{k.entity, k.author, e.added, e.deleted})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].entity != rows[j].entity {
			return rows[i].entity < rows[j].entity
		}
		return rows[i].author < rows[j].author
	})

	out := [][]string{{"entity", "author", "added", "deleted"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, r.author, fmt.Sprint(r.added), fmt.Sprint(r.deleted)})
	}
	return out
}

// MainDev returns the author with the most lines added per entity.
func MainDev(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	addedByKey := map[key]int{}
	totalAddedByEntity := map[string]int{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		addedByKey[k] += c.LocAdded
		totalAddedByEntity[c.Entity] += c.LocAdded
	}

	// find main dev per entity
	type bestEntry struct {
		author string
		added  int
	}
	bestByEntity := map[string]bestEntry{}
	for k, added := range addedByKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || added > cur.added {
			bestByEntity[k.entity] = bestEntry{k.author, added}
		}
	}

	type row struct {
		entity     string
		mainDev    string
		added      int
		totalAdded int
		ownership  float64
	}
	rows := make([]row, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := totalAddedByEntity[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(best.added) / float64(total) * 100.0
		}
		rows = append(rows, row{entity, best.author, best.added, total, ownership})
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

// RefactoringMainDev returns the author with the most lines deleted per entity.
func RefactoringMainDev(commits []model.Commit, _ model.Options) [][]string {
	type key struct{ entity, author string }
	deletedByKey := map[key]int{}
	totalDeletedByEntity := map[string]int{}

	for _, c := range commits {
		k := key{c.Entity, c.Author}
		deletedByKey[k] += c.LocDeleted
		totalDeletedByEntity[c.Entity] += c.LocDeleted
	}

	type bestEntry struct {
		author  string
		deleted int
	}
	bestByEntity := map[string]bestEntry{}
	for k, deleted := range deletedByKey {
		cur, ok := bestByEntity[k.entity]
		if !ok || deleted > cur.deleted {
			bestByEntity[k.entity] = bestEntry{k.author, deleted}
		}
	}

	type row struct {
		entity       string
		mainDev      string
		removed      int
		totalRemoved int
		ownership    float64
	}
	rows := make([]row, 0, len(bestByEntity))
	for entity, best := range bestByEntity {
		total := totalDeletedByEntity[entity]
		var ownership float64
		if total > 0 {
			ownership = float64(best.deleted) / float64(total) * 100.0
		}
		rows = append(rows, row{entity, best.author, best.deleted, total, ownership})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].entity < rows[j].entity })

	out := [][]string{{"entity", "main-dev", "removed", "total-removed", "ownership"}}
	for _, r := range rows {
		out = append(out, []string{
			r.entity, r.mainDev,
			fmt.Sprint(r.removed), fmt.Sprint(r.totalRemoved),
			fmt.Sprintf("%.2f", r.ownership),
		})
	}
	return out
}
