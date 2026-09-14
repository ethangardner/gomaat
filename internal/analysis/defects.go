package analysis

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type DefectsResult struct {
	Entity          string
	BugfixRevisions int
	TotalRevisions  int
	DefectRatio     float64
}

// Defects classifies commits via isBugfix, then reports defect density
// (bugfix revisions vs. total revisions) per entity, reusing Revisions for
// both counts rather than reimplementing revision counting.
func Defects(commits []model.Commit, isBugfix func(model.Commit) bool, opts model.Options) []DefectsResult {
	total := Revisions(commits, opts)

	bugfixCommits := make([]model.Commit, 0, len(commits))
	for _, c := range commits {
		if isBugfix(c) {
			bugfixCommits = append(bugfixCommits, c)
		}
	}
	bugfixRevsByEntity := make(map[string]int, len(total))
	for _, r := range Revisions(bugfixCommits, opts) {
		bugfixRevsByEntity[r.Entity] = r.Revs
	}

	results := make([]DefectsResult, 0, len(total))
	for _, r := range total {
		bugfixRevs := bugfixRevsByEntity[r.Entity]
		var ratio float64
		if r.Revs > 0 {
			ratio = float64(bugfixRevs) / float64(r.Revs)
		}
		results = append(results, DefectsResult{
			Entity:          r.Entity,
			BugfixRevisions: bugfixRevs,
			TotalRevisions:  r.Revs,
			DefectRatio:     ratio,
		})
	}
	slices.SortFunc(results, func(a, b DefectsResult) int {
		if c := cmp.Compare(b.DefectRatio, a.DefectRatio); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})

	return results
}

func FormatDefects(results []DefectsResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "bugfix-revisions", "total-revisions", "defect-ratio"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity,
			fmt.Sprint(r.BugfixRevisions),
			fmt.Sprint(r.TotalRevisions),
			fmt.Sprintf("%.2f", r.DefectRatio),
		})
	}
	return out
}
