package analysis

import (
	"fmt"

	"github.com/ethangardner/gomaat/internal/model"
)

// Identity passes commits through unchanged, producing a raw revision listing.
func Identity(commits []model.Commit, _ model.Options) []model.Commit {
	return commits
}

func FormatIdentity(results []model.Commit, _ model.Options) [][]string {
	return formatRows([]string{"entity", "rev", "date", "author", "loc-added", "loc-deleted"}, results, func(c model.Commit) []string {
		return []string{
			c.Entity,
			c.Rev,
			c.Date,
			c.Author,
			fmt.Sprint(c.LocAdded),
			fmt.Sprint(c.LocDeleted),
		}
	})
}
