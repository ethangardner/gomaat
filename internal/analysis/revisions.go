package analysis

import (
	"fmt"
	"sort"

	"gomaat/internal/model"
)

func Revisions(commits []model.Commit, _ model.Options) [][]string {
	revsByEntity := map[string]map[string]struct{}{}
	for _, c := range commits {
		if _, ok := revsByEntity[c.Entity]; !ok {
			revsByEntity[c.Entity] = map[string]struct{}{}
		}
		revsByEntity[c.Entity][c.Rev] = struct{}{}
	}

	type row struct {
		entity string
		nRevs  int
	}
	rows := make([]row, 0, len(revsByEntity))
	for entity, revs := range revsByEntity {
		rows = append(rows, row{entity, len(revs)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].nRevs != rows[j].nRevs {
			return rows[i].nRevs > rows[j].nRevs
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "n-revs"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprint(r.nRevs)})
	}
	return out
}
