package analysis

import (
	"fmt"
	"sort"
	"time"

	"gomaat/internal/model"
)

type AgeResult struct {
	Entity    string
	AgeMonths int
}

// Age calculates months since last modification for each entity.
func Age(commits []model.Commit, opts model.Options) []AgeResult {
	lastDate := map[string]string{}
	for _, c := range commits {
		cur, ok := lastDate[c.Entity]
		if !ok || c.Date > cur {
			lastDate[c.Entity] = c.Date
		}
	}

	now := opts.AgeTimeNow
	if now.IsZero() {
		now = time.Now()
	}

	results := make([]AgeResult, 0, len(lastDate))
	for entity, dateStr := range lastDate {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		months := monthsBetween(t, now)
		results = append(results, AgeResult{entity, months})
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].AgeMonths != results[j].AgeMonths {
			return results[i].AgeMonths < results[j].AgeMonths
		}
		return results[i].Entity < results[j].Entity
	})

	return results
}

func FormatAge(results []AgeResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "age-months"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.AgeMonths)})
	}
	return out
}

func monthsBetween(from, to time.Time) int {
	years := to.Year() - from.Year()
	months := int(to.Month()) - int(from.Month())
	total := years*12 + months
	if total < 0 {
		return 0
	}
	return total
}
