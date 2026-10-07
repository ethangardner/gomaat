package analysis

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/model"
)

// whoKnowsCommits has Alice writing most of src/core long ago and Bob making
// small, recent changes, so all-time and recent ownership disagree.
var whoKnowsCommits = []model.Commit{
	{Rev: "r1", Date: "2021-03-01", Author: "Alice", Entity: "src/core/engine.go", LocAdded: 900},
	{Rev: "r2", Date: "2024-05-01", Author: "Bob", Entity: "src/core/engine.go", LocAdded: 30, LocDeleted: 5},
	{Rev: "r3", Date: "2024-06-01", Author: "Carol", Entity: "src/core/parser.go", LocAdded: 10},
	{Rev: "r3", Date: "2024-06-01", Author: "Carol", Entity: "src/corex.go", LocAdded: 500},
	{Rev: "r4", Date: "2024-06-02", Author: "Dave", Entity: "src/core/parser.go", LocDeleted: 3},
}

var whoKnowsAfter = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func TestWhoKnows(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []WhoKnowsResult
	}{
		{
			name: "file",
			path: "src/core/engine.go",
			want: []WhoKnowsResult{{"Bob", 30, 100}},
		},
		{
			name: "directory aggregates entities under it, not siblings sharing its prefix",
			path: "src/core",
			want: []WhoKnowsResult{{"Bob", 30, 75}, {"Carol", 10, 25}, {"Dave", 0, 0}},
		},
		{
			name: "trailing slash",
			path: "src/core/",
			want: []WhoKnowsResult{{"Bob", 30, 75}, {"Carol", 10, 25}, {"Dave", 0, 0}},
		},
		{name: "no match", path: "docs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WhoKnows(whoKnowsCommits, model.Options{Path: tt.path, After: whoKnowsAfter})
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestWhoKnowsDisagreesWithMainDev(t *testing.T) {
	opts := model.Options{Path: "src/core/engine.go", After: whoKnowsAfter}
	if got := MainDev(whoKnowsCommits, opts)[0].Contributor; got != "Alice" {
		t.Fatalf("main-dev: got %q, want Alice", got)
	}
	if got := WhoKnows(whoKnowsCommits, opts)[0].Author; got != "Bob" {
		t.Errorf("whoknows: got %q, want Bob", got)
	}
}

func TestWhoKnowsIncludesCommitsOnCutoffDate(t *testing.T) {
	opts := model.Options{Path: "src/core/engine.go", After: time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC)}
	if got := WhoKnows(whoKnowsCommits, opts)[0].Author; got != "Alice" {
		t.Errorf("got %q, want Alice", got)
	}
}

func TestFormatWhoKnows(t *testing.T) {
	rows := FormatWhoKnows([]WhoKnowsResult{{"Bob", 30, 75}}, model.Options{})
	want := [][]string{{"author", "added", "ownership"}, {"Bob", "30", "75.00"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %v, want %v", rows, want)
	}
}
