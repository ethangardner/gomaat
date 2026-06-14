package analysis

import (
	"fmt"
	"sort"

	"gomaat/internal/model"
)

type RevisionsResult struct {
	Entity string
	Revs   int
}

func Revisions(commits []model.Commit, _ model.Options) []RevisionsResult {
	revsByEntity := map[string]map[string]struct{}{}
	for _, c := range commits {
		if _, ok := revsByEntity[c.Entity]; !ok {
			revsByEntity[c.Entity] = map[string]struct{}{}
		}
		revsByEntity[c.Entity][c.Rev] = struct{}{}
	}

	results := make([]RevisionsResult, 0, len(revsByEntity))
	for entity, revs := range revsByEntity {
		results = append(results, RevisionsResult{entity, len(revs)})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Revs != results[j].Revs {
			return results[i].Revs > results[j].Revs
		}
		return results[i].Entity < results[j].Entity
	})

	return results
}

func FormatRevisions(results []RevisionsResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "n-revs"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Revs)})
	}
	return out
}
