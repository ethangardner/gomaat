package analysis

import (
	"fmt"

	"gomaat/internal/model"
)

func Summary(commits []model.Commit, _ model.Options) [][]string {
	revs := map[string]struct{}{}
	entities := map[string]struct{}{}
	authors := map[string]struct{}{}

	for _, c := range commits {
		revs[c.Rev] = struct{}{}
		entities[c.Entity] = struct{}{}
		authors[c.Author] = struct{}{}
	}

	return [][]string{
		{"statistic", "value"},
		{"number-of-commits", fmt.Sprint(len(revs))},
		{"number-of-entities", fmt.Sprint(len(entities))},
		{"number-of-entities-changed", fmt.Sprint(len(commits))},
		{"number-of-authors", fmt.Sprint(len(authors))},
	}
}
