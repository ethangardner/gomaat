package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

var simpleCommits = []model.Commit{
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "foo.go", LocAdded: 5, LocDeleted: 2},
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "bar.go", LocAdded: 1, LocDeleted: 0},
	{Rev: "r2", Date: "2024-01-02", Author: "Bob", Entity: "foo.go", LocAdded: 3, LocDeleted: 1},
}

func TestRevisions(t *testing.T) {
	results := Revisions(simpleCommits, model.Options{})

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// sorted by n-revs desc: foo.go (2) > bar.go (1)
	if results[0].Entity != "foo.go" || results[0].Revs != 2 {
		t.Errorf("result 0: got %v, want {foo.go 2}", results[0])
	}
	if results[1].Entity != "bar.go" || results[1].Revs != 1 {
		t.Errorf("result 1: got %v, want {bar.go 1}", results[1])
	}

	rows := FormatRevisions(results, model.Options{})
	assertFormattedRows(t, rows, "entity", 3)
	if rows[1][0] != "foo.go" || rows[1][1] != "2" {
		t.Errorf("row 1: got %v, want [foo.go 2]", rows[1])
	}
}

func TestSummary(t *testing.T) {
	result := Summary(simpleCommits, model.Options{})

	if result.Commits != 2 {
		t.Errorf("Commits: got %d, want 2", result.Commits)
	}
	if result.Entities != 2 {
		t.Errorf("Entities: got %d, want 2", result.Entities)
	}
	if result.EntitiesChanged != 3 {
		t.Errorf("EntitiesChanged: got %d, want 3", result.EntitiesChanged)
	}
	if result.Authors != 2 {
		t.Errorf("Authors: got %d, want 2", result.Authors)
	}

	// Verify formatter
	rows := FormatSummary(result, model.Options{})
	if len(rows) != 5 {
		t.Fatalf("expected 5 rows (header + 4 stats), got %d", len(rows))
	}
	stats := map[string]string{}
	for _, r := range rows[1:] {
		stats[r[0]] = r[1]
	}
	if stats["number-of-commits"] != "2" {
		t.Errorf("number-of-commits: got %q, want 2", stats["number-of-commits"])
	}
}

func TestIdentity(t *testing.T) {
	results := Identity(simpleCommits, model.Options{})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Entity != "foo.go" || results[0].Rev != "r1" {
		t.Errorf("result 0: got %v, want entity foo.go and rev r1", results[0])
	}

	rows := FormatIdentity(results, model.Options{})
	assertFormattedRows(t, rows, "entity", 4)
	r := rows[1]
	if r[0] != "foo.go" || r[1] != "r1" || r[2] != "2024-01-01" || r[3] != "Alice" || r[4] != "5" || r[5] != "2" {
		t.Errorf("row 1: got %v, want [foo.go r1 2024-01-01 Alice 5 2]", r)
	}
}
