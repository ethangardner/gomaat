package analysis

import (
	"fmt"
	"sort"

	"godemaat/internal/model"
)

func Authors(commits []model.Commit, _ model.Options) [][]string {
	type entry struct {
		authors   map[string]struct{}
		revisions map[string]struct{}
	}
	entities := map[string]*entry{}

	for _, c := range commits {
		e, ok := entities[c.Entity]
		if !ok {
			e = &entry{
				authors:   map[string]struct{}{},
				revisions: map[string]struct{}{},
			}
			entities[c.Entity] = e
		}
		e.authors[c.Author] = struct{}{}
		e.revisions[c.Rev] = struct{}{}
	}

	type row struct {
		entity   string
		nAuthors int
		nRevs    int
	}
	rows := make([]row, 0, len(entities))
	for entity, e := range entities {
		rows = append(rows, row{entity, len(e.authors), len(e.revisions)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].nAuthors != rows[j].nAuthors {
			return rows[i].nAuthors > rows[j].nAuthors
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "n-authors", "n-revs"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprint(r.nAuthors), fmt.Sprint(r.nRevs)})
	}
	return out
}
