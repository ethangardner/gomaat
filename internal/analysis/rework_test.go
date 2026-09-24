package analysis

import (
	"errors"
	"iter"
	"math"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/gitdiff"
	"github.com/ethangardner/gomaat/internal/model"
)

var reworkEpoch = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

func day(n int) time.Time { return reworkEpoch.AddDate(0, 0, n) }

// reworkOpts uses a 14-day window judged at day 100, so every line added
// before day 86 is old enough to judge.
var reworkOpts = model.Options{ReworkWindow: 14 * 24 * time.Hour, ReworkTimeNow: day(100)}

func commitsOf(cs ...gitdiff.Commit) iter.Seq2[gitdiff.Commit, error] {
	return func(yield func(gitdiff.Commit, error) bool) {
		for _, c := range cs {
			if !yield(c, nil) {
				return
			}
		}
	}
}

func commitOn(d int, files ...gitdiff.FileDiff) gitdiff.Commit {
	return gitdiff.Commit{Rev: time.Duration(d).String(), Time: day(d), Parents: []string{"p"}, Files: files}
}

func fileDiff(path string, hunks ...gitdiff.Hunk) gitdiff.FileDiff {
	return gitdiff.FileDiff{Path: path, Hunks: hunks}
}

// addFile creates path with lines.
func addFile(path string, lines ...string) gitdiff.FileDiff {
	return fileDiff(path, gitdiff.Hunk{OldStart: 0, NewStart: 1, Added: lines})
}

func runRework(t *testing.T, cs ...gitdiff.Commit) []ReworkResult {
	t.Helper()
	results, err := Rework(commitsOf(cs...), reworkOpts)
	if err != nil {
		t.Fatalf("Rework: %v", err)
	}
	return results
}

