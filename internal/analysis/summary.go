package analysis

import (
	"fmt"

	"gomaat/internal/model"
)

type SummaryResult struct {
	Commits         int
	Entities        int
	EntitiesChanged int
	Authors         int
}

func Summary(commits []model.Commit, _ model.Options) SummaryResult {
	revs := map[string]struct{}{}
	entities := map[string]struct{}{}
	authors := map[string]struct{}{}

	for _, c := range commits {
		revs[c.Rev] = struct{}{}
		entities[c.Entity] = struct{}{}
		authors[c.Author] = struct{}{}
	}

	return SummaryResult{
		Commits:         len(revs),
		Entities:        len(entities),
		EntitiesChanged: len(commits),
		Authors:         len(authors),
	}
}

func FormatSummary(result SummaryResult, _ model.Options) [][]string {
	return [][]string{
		{"statistic", "value"},
		{"number-of-commits", fmt.Sprint(result.Commits)},
		{"number-of-entities", fmt.Sprint(result.Entities)},
		{"number-of-entities-changed", fmt.Sprint(result.EntitiesChanged)},
		{"number-of-authors", fmt.Sprint(result.Authors)},
	}
}
