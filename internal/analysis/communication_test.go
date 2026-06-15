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

	results := Communication(commits, model.Options{})
	if len(results) != 2 { // Alice→Bob and Bob→Alice
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both pairs should have strength=50; sorted desc by author name: Bob first
	if results[0].Author != "Bob" || results[0].Peer != "Alice" || results[0].Shared != 1 || results[0].Average != 2 || results[0].Strength != 50 {
		t.Errorf("result 0: got %v, want {Bob Alice 1 2 50}", results[0])
	}
	if results[1].Author != "Alice" || results[1].Peer != "Bob" {
		t.Errorf("result 1: got %v, want {Alice Bob ...}", results[1])
	}

	// Verify formatter
	rows := FormatCommunication(results, model.Options{})
	if rows[0][0] != "author" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestCommunicationSingleAuthor(t *testing.T) {
	// Only one author touching any entities → no non-self pairs
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "bar.go", Author: "Alice"},
	}
	results := Communication(commits, model.Options{})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	rows := FormatCommunication(results, model.Options{})
	if len(rows) != 1 {
		t.Errorf("expected header only (no pairs), got %d rows", len(rows))
	}
}
