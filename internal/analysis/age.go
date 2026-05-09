package analysis

import (
	"fmt"
	"sort"
	"time"

	"godemaat/internal/model"
)

// Age calculates months since last modification for each entity.
func Age(commits []model.Commit, opts model.Options) [][]string {
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

	type row struct {
		entity    string
		ageMonths int
	}
	rows := make([]row, 0, len(lastDate))
	for entity, dateStr := range lastDate {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		months := monthsBetween(t, now)
		rows = append(rows, row{entity, months})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ageMonths != rows[j].ageMonths {
			return rows[i].ageMonths < rows[j].ageMonths
		}
		return rows[i].entity < rows[j].entity
	})

	out := [][]string{{"entity", "age-months"}}
	for _, r := range rows {
		out = append(out, []string{r.entity, fmt.Sprint(r.ageMonths)})
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