func TestRework(t *testing.T) {
	tests := []struct {
		name    string
		commits []gitdiff.Commit
		want    []ReworkResult
	}{
		{
			name: "block deleted within window is reworked",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "one()", "two()", "three()", "four()")),
				commitOn(5, fileDiff("a.go", gitdiff.Hunk{OldStart: 2, NewStart: 1, Deleted: []string{"two()", "three()"}})),
			},
			want: []ReworkResult{{"a.go", 4, 2, 50}},
		},
		{
			name: "deletion after the window is not rework",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "one()", "two()")),
				commitOn(15, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"one()"}})),
			},
			want: []ReworkResult{{"a.go", 2, 0, 0}},
		},
		{
			name: "substantial rewrite in place is rework",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "total := sum(values)")),
				commitOn(3, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 1,
					Deleted: []string{"total := sum(values)"}, Added: []string{"return strings.Join(parts, sep)"}})),
			},
			// the replacement line (day 3) is also added and survives
			want: []ReworkResult{{"a.go", 2, 1, 50}},
		},
		{
			name: "small edit in place keeps the original line",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "if count > limit {")),
				commitOn(3, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 1,
					Deleted: []string{"if count > limit {"}, Added: []string{"if count >= limit {"}})),
				// ...and the edited line is still judged from day 0: removing it
				// on day 20 is outside that window.
				commitOn(20, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"if count >= limit {"}})),
			},
			want: []ReworkResult{{"a.go", 1, 0, 0}},
		},
		{
			name: "re-indentation is not rework",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "x := 1")),
				commitOn(2, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 1,
					Deleted: []string{"x := 1"}, Added: []string{"\tx  :=  1"}})),
			},
			want: []ReworkResult{{"a.go", 1, 0, 0}},
		},
		{
			name: "moved line keeps its provenance and origin entity",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "helper()", "keep()")),
				commitOn(2,
					fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"helper()"}}),
					addFile("b.go", "helper()"),
				),
				// deleting it from b.go within a.go's window is rework of a.go
				commitOn(10, fileDiff("b.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"helper()"}})),
			},
			want: []ReworkResult{{"a.go", 2, 1, 50}},
		},
		{
			name: "rename (delete + add under --no-renames) carries provenance",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("old.go", "a()", "b()")),
				commitOn(1,
					gitdiff.FileDiff{Path: "old.go", Removed: true, Hunks: []gitdiff.Hunk{{OldStart: 1, NewStart: 0, Deleted: []string{"a()", "b()"}}}},
					addFile("new.go", "a()", "b()"),
				),
				commitOn(30, fileDiff("new.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"a()"}})),
			},
			want: []ReworkResult{{"old.go", 2, 0, 0}},
		},
		{
			name: "whole file deleted within window",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("tmp.go", "a()", "b()")),
				commitOn(1, gitdiff.FileDiff{Path: "tmp.go", Removed: true, Hunks: []gitdiff.Hunk{{OldStart: 1, NewStart: 0, Deleted: []string{"a()", "b()"}}}}),
			},
			want: []ReworkResult{{"tmp.go", 2, 2, 100}},
		},
		{
			name: "blank lines are not counted",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "a()", "", "   ")),
				commitOn(1, fileDiff("a.go", gitdiff.Hunk{OldStart: 2, NewStart: 1, Deleted: []string{"", "   "}})),
			},
			want: []ReworkResult{{"a.go", 1, 0, 0}},
		},
		{
			name: "lines too young to judge are excluded",
			commits: []gitdiff.Commit{
				commitOn(90, addFile("young.go", "a()", "b()")),
				commitOn(91, fileDiff("young.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"a()"}})),
			},
			want: []ReworkResult{},
		},
		{
			name: "merge-introduced lines count as landing at merge time",
			commits: []gitdiff.Commit{
				{Rev: "m", Time: day(0), Parents: []string{"p1", "p2"}, Files: []gitdiff.FileDiff{addFile("a.go", "merged()", "kept()")}},
				commitOn(1, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"merged()"}})),
			},
			want: []ReworkResult{{"a.go", 2, 1, 50}},
		},
		{
			name: "lines from before the analyzed history are untracked",
			commits: []gitdiff.Commit{
				// a.go already had 10 lines we never saw; replace line 5 with
				// two new ones, then remove one of those.
				commitOn(0, fileDiff("a.go", gitdiff.Hunk{OldStart: 5, NewStart: 5, Deleted: []string{"legacy()"}, Added: []string{"fresh()", "extra()"}})),
				commitOn(1, fileDiff("a.go", gitdiff.Hunk{OldStart: 6, NewStart: 5, Deleted: []string{"extra()"}})),
				commitOn(2, fileDiff("a.go", gitdiff.Hunk{OldStart: 9, NewStart: 8, Deleted: []string{"legacy2()"}})),
			},
			want: []ReworkResult{{"a.go", 2, 1, 50}},
		},
		{
			name: "insertion between existing lines shifts later lines",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.go", "first()", "last()")),
				// insert after line 1 on day 20 (so it's outside day 0's window
				// but judged on its own)...
				commitOn(20, fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 2, Added: []string{"middle()"}})),
				// ...then remove it (now line 2) on day 22, and last() (now line 3).
				commitOn(22, fileDiff("a.go",
					gitdiff.Hunk{OldStart: 2, NewStart: 1, Deleted: []string{"middle()"}},
					gitdiff.Hunk{OldStart: 3, NewStart: 1, Deleted: []string{"last()"}},
				)),
			},
			want: []ReworkResult{{"a.go", 3, 1, 100.0 / 3}},
		},
		{
			name: "binary change resets a file's tracking",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("a.dat", "text")),
				commitOn(1, gitdiff.FileDiff{Path: "a.dat", Binary: true}),
				commitOn(2, fileDiff("a.dat", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"text"}})),
			},
			want: []ReworkResult{{"a.dat", 1, 0, 0}},
		},
		{
			name: "sorted by reworked lines desc, then entity",
			commits: []gitdiff.Commit{
				commitOn(0, addFile("b.go", "x()"), addFile("a.go", "y()"), addFile("c.go", "p()", "q()")),
				commitOn(1,
					fileDiff("b.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"x()"}}),
					fileDiff("a.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"y()"}}),
					fileDiff("c.go", gitdiff.Hunk{OldStart: 1, NewStart: 0, Deleted: []string{"p()", "q()"}}),
				),
			},
			want: []ReworkResult{{"c.go", 2, 2, 100}, {"a.go", 1, 1, 100}, {"b.go", 1, 1, 100}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runRework(t, tt.commits...); !slices.EqualFunc(got, tt.want, sameReworkResult) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func sameReworkResult(g, w ReworkResult) bool {
	return g.Entity == w.Entity && g.Added == w.Added && g.Reworked == w.Reworked && math.Abs(g.Ratio-w.Ratio) <= 1e-9
}

func TestReworkPropagatesError(t *testing.T) {
	boom := errors.New("boom")
	seq := func(yield func(gitdiff.Commit, error) bool) { yield(gitdiff.Commit{}, boom) }
	if _, err := Rework(seq, reworkOpts); !errors.Is(err, boom) {
		t.Errorf("got %v, want %v", err, boom)
	}
}

func TestReworkEmpty(t *testing.T) {
	assertEmptyResults(t, runRework(t))
}

func TestFormatRework(t *testing.T) {
	rows := FormatRework([]ReworkResult{{"a.go", 3, 1, 100.0 / 3}}, model.Options{})
	want := [][]string{
		{"entity", "added-lines", "reworked-lines", "rework-ratio"},
		{"a.go", "3", "1", "33.33"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("got %v, want %v", rows, want)
	}
}

func TestTokenSimilarity(t *testing.T) {
	tests := []struct {
		a, b string
		want float64
	}{
		{"", "", 0},
		{"{", "}", 0},
		{"a", "", 0},
		{"foo(bar)", "foo( bar )", 1},
		{"if x > 0 {", "if x >= 0 {", 1},
		{"legacy()", "fresh()", 0},
		{"return nil", "return err", 0.5},
		{"naïve_name := compute(a, b)", "naïve_name := compute(a, c)", 0.75},
	}
	for _, tt := range tests {
		if got := tokenSimilarity(tt.a, tt.b); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("tokenSimilarity(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
