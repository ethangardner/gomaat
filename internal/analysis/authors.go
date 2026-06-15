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
		authors map[string]struct{}
		revs    map[string]struct{}
	}
	byEntity := map[string]*entry{}

	for _, c := range commits {
		e, ok := byEntity[c.Entity]
		if !ok {
			e = &entry{
				authors: make(map[string]struct{}),
				revs:    make(map[string]struct{}),
			}
			byEntity[c.Entity] = e
		}
		e.authors[c.Author] = struct{}{}
		e.revs[c.Rev] = struct{}{}
	}

	results := make([]AuthorsResult, 0, len(byEntity))
	for entity, e := range byEntity {
		results = append(results, AuthorsResult{entity, len(e.authors), len(e.revs)})
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
