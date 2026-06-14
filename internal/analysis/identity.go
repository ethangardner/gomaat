package analysis

import (
	"fmt"

	"gomaat/internal/model"
)

func Identity(commits []model.Commit, _ model.Options) []model.Commit {
	return commits
}

func FormatIdentity(results []model.Commit, _ model.Options) [][]string {
	out := [][]string{{"entity", "rev", "date", "author", "loc-added", "loc-deleted"}}
	for _, c := range results {
		out = append(out, []string{
			c.Entity,
			c.Rev,
			c.Date,
			c.Author,
			fmt.Sprint(c.LocAdded),
			fmt.Sprint(c.LocDeleted),
		})
	}
	return out
}
