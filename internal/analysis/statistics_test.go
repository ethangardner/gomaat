package analysis

import (
	"math"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestPerCommitTotals(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go", LocAdded: 1, LocDeleted: 0},
		{Rev: "r1", Entity: "b.go", LocAdded: 2, LocDeleted: 1},
		{Rev: "r2", Entity: "a.go", LocAdded: 0, LocDeleted: 3},
		{Rev: "r2", Entity: "a.go", LocAdded: 1, LocDeleted: 0},
	}

	files, loc := perCommitTotals(commits)

	wantFiles := []float64{2, 1}
	wantLoc := []float64{4, 4}
	for i := range wantFiles {
		if files[i] != wantFiles[i] {
			t.Errorf("filesPerCommit[%d] = %v, want %v", i, files[i], wantFiles[i])
		}
		if loc[i] != wantLoc[i] {
			t.Errorf("locPerCommit[%d] = %v, want %v", i, loc[i], wantLoc[i])
		}
	}
}

func TestStatisticsKnownDistribution(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "e1"},
		{Rev: "r2", Entity: "e2"}, {Rev: "r2", Entity: "e3"},
		{Rev: "r3", Entity: "e4"}, {Rev: "r3", Entity: "e5"}, {Rev: "r3", Entity: "e6"},
		{Rev: "r4", Entity: "e7"}, {Rev: "r4", Entity: "e8"}, {Rev: "r4", Entity: "e9"}, {Rev: "r4", Entity: "e10"},
		{Rev: "r5", Entity: "e11"}, {Rev: "r5", Entity: "e12"}, {Rev: "r5", Entity: "e13"}, {Rev: "r5", Entity: "e14"}, {Rev: "r5", Entity: "e15"},
	}

	results := Statistics(commits, model.Options{})

	var got StatisticsResult
	for _, r := range results {
		if r.Metric == "files-changed-per-commit" {
			got = r
		}
	}

	if got.Count != 5 || got.Min != 1 || got.Q1 != 2 || got.Median != 3 || got.Q3 != 4 || got.Max != 5 || got.Mean != 3 {
		t.Fatalf("files-changed-per-commit = %+v, want Count 5/Min 1/Q1 2/Median 3/Q3 4/Max 5/Mean 3", got)
	}
	if !floatEquals(got.Stddev, math.Sqrt(2.5)) {
		t.Errorf("Stddev = %v, want %v", got.Stddev, math.Sqrt(2.5))
	}
}

func TestStatisticsIgnoresMaxChangesetSize(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go"},
		{Rev: "r1", Entity: "b.go"},
	}

	filtered := Statistics(commits, model.Options{MaxChangesetSize: 1})
	unfiltered := Statistics(commits, model.Options{})

	var socFiltered, socUnfiltered StatisticsResult
	for _, r := range filtered {
		if r.Metric == "soc-per-entity" {
			socFiltered = r
		}
	}
	for _, r := range unfiltered {
		if r.Metric == "soc-per-entity" {
			socUnfiltered = r
		}
	}

	if socFiltered.Count == 0 || socFiltered.Mean == 0 {
		t.Fatalf("soc-per-entity with MaxChangesetSize:1 = %+v, want non-zero (opts should be ignored)", socFiltered)
	}
	if socFiltered != socUnfiltered {
		t.Errorf("soc-per-entity differs between opts: %+v vs %+v, want identical", socFiltered, socUnfiltered)
	}
}

func TestStatisticsRowOrderAndHeader(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go", Author: "Alice"},
	}
	rows := FormatStatistics(Statistics(commits, model.Options{}), model.Options{})

	assertFormattedRows(t, rows, "metric", 6)

	wantOrder := []string{
		"files-changed-per-commit",
		"lines-changed-per-commit",
		"revisions-per-entity",
		"authors-per-entity",
		"soc-per-entity",
	}
	for i, want := range wantOrder {
		if rows[i+1][0] != want {
			t.Errorf("rows[%d][0] = %q, want %q", i+1, rows[i+1][0], want)
		}
	}
}

func TestStatisticsEmptyInput(t *testing.T) {
	results := Statistics(nil, model.Options{})
	if len(results) != 5 {
		t.Fatalf("expected 5 metrics, got %d", len(results))
	}
	for _, r := range results {
		if r.Count != 0 || r.Min != 0 || r.Max != 0 || r.Mean != 0 || r.Stddev != 0 {
			t.Errorf("metric %q on empty input = %+v, want all zero", r.Metric, r)
		}
	}

	rows := FormatStatistics(results, model.Options{})
	if len(rows) != 6 {
		t.Errorf("expected 6 rows (header + 5 metrics), got %d", len(rows))
	}
}
