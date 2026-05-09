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
	rows := AbsChurn(churnCommits, model.Options{})
	if rows[0][0] != "date" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted by date asc
	if rows[1][0] != "2024-01-01" || rows[1][1] != "13" || rows[1][2] != "6" || rows[1][3] != "1" {
		t.Errorf("row 1: got %v, want [2024-01-01 13 6 1]", rows[1])
	}
	if rows[2][0] != "2024-01-02" || rows[2][1] != "2" || rows[2][2] != "8" || rows[2][3] != "1" {
		t.Errorf("row 2: got %v, want [2024-01-02 2 8 1]", rows[2])
	}
}

func TestAuthorChurn(t *testing.T) {
	rows := AuthorChurn(churnCommits, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted by author asc: Alice before Bob
	if rows[1][0] != "Alice" || rows[1][1] != "13" || rows[1][2] != "6" || rows[1][3] != "1" {
		t.Errorf("Alice row: got %v, want [Alice 13 6 1]", rows[1])
	}
	if rows[2][0] != "Bob" || rows[2][1] != "2" || rows[2][2] != "8" || rows[2][3] != "1" {
		t.Errorf("Bob row: got %v, want [Bob 2 8 1]", rows[2])
	}
}

func TestEntityChurn(t *testing.T) {
	rows := EntityChurn(churnCommits, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted by added desc: foo.go (12) before bar.go (3)
	if rows[1][0] != "foo.go" || rows[1][1] != "12" || rows[1][2] != "13" || rows[1][3] != "2" {
		t.Errorf("foo.go row: got %v, want [foo.go 12 13 2]", rows[1])
	}
	if rows[2][0] != "bar.go" || rows[2][1] != "3" || rows[2][2] != "1" || rows[2][3] != "1" {
		t.Errorf("bar.go row: got %v, want [bar.go 3 1 1]", rows[2])
	}
}

func TestEntityOwnership(t *testing.T) {
	rows := EntityOwnership(churnCommits, model.Options{})
	// 3 (entity,author) pairs: bar.go/Alice, foo.go/Alice, foo.go/Bob
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows (header + 3), got %d", len(rows))
	}
	// sorted entity asc, then author asc
	if rows[1][0] != "bar.go" || rows[1][1] != "Alice" || rows[1][2] != "3" || rows[1][3] != "1" {
		t.Errorf("bar.go/Alice row: got %v, want [bar.go Alice 3 1]", rows[1])
	}
}

func TestMainDev(t *testing.T) {
	rows := MainDev(churnCommits, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted by entity: bar.go, foo.go
	// bar.go: Alice 3/3 = 100%
	if rows[1][0] != "bar.go" || rows[1][1] != "Alice" || rows[1][2] != "3" || rows[1][3] != "3" || rows[1][4] != "100.00" {
		t.Errorf("bar.go row: got %v, want [bar.go Alice 3 3 100.00]", rows[1])
	}
	// foo.go: Alice (10 added) beats Bob (2 added), 10/12 ≈ 83.33%
	if rows[2][0] != "foo.go" || rows[2][1] != "Alice" || rows[2][4] != "83.33" {
		t.Errorf("foo.go row: got %v, want entity=foo.go dev=Alice ownership=83.33", rows[2])
	}
}

func TestRefactoringMainDev(t *testing.T) {
	rows := RefactoringMainDev(churnCommits, model.Options{})
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// foo.go: Bob deleted 8, Alice deleted 5 → Bob is refactoring main dev, 8/13 ≈ 61.54%
	if rows[2][0] != "foo.go" || rows[2][1] != "Bob" || rows[2][4] != "61.54" {
		t.Errorf("foo.go row: got %v, want entity=foo.go dev=Bob ownership=61.54", rows[2])
	}
}
