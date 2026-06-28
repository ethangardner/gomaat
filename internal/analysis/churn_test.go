package analysis

import (
	"testing"

	"gomaat/internal/model"
)

// shared fixture for all churn tests
var churnCommits = []model.Commit{
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "foo.go", LocAdded: 10, LocDeleted: 5},
	{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "bar.go", LocAdded: 3, LocDeleted: 1},
	{Rev: "r2", Date: "2024-01-02", Author: "Bob", Entity: "foo.go", LocAdded: 2, LocDeleted: 8},
}

func TestAbsChurn(t *testing.T) {
	results := AbsChurn(churnCommits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// sorted by date asc
	if results[0].Key != "2024-01-01" || results[0].Added != 13 || results[0].Deleted != 6 || results[0].Commits != 1 {
		t.Errorf("result 0: got %v, want {2024-01-01 13 6 1}", results[0])
	}
	if results[1].Key != "2024-01-02" || results[1].Added != 2 || results[1].Deleted != 8 || results[1].Commits != 1 {
		t.Errorf("result 1: got %v, want {2024-01-02 2 8 1}", results[1])
	}

	// Verify formatter
	rows := FormatAbsChurn(results, model.Options{})
	if rows[0][0] != "date" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestAuthorChurn(t *testing.T) {
	results := AuthorChurn(churnCommits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// sorted by author asc: Alice before Bob
	if results[0].Key != "Alice" || results[0].Added != 13 || results[0].Deleted != 6 || results[0].Commits != 1 {
		t.Errorf("Alice result: got %v, want {Alice 13 6 1}", results[0])
	}
	if results[1].Key != "Bob" || results[1].Added != 2 || results[1].Deleted != 8 || results[1].Commits != 1 {
		t.Errorf("Bob result: got %v, want {Bob 2 8 1}", results[1])
	}

	rows := FormatAuthorChurn(results, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestEntityChurn(t *testing.T) {
	results := EntityChurn(churnCommits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// sorted by added desc: foo.go (12) before bar.go (3)
	if results[0].Key != "foo.go" || results[0].Added != 12 || results[0].Deleted != 13 || results[0].Commits != 2 {
		t.Errorf("foo.go result: got %v, want {foo.go 12 13 2}", results[0])
	}
	if results[1].Key != "bar.go" || results[1].Added != 3 || results[1].Deleted != 1 || results[1].Commits != 1 {
		t.Errorf("bar.go result: got %v, want {bar.go 3 1 1}", results[1])
	}

	rows := FormatEntityChurn(results, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestEntityOwnership(t *testing.T) {
	results := EntityOwnership(churnCommits, model.Options{})
	// 3 (entity,author) pairs: bar.go/Alice, foo.go/Alice, foo.go/Bob
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// sorted entity asc, then author asc
	if results[0].Entity != "bar.go" || results[0].Author != "Alice" || results[0].Added != 3 || results[0].Deleted != 1 {
		t.Errorf("bar.go/Alice result: got %v, want {bar.go Alice 3 1}", results[0])
	}

	rows := FormatEntityOwnership(results, model.Options{})
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows (header + 3), got %d", len(rows))
	}
}

func TestMainDev(t *testing.T) {
	results := MainDev(churnCommits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// sorted by entity: bar.go, foo.go
	// bar.go: Alice 3/3 = 100%
	if results[0].Entity != "bar.go" || results[0].Contributor != "Alice" || results[0].Count != 3 || results[0].Total != 3 || results[0].Ownership != 100.0 {
		t.Errorf("bar.go result: got %v, want {bar.go Alice 3 3 100.00}", results[0])
	}
	// foo.go: Alice (10 added) beats Bob (2 added), 10/12 ≈ 83.33%
	if results[1].Entity != "foo.go" || results[1].Contributor != "Alice" || results[1].Ownership < 83.3 || results[1].Ownership > 83.4 {
		t.Errorf("foo.go result: got %v, want entity=foo.go dev=Alice ownership ≈ 83.33", results[1])
	}

	rows := FormatMainDev(results, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}

func TestRefactoringMainDev(t *testing.T) {
	results := RefactoringMainDev(churnCommits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// foo.go: Bob deleted 8, Alice deleted 5 → Bob is refactoring main dev, 8/13 ≈ 61.54%
	if results[1].Entity != "foo.go" || results[1].Contributor != "Bob" || results[1].Ownership < 61.5 || results[1].Ownership > 61.6 {
		t.Errorf("foo.go result: got %v, want entity=foo.go dev=Bob ownership ≈ 61.54", results[1])
	}

	rows := FormatRefactoringMainDev(results, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
}
