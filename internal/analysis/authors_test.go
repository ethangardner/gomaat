package analysis

import (
	"testing"

	"gomaat/internal/model"
)

func TestAuthors(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r2", Author: "Bob", Entity: "foo.go"},
		{Rev: "r3", Author: "Alice", Entity: "foo.go"}, // Alice again, same entity
		{Rev: "r1", Author: "Alice", Entity: "bar.go"},
	}

	rows := Authors(commits, model.Options{})

	if rows[0][0] != "entity" {
		t.Fatalf("expected header row, got %v", rows[0])
	}
	if len(rows) != 3 { // header + 2 entities
		t.Fatalf("expected 3 rows (header + 2), got %d", len(rows))
	}

	// foo.go has 2 authors → sorts first
	if rows[1][0] != "foo.go" || rows[1][1] != "2" || rows[1][2] != "3" {
		t.Errorf("row 1: got %v, want [foo.go 2 3]", rows[1])
	}
	// bar.go has 1 author
	if rows[2][0] != "bar.go" || rows[2][1] != "1" || rows[2][2] != "1" {
		t.Errorf("row 2: got %v, want [bar.go 1 1]", rows[2])
	}
}

func TestAuthorsDeduplicatesRevisions(t *testing.T) {
	// Same author, same rev, same entity — should count as one revision
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
	}
	rows := Authors(commits, model.Options{})
	if rows[1][2] != "1" {
		t.Errorf("expected 1 revision (deduped), got %q", rows[1][2])
	}
}
