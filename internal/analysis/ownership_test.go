package analysis

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestBusFactor(t *testing.T) {
	tests := []struct {
		name      string
		commits   []model.Commit
		busFactor int
		owners    []string
		ownership float64
	}{
		{
			name: "single owner",
			commits: []model.Commit{
				{Author: "Alice", Entity: "foo.go", LocAdded: 10},
			},
			busFactor: 1,
			owners:    []string{"Alice"},
			ownership: 100,
		},
		{
			name: "majority owner",
			commits: []model.Commit{
				{Author: "Alice", Entity: "foo.go", LocAdded: 60},
				{Author: "Bob", Entity: "foo.go", LocAdded: 40},
			},
			busFactor: 1,
			owners:    []string{"Alice"},
			ownership: 60,
		},
		{
			name: "even three-way ownership needs two owners",
			commits: []model.Commit{
				{Author: "Carol", Entity: "foo.go", LocAdded: 10},
				{Author: "Bob", Entity: "foo.go", LocAdded: 10},
				{Author: "Alice", Entity: "foo.go", LocAdded: 10},
			},
			busFactor: 2,
			owners:    []string{"Alice", "Bob"},
			ownership: 200.0 / 3,
		},
		{
			name: "exactly half is not a majority",
			commits: []model.Commit{
				{Author: "Alice", Entity: "foo.go", LocAdded: 50},
				{Author: "Bob", Entity: "foo.go", LocAdded: 50},
			},
			busFactor: 2,
			owners:    []string{"Alice", "Bob"},
			ownership: 100,
		},
		{
			name: "sums an author's lines across commits",
			commits: []model.Commit{
				{Author: "Alice", Entity: "foo.go", LocAdded: 30},
				{Author: "Bob", Entity: "foo.go", LocAdded: 40},
				{Author: "Alice", Entity: "foo.go", LocAdded: 30},
			},
			busFactor: 1,
			owners:    []string{"Alice"},
			ownership: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := BusFactor(tt.commits, model.Options{})
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			r := results[0]
			if r.BusFactor != tt.busFactor {
				t.Errorf("bus-factor: got %d, want %d", r.BusFactor, tt.busFactor)
			}
			if !slices.Equal(r.TopOwners, tt.owners) {
				t.Errorf("top-owners: got %v, want %v", r.TopOwners, tt.owners)
			}
			if math.Abs(r.Ownership-tt.ownership) > 1e-9 {
				t.Errorf("ownership: got %f, want %f", r.Ownership, tt.ownership)
			}
		})
	}
}

func TestBusFactorSortsRiskiestFirstAndSkipsUnowned(t *testing.T) {
	commits := []model.Commit{
		{Author: "Alice", Entity: "shared.go", LocAdded: 10},
		{Author: "Bob", Entity: "shared.go", LocAdded: 10},
		{Author: "Alice", Entity: "b.go", LocAdded: 10},
		{Author: "Alice", Entity: "a.go", LocAdded: 10},
		{Author: "Alice", Entity: "deleted-only.go", LocDeleted: 5},
	}
	results := BusFactor(commits, model.Options{})

	var got []string
	for _, r := range results {
		got = append(got, r.Entity)
	}
	if want := []string{"a.go", "b.go", "shared.go"}; !slices.Equal(got, want) {
		t.Errorf("order: got %v, want %v", got, want)
	}
}

func TestBusFactorEmpty(t *testing.T) {
	assertEmptyResults(t, BusFactor(nil, model.Options{}))
}

func TestFormatBusFactor(t *testing.T) {
	rows := FormatBusFactor([]BusFactorResult{
		{Entity: "foo.go", BusFactor: 2, TopOwners: []string{"Alice", "Bob"}, Ownership: 200.0 / 3},
	}, model.Options{})
	want := [][]string{
		{"entity", "bus-factor", "top-owners", "ownership"},
		{"foo.go", "2", "Alice;Bob", "66.67"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %v, want %v", rows, want)
	}
}

func TestKnowledgeLoss(t *testing.T) {
	former := map[string]struct{}{"Alice": {}, "Ghost": {}}
	commits := []model.Commit{
		// solely owned by a former author
		{Author: "Alice", Entity: "legacy.go", LocAdded: 40},
		{Author: "Alice", Entity: "legacy.go", LocAdded: 60},
		// mixed ownership
		{Author: "Alice", Entity: "mixed.go", LocAdded: 25},
		{Author: "Bob", Entity: "mixed.go", LocAdded: 75},
		// no former authors
		{Author: "Bob", Entity: "fresh.go", LocAdded: 10},
		// no added lines to attribute
		{Author: "Alice", Entity: "deleted-only.go", LocDeleted: 5},
	}

	got := KnowledgeLoss(commits, model.Options{FormerAuthors: former})
	want := []KnowledgeLossResult{
		{Entity: "legacy.go", FormerAdded: 100, TotalAdded: 100, Loss: 100},
		{Entity: "mixed.go", FormerAdded: 25, TotalAdded: 100, Loss: 25},
		{Entity: "fresh.go", FormerAdded: 0, TotalAdded: 10, Loss: 0},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestKnowledgeLossNoFormerAuthorsInLog(t *testing.T) {
	commits := []model.Commit{{Author: "Bob", Entity: "foo.go", LocAdded: 10}}
	for _, former := range []map[string]struct{}{nil, {"Ghost": {}}} {
		results := KnowledgeLoss(commits, model.Options{FormerAuthors: former})
		if len(results) != 1 || results[0].Loss != 0 {
			t.Errorf("former %v: expected one 0%% result, got %+v", former, results)
		}
	}
}

func TestKnowledgeLossEmpty(t *testing.T) {
	assertEmptyResults(t, KnowledgeLoss(nil, model.Options{}))
}

func TestFormatKnowledgeLoss(t *testing.T) {
	rows := FormatKnowledgeLoss([]KnowledgeLossResult{
		{Entity: "mixed.go", FormerAdded: 1, TotalAdded: 3, Loss: 100.0 / 3},
	}, model.Options{})
	want := [][]string{
		{"entity", "former-added", "total-added", "knowledge-loss"},
		{"mixed.go", "1", "3", "33.33"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %v, want %v", rows, want)
	}
}
