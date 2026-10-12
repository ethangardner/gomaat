package analysis

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/ethangardner/gomaat/internal/model"
)

type WhoKnowsResult struct {
	Author    string
	Added     int
	Ownership float64 // Added as a share of all lines added to the path in the window
}

// WhoKnows ranks authors by lines added on or after opts.After to opts.Path:
// a single entity, or every entity beneath it when it names a directory. An
// empty Path covers every entity. Unlike MainDev, older contributions drop
// out entirely rather than counting forever, so someone who wrote a file
// years ago and hasn't touched it since doesn't outrank a current
// maintainer. Authors whose commits in the window only deleted lines are
// listed with 0 added. Sorted by added descending, then author.
func WhoKnows(commits []model.Commit, opts model.Options) []WhoKnowsResult {
	after := opts.After.Format(time.DateOnly)
	var recent []model.Commit
	for _, c := range commits {
		if c.Date >= after && underPath(c.Entity, opts.Path) {
			recent = append(recent, c)
		}
	}

	churn := AuthorChurn(recent, opts)
	var total int
	for _, c := range churn {
		total += c.Added
	}

	results := make([]WhoKnowsResult, len(churn))
	for i, c := range churn {
		results[i] = WhoKnowsResult{c.Key, c.Added, percent(c.Added, total)}
	}
	slices.SortFunc(results, func(a, b WhoKnowsResult) int {
		return cmp.Or(cmp.Compare(b.Added, a.Added), cmp.Compare(a.Author, b.Author))
	})
	return results
}

func FormatWhoKnows(results []WhoKnowsResult, _ model.Options) [][]string {
	return formatRows([]string{"author", "added", "ownership"}, results, func(r WhoKnowsResult) []string {
		return []string{r.Author, fmt.Sprint(r.Added), fmt.Sprintf("%.2f", r.Ownership)}
	})
}

// underPath reports whether entity is dir itself or lies beneath it. Matching
// whole path segments keeps "src/core" from matching "src/corex.go".
func underPath(entity, dir string) bool {
	dir = strings.TrimSuffix(dir, "/")
	return dir == "" || entity == dir || strings.HasPrefix(entity, dir+"/")
}
