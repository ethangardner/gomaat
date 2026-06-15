package analysis

import (
	"fmt"
	"sort"

	"gomaat/internal/model"
)

type AuthorsResult struct {
	Entity  string
	Authors int
	Revs    int
}

func Authors(commits []model.Commit, _ model.Options) []AuthorsResult {
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

	results := make([]AuthorsResult, 0, len(entities))
	for entity, e := range entities {
		results = append(results, AuthorsResult{entity, len(e.authors), len(e.revisions)})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Authors != results[j].Authors {
			return results[i].Authors > results[j].Authors
		}
		return results[i].Entity < results[j].Entity
	})

	return results
}

func FormatAuthors(results []AuthorsResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "n-authors", "n-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Authors), fmt.Sprint(r.Revs)})
	}
	return out
}
