package analysis

import (
	"testing"

	"gomaat/internal/model"
)

func TestFragmentation(t *testing.T) {
	tests := []struct {
		name     string
		commits  []model.Commit
		entity   string
		fractal  string
		totalRevs string
	}{
		{
			name: "single author owns everything",
			commits: []model.Commit{
				{Rev: "r1", Author: "Alice", Entity: "foo.go"},
				{Rev: "r2", Author: "Alice", Entity: "foo.go"},
			},
			entity:    "foo.go",
			fractal:   "0.00",
			totalRevs: "2",
		},
		{
			name: "two equal authors",
			commits: []model.Commit{
				{Rev: "r1", Author: "Alice", Entity: "bar.go"},
				{Rev: "r2", Author: "Bob", Entity: "bar.go"},
			},
			entity:    "bar.go",
			fractal:   "0.50",
			totalRevs: "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := Fragmentation(tt.commits, model.Options{})
			if len(rows) < 2 {
				t.Fatalf("expected at least one data row, got %d rows", len(rows))
			}
			r := rows[1]
			if r[0] != tt.entity {
				t.Errorf("entity: got %q, want %q", r[0], tt.entity)
			}
			if r[1] != tt.fractal {
				t.Errorf("fractal-value: got %q, want %q", r[1], tt.fractal)
			}
			if r[2] != tt.totalRevs {
				t.Errorf("total-revs: got %q, want %q", r[2], tt.totalRevs)
			}
		})
	}
}

func TestEntityEffort(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r2", Author: "Alice", Entity: "foo.go"},
		{Rev: "r1", Author: "Bob", Entity: "foo.go"},
		{Rev: "r1", Author: "Alice", Entity: "bar.go"},
	}
	rows := EntityEffort(commits, model.Options{})

	if rows[0][0] != "entity" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 4 { // header + bar.go/Alice + foo.go/Alice + foo.go/Bob
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	// sorted entity asc, then author-revs desc within entity
	if rows[1][0] != "bar.go" || rows[1][1] != "Alice" || rows[1][2] != "1" || rows[1][3] != "1" {
		t.Errorf("bar.go/Alice row: got %v, want [bar.go Alice 1 1]", rows[1])
	}
	if rows[2][0] != "foo.go" || rows[2][1] != "Alice" || rows[2][2] != "2" || rows[2][3] != "2" {
		t.Errorf("foo.go/Alice row: got %v, want [foo.go Alice 2 2]", rows[2])
	}
	if rows[3][0] != "foo.go" || rows[3][1] != "Bob" || rows[3][2] != "1" || rows[3][3] != "2" {
		t.Errorf("foo.go/Bob row: got %v, want [foo.go Bob 1 2]", rows[3])
	}
}

func TestMainDevByRevs(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r2", Author: "Alice", Entity: "foo.go"},
		{Rev: "r1", Author: "Bob", Entity: "foo.go"},
	}
	rows := MainDevByRevs(commits, model.Options{})

	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// Alice: 2 revs out of 2 total → 100%
	if rows[1][1] != "Alice" || rows[1][2] != "2" || rows[1][3] != "2" || rows[1][4] != "100.00" {
		t.Errorf("got %v, want [foo.go Alice 2 2 100.00]", rows[1])
	}
}

func TestFragmentationSortOrder(t *testing.T) {
	// Two entities: foo.go (single author, fractal=0) and bar.go (two equal, fractal=0.5)
	// Sorted descending by fractal → bar.go first
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r2", Author: "Alice", Entity: "bar.go"},
		{Rev: "r3", Author: "Bob", Entity: "bar.go"},
	}
	rows := Fragmentation(commits, model.Options{})
	if rows[1][0] != "bar.go" {
		t.Errorf("expected bar.go (higher fractal) first, got %q", rows[1][0])
	}
}
