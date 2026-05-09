package analysis

import (
	"testing"

	"gomaat/internal/model"
)

var simpleCommits = []model.Commit{
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "foo.go", LocAdded: 5, LocDeleted: 2},
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "bar.go", LocAdded: 1, LocDeleted: 0},
	{Rev: "r2", Date: "2024-01-02", Author: "Bob", Entity: "foo.go", LocAdded: 3, LocDeleted: 1},
}

func TestRevisions(t *testing.T) {
	rows := Revisions(simpleCommits, model.Options{})

	if rows[0][0] != "entity" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted by n-revs desc: foo.go (2) > bar.go (1)
	if rows[1][0] != "foo.go" || rows[1][1] != "2" {
		t.Errorf("row 1: got %v, want [foo.go 2]", rows[1])
	}
	if rows[2][0] != "bar.go" || rows[2][1] != "1" {
		t.Errorf("row 2: got %v, want [bar.go 1]", rows[2])
	}
}

func TestSummary(t *testing.T) {
	rows := Summary(simpleCommits, model.Options{})

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
	if stats["number-of-entities"] != "2" {
		t.Errorf("number-of-entities: got %q, want 2", stats["number-of-entities"])
	}
	if stats["number-of-entities-changed"] != "3" {
		t.Errorf("number-of-entities-changed: got %q, want 3", stats["number-of-entities-changed"])
	}
	if stats["number-of-authors"] != "2" {
		t.Errorf("number-of-authors: got %q, want 2", stats["number-of-authors"])
	}
}

func TestIdentity(t *testing.T) {
	rows := Identity(simpleCommits, model.Options{})

	if rows[0][0] != "entity" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 4 { // header + 3 commits
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	r := rows[1]
	if r[0] != "foo.go" || r[1] != "r1" || r[2] != "2024-01-01" || r[3] != "Alice" || r[4] != "5" || r[5] != "2" {
		t.Errorf("row 1: got %v, want [foo.go r1 2024-01-01 Alice 5 2]", r)
	}
}
