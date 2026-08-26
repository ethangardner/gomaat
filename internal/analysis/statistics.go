package analysis

import (
	"fmt"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type StatisticsResult struct {
	Metric string
	Count  int
	Min    float64
	Q1     float64
	Median float64
	Q3     float64
	Max    float64
	Mean   float64
	Stddev float64
}

// unfilteredOpts is passed to opts-sensitive analyses (SumOfCoupling) that
// Statistics reuses internally. Statistics is registered via simpleCmd,
// which calls the analysis func with a zero-value model.Options — and
// SumOfCoupling's filteredChangesets treats opts.MaxChangesetSize=0 as
// "drop every changeset" (len(deduped) > 0 is always true), which would
// silently zero out the soc-per-entity metric. Statistics is a
// whole-dataset health report, not a filtered coupling query, so it
// deliberately ignores its own opts parameter and always computes reused
// metrics unfiltered.
var unfilteredOpts = model.Options{MaxChangesetSize: math.MaxInt}

// Statistics computes descriptive statistics (count/min/q1/median/q3/max/
// mean/sample-stddev) for a curated set of core metrics. It intentionally
// ignores opts: this is a whole-codebase health report, not a filtered
// query, so none of model.Options' threshold fields (MinRevs,
// MaxChangesetSize, etc.) apply. Entity/author remapping via -g/-p still
// applies normally, since runAnalysis rewrites commits before Statistics
// ever sees them.
func Statistics(commits []model.Commit, _ model.Options) []StatisticsResult {
	filesPerCommit, locPerCommit := perCommitTotals(commits)

	revsPerEntity := extractFloats(Revisions(commits, unfilteredOpts), func(r RevisionsResult) int { return r.Revs })
	authorsPerEntity := extractFloats(Authors(commits, unfilteredOpts), func(r AuthorsResult) int { return r.Authors })
	socPerEntity := extractFloats(SumOfCoupling(commits, unfilteredOpts), func(r SumOfCouplingResult) int { return r.Soc })

	metrics := []struct {
		name   string
		values []float64
	}{
		{"files-changed-per-commit", filesPerCommit},
		{"lines-changed-per-commit", locPerCommit},
		{"revisions-per-entity", revsPerEntity},
		{"authors-per-entity", authorsPerEntity},
		{"soc-per-entity", socPerEntity},
	}

	results := make([]StatisticsResult, 0, len(metrics))
	for _, m := range metrics {
		results = append(results, describe(m.name, m.values))
	}
	return results
}

func FormatStatistics(results []StatisticsResult, _ model.Options) [][]string {
	out := [][]string{{"metric", "count", "min", "q1", "median", "q3", "max", "mean", "stddev"}}
	for _, r := range results {
		out = append(out, []string{
			r.Metric,
			fmt.Sprint(r.Count),
			fmt.Sprintf("%.2f", r.Min),
			fmt.Sprintf("%.2f", r.Q1),
			fmt.Sprintf("%.2f", r.Median),
			fmt.Sprintf("%.2f", r.Q3),
			fmt.Sprintf("%.2f", r.Max),
			fmt.Sprintf("%.2f", r.Mean),
			fmt.Sprintf("%.2f", r.Stddev),
		})
	}
	return out
}

// perCommitTotals groups commits by Rev and returns, for each distinct Rev
// (in ascending Rev order, for determinism), the number of distinct
// entities touched and the total lines changed (sum of LocAdded+LocDeleted
// across every row sharing that Rev — not deduplicated by entity, matching
// how git numstat reports one line per touched file).
//
// This intentionally does not reuse filteredChangesets (coupling.go): that
// helper drops any changeset over opts.MaxChangesetSize, which exists to
// bound pairwise-coupling computation cost, not to describe the dataset.
// Applying it here would silently exclude commits from a "codebase health"
// report. perCommitTotals is therefore independent of model.Options
// entirely, by design.
func perCommitTotals(commits []model.Commit) (filesPerCommit, locPerCommit []float64) {
	type agg struct {
		entities map[string]struct{}
		loc      int
	}
	byRev := map[string]*agg{}
	for _, c := range commits {
		a, ok := byRev[c.Rev]
		if !ok {
			a = &agg{entities: map[string]struct{}{}}
			byRev[c.Rev] = a
		}
		a.entities[c.Entity] = struct{}{}
		a.loc += c.LocAdded + c.LocDeleted
	}

	revs := make([]string, 0, len(byRev))
	for rev := range byRev {
		revs = append(revs, rev)
	}
	slices.Sort(revs)

	filesPerCommit = make([]float64, len(revs))
	locPerCommit = make([]float64, len(revs))
	for i, rev := range revs {
		filesPerCommit[i] = float64(len(byRev[rev].entities))
		locPerCommit[i] = float64(byRev[rev].loc)
	}
	return filesPerCommit, locPerCommit
}

// extractFloats projects a []int field out of an arbitrary result slice
// into a []float64, for feeding into describe().
func extractFloats[T any](items []T, get func(T) int) []float64 {
	out := make([]float64, len(items))
	for i, it := range items {
		out[i] = float64(get(it))
	}
	return out
}
