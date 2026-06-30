package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestFragmentation(t *testing.T) {
	tests := []struct {
		name      string
		commits   []model.Commit
		entity    string
		fractal   float64
		totalRevs int
	}{
		{
			name: "single author owns everything",
			commits: []model.Commit{
				{Rev: "r1", Author: "Alice", Entity: "foo.go"},
				{Rev: "r2", Author: "Alice", Entity: "foo.go"},
			},
			entity:    "foo.go",
			fractal:   0.00,
			totalRevs: 2,
		},
		{
			name: "two equal authors",
			commits: []model.Commit{
				{Rev: "r1", Author: "Alice", Entity: "bar.go"},
				{Rev: "r2", Author: "Bob", Entity: "bar.go"},
			},
			entity:    "bar.go",
			fractal:   0.50,
			totalRevs: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := Fragmentation(tt.commits, model.Options{})
			if len(results) < 1 {
				t.Fatalf("expected at least one result, got %d", len(results))
			}
			r := results[0]
			if r.Entity != tt.entity {
				t.Errorf("entity: got %q, want %q", r.Entity, tt.entity)
			}
			if r.Fractal != tt.fractal {
				t.Errorf("fractal: got %f, want %f", r.Fractal, tt.fractal)
			}
			if r.TotalRevs != tt.totalRevs {
				t.Errorf("total-revs: got %d, want %d", r.TotalRevs, tt.totalRevs)
			}

			// Verify formatter
			rows := FormatFragmentation(results, model.Options{})
			if len(rows) < 2 {
				t.Fatalf("expected at least two formatted rows")
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
	results := EntityEffort(commits, model.Options{})

	if len(results) != 3 { // bar.go/Alice, foo.go/Alice, foo.go/Bob
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	// sorted entity asc, then author-revs desc within entity
	if results[0].Entity != "bar.go" || results[0].Author != "Alice" || results[0].AuthorRevs != 1 || results[0].TotalRevs != 1 {
		t.Errorf("bar.go/Alice result: got %v", results[0])
	}
	if results[1].Entity != "foo.go" || results[1].Author != "Alice" || results[1].AuthorRevs != 2 || results[1].TotalRevs != 2 {
		t.Errorf("foo.go/Alice result: got %v", results[1])
	}
	if results[2].Entity != "foo.go" || results[2].Author != "Bob" || results[2].AuthorRevs != 1 || results[2].TotalRevs != 2 {
		t.Errorf("foo.go/Bob result: got %v", results[2])
	}

	assertFormattedRows(t, FormatEntityEffort(results, model.Options{}), "entity", 4)
}

func TestMainDevByRevs(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go"},
		{Rev: "r2", Author: "Alice", Entity: "foo.go"},
		{Rev: "r1", Author: "Bob", Entity: "foo.go"},
	}
	results := MainDevByRevs(commits, model.Options{})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Alice: 2 revs out of 2 total → 100%
	if results[0].Contributor != "Alice" || results[0].Count != 2 || results[0].Total != 2 || results[0].Ownership != 100.0 {
		t.Errorf("got %v", results[0])
	}

	// Verify formatter
	rows := FormatMainDevByRevs(results, model.Options{})
	if rows[1][1] != "Alice" || rows[1][2] != "2" || rows[1][3] != "2" || rows[1][4] != "100.00" {
		t.Errorf("got %v, want [foo.go Alice 2 2 100.00] formatted", rows[1])
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
	results := Fragmentation(commits, model.Options{})
	if results[0].Entity != "bar.go" {
		t.Errorf("expected bar.go (higher fractal) first, got %q", results[0].Entity)
	}
}

func TestFragmentationTiebreaker(t *testing.T) {
	// Two single-author entities both have fractal=0; tie-breaker sorts alphabetically by entity name.
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "b.go"},
		{Rev: "r2", Author: "Bob", Entity: "a.go"},
	}
	results := Fragmentation(commits, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Fractal != results[1].Fractal {
		t.Fatalf("expected equal fractal values for tie-breaker test, got %v and %v", results[0].Fractal, results[1].Fractal)
	}
	if results[0].Entity != "a.go" {
		t.Errorf("expected a.go first (alphabetical tie-breaker), got %q", results[0].Entity)
	}
}
