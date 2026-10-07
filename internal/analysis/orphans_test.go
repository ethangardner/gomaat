package analysis

import (
	"reflect"
	"slices"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestOrphans(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "dumped.go", LocAdded: 500},
		{Rev: "r1", Date: "2024-01-01", Author: "Alice", Entity: "tiny.go", LocAdded: 10},
		{Rev: "r2", Date: "2024-02-01", Author: "Bob", Entity: "at-threshold.go", LocAdded: 200},
		{Rev: "r2", Date: "2024-02-01", Author: "Bob", Entity: "revised.go", LocAdded: 900},
		{Rev: "r3", Date: "2024-03-01", Author: "Carol", Entity: "revised.go", LocAdded: 1, LocDeleted: 1},
	}
	opts := model.Options{MinLines: 200}

	got := Orphans(commits, opts)
	want := []OrphanResult{
		{Entity: "dumped.go", Added: 500, Date: "2024-01-01", Author: "Alice"},
		{Entity: "at-threshold.go", Added: 200, Date: "2024-02-01", Author: "Bob"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}

	rows := FormatOrphans(got, opts)
	wantRows := [][]string{
		{"entity", "added-lines", "date", "author"},
		{"dumped.go", "500", "2024-01-01", "Alice"},
		{"at-threshold.go", "200", "2024-02-01", "Bob"},
	}
	if !reflect.DeepEqual(rows, wantRows) {
		t.Errorf("rows: got %v, want %v", rows, wantRows)
	}
}

func TestOrphansEmpty(t *testing.T) {
	assertEmptyResults(t, Orphans(nil, model.Options{MinLines: 200}))
}
