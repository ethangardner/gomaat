package analysis

import (
	"testing"

	"gomaat/internal/model"
)

func TestCommunication(t *testing.T) {
	// foo.go touched by Alice+Bob → shared entity
	// bar.go touched by Alice only
	// baz.go touched by Bob only
	//
	// freqs: AA=2, BB=2, AB=1, BA=1
	// Each non-self pair: shared=1, avg=ceil((2+2)/2)=2, strength=50
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "foo.go", Author: "Bob"},
		{Entity: "bar.go", Author: "Alice"},
		{Entity: "baz.go", Author: "Bob"},
	}

	rows := Communication(commits, model.Options{})
	if rows[0][0] != "author" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 { // header + Alice→Bob + Bob→Alice
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// Both pairs should have strength=50; sorted desc by author name: Bob first
	if rows[1][0] != "Bob" || rows[1][1] != "Alice" || rows[1][2] != "1" || rows[1][3] != "2" || rows[1][4] != "50" {
		t.Errorf("row 1: got %v, want [Bob Alice 1 2 50]", rows[1])
	}
	if rows[2][0] != "Alice" || rows[2][1] != "Bob" {
		t.Errorf("row 2: got %v, want [Alice Bob ...]", rows[2])
	}
}

func TestCommunicationSingleAuthor(t *testing.T) {
	// Only one author touching any entities → no non-self pairs
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "bar.go", Author: "Alice"},
	}
	rows := Communication(commits, model.Options{})
	if len(rows) != 1 {
		t.Errorf("expected header only (no pairs), got %d rows", len(rows))
	}
}
