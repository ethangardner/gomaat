package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestHotspotsJoinAndScore(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "hot.go"},
		{Rev: "r2", Author: "Alice", Entity: "hot.go"},
		{Rev: "r3", Author: "Bob", Entity: "hot.go"},
		{Rev: "r1", Author: "Alice", Entity: "cold.go"},
	}
	locByFile := map[string]int{
		"hot.go":  100,
		"cold.go": 100,
	}

	results := Hotspots(commits, locByFile, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// hot.go: 3 revs (max), 100 lines (max) -> score 100
	if results[0].Entity != "hot.go" {
		t.Errorf("expected hot.go first, got %q", results[0].Entity)
	}
	if results[0].Revisions != 3 {
		t.Errorf("hot.go revisions: got %d, want 3", results[0].Revisions)
	}
	if results[0].Lines != 100 {
		t.Errorf("hot.go lines: got %d, want 100", results[0].Lines)
	}
	if results[0].Score != 100.0 {
		t.Errorf("hot.go score: got %f, want 100.0", results[0].Score)
	}

	// cold.go: 1 rev out of max 3, 100 lines out of max 100 -> score = (1/3)*(1)*100 = 33.33
	if results[1].Entity != "cold.go" {
		t.Errorf("expected cold.go second, got %q", results[1].Entity)
	}
	if results[1].Score != 33.33 {
		t.Errorf("cold.go score: got %f, want 33.33", results[1].Score)
	}
}

func TestHotspotsExcludesFileMissingFromDisk(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "deleted.go"},
		{Rev: "r1", Author: "Alice", Entity: "kept.go"},
	}
	// deleted.go has history but no current lines-of-code entry (e.g. deleted from disk).
	locByFile := map[string]int{
		"kept.go": 50,
	}

	results := Hotspots(commits, locByFile, model.Options{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result (deleted.go excluded), got %d: %v", len(results), results)
	}
	if results[0].Entity != "kept.go" {
		t.Errorf("expected kept.go, got %q", results[0].Entity)
	}
}

func TestHotspotsExcludesFileMissingFromLog(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "tracked.go"},
	}
	// new.go exists on disk but has no revision history in this log.
	locByFile := map[string]int{
		"tracked.go": 20,
		"new.go":     500,
	}

	results := Hotspots(commits, locByFile, model.Options{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result (new.go excluded), got %d: %v", len(results), results)
	}
	if results[0].Entity != "tracked.go" {
		t.Errorf("expected tracked.go, got %q", results[0].Entity)
	}
}

func TestHotspotsFractalPassthrough(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "shared.go"},
		{Rev: "r2", Author: "Bob", Entity: "shared.go"},
	}
	locByFile := map[string]int{"shared.go": 10}

	results := Hotspots(commits, locByFile, model.Options{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Two equal authors -> fractal 0.5, matching Fragmentation's own formula.
	if results[0].Fractal != 0.5 {
		t.Errorf("fractal: got %f, want 0.5", results[0].Fractal)
	}
}

func TestHotspotsEmptyInput(t *testing.T) {
	results := Hotspots(nil, map[string]int{}, model.Options{})
	assertEmptyResults(t, results)
}

func TestHotspotsSortTiebreaker(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "b.go"},
		{Rev: "r1", Author: "Alice", Entity: "a.go"},
	}
	locByFile := map[string]int{"a.go": 10, "b.go": 10}

	results := Hotspots(commits, locByFile, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Score != results[1].Score {
		t.Fatalf("expected equal scores for tiebreaker test, got %v and %v", results[0].Score, results[1].Score)
	}
	if results[0].Entity != "a.go" {
		t.Errorf("expected a.go first (alphabetical tiebreaker), got %q", results[0].Entity)
	}
}

func TestFormatHotspots(t *testing.T) {
	results := []HotspotResult{
		{Entity: "hot.go", Revisions: 10, Lines: 200, Score: 87.5, Fractal: 0.33},
	}
	rows := FormatHotspots(results, model.Options{})
	assertFormattedRows(t, rows, "entity", 2)
	want := []string{"hot.go", "10", "200", "87.50", "0.33"}
	for i, col := range want {
		if rows[1][i] != col {
			t.Errorf("row[%d]: got %q, want %q", i, rows[1][i], col)
		}
	}
}
