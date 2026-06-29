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

	results := Authors(commits, model.Options{})

	if len(results) != 2 { // 2 entities
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// foo.go has 2 authors → sorts first
	if results[0].Entity != "foo.go" || results[0].Authors != 2 || results[0].Revs != 3 {
		t.Errorf("result 0: got %v, want {foo.go 2 3}", results[0])
	}
	// bar.go has 1 author
	if results[1].Entity != "bar.go" || results[1].Authors != 1 || results[1].Revs != 1 {
		t.Errorf("result 1: got %v, want {bar.go 1 1}", results[1])
	}

	assertFormattedRows(t, FormatAuthors(results, model.Options{}), "entity", 3)
}

func TestAuthorsDeduplicatesRevisions(t *testing.T) {
	// Same author, same rev, same entity — should count as one revision
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
	}
	results := Authors(commits, model.Options{})
	if results[0].Revs != 1 {
		t.Errorf("expected 1 revision (deduped), got %d", results[0].Revs)
	}

	rows := FormatAuthors(results, model.Options{})
	if rows[1][2] != "1" {
		t.Errorf("expected 1 revision (deduped) in formatted rows, got %q", rows[1][2])
	}
}
