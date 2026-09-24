package gitdiff

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func collect(t *testing.T, input string) []Commit {
	t.Helper()
	var commits []Commit
	for c, err := range Parse(strings.NewReader(input)) {
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		commits = append(commits, c)
	}
	return commits
}

func TestParse(t *testing.T) {
	input := "\x00aaa\x001700000000\x00Alice\x00\n" +
		"\n" +
		"diff --git a/new.go b/new.go\n" +
		"new file mode 100644\n" +
		"index 0000000..1111111\n" +
		"--- /dev/null\n" +
		"+++ b/new.go\n" +
		"@@ -0,0 +1,2 @@\n" +
		"+package x\n" +
		"+-- looks like a header\n" +
		"\x00bbb\x001700086400\x00Bob Smith\x00aaa\n" +
		"\n" +
		"diff --git a/new.go b/new.go\n" +
		"--- a/new.go\n" +
		"+++ b/new.go\n" +
		"@@ -2 +2 @@ func\n" +
		"--- looks like a header\n" +
		"+++ also looks like a header\n" +
		"@@ -3,0 +4 @@\n" +
		"+tail\n" +
		"\\ No newline at end of file\n" +
		"diff --git a/logo.png b/logo.png\n" +
		"index 1..2 100644\n" +
		"Binary files a/logo.png and b/logo.png differ\n" +
		"diff --git a/gone.txt b/gone.txt\n" +
		"deleted file mode 100644\n" +
		"--- a/gone.txt\n" +
		"+++ /dev/null\n" +
		"@@ -1 +0,0 @@\n" +
		"-bye\n" +
		"\x00ccc\x001700172800\x00Carol\x00aaa bbb\n"

	want := []Commit{
		{
			Rev: "aaa", Time: time.Unix(1700000000, 0).UTC(), Author: "Alice",
			Files: []FileDiff{{Path: "new.go", Hunks: []Hunk{
				{OldStart: 0, NewStart: 1, Added: []string{"package x", "-- looks like a header"}},
			}}},
		},
		{
			Rev: "bbb", Time: time.Unix(1700086400, 0).UTC(), Author: "Bob Smith", Parents: []string{"aaa"},
			Files: []FileDiff{
				{Path: "new.go", Hunks: []Hunk{
					{OldStart: 2, NewStart: 2, Deleted: []string{"-- looks like a header"}, Added: []string{"++ also looks like a header"}},
					{OldStart: 3, NewStart: 4, Added: []string{"tail"}},
				}},
				{Path: "logo.png", Binary: true},
				{Path: "gone.txt", Removed: true, Hunks: []Hunk{{OldStart: 1, NewStart: 0, Deleted: []string{"bye"}}}},
			},
		},
		{Rev: "ccc", Time: time.Unix(1700172800, 0).UTC(), Author: "Carol", Parents: []string{"aaa", "bbb"}},
	}

	got := collect(t, input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Parse mismatch\n got: %+v\nwant: %+v", got, want)
	}
}

func TestParseQuotedAndSpacedPaths(t *testing.T) {
	input := "\x00aaa\x001700000000\x00Alice\x00\n" +
		"diff --git a/a b.txt b/a b.txt\n" +
		"--- /dev/null\n" +
		"+++ b/a b.txt\t\n" +
		"@@ -0,0 +1 @@\n" +
		"+x\n" +
		"diff --git \"a/\\303\\251\\tt.txt\" \"b/\\303\\251\\tt.txt\"\n" +
		"--- /dev/null\n" +
		"+++ \"b/\\303\\251\\tt.txt\"\n" +
		"@@ -0,0 +1 @@\n" +
		"+y\n" +
		"diff --git a/bin with space.png b/bin with space.png\n" +
		"Binary files /dev/null and b/bin with space.png differ\n" +
		"diff --git \"a/\\303\\251.png\" \"b/\\303\\251.png\"\n" +
		"Binary files /dev/null and \"b/\\303\\251.png\" differ\n"

	var paths []string
	for _, f := range collect(t, input)[0].Files {
		paths = append(paths, f.Path)
	}
	want := []string{"a b.txt", "é\tt.txt", "bin with space.png", "é.png"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("paths: got %q, want %q", paths, want)
	}
}

func TestParseEmpty(t *testing.T) {
	if got := collect(t, ""); len(got) != 0 {
		t.Errorf("expected no commits, got %d", len(got))
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"bad header", "\x00aaa\x00Alice\n", "malformed commit header"},
		{"bad time", "\x00aaa\x00noon\x00Alice\x00\n", "malformed commit time"},
		{"bad hunk", "\x00a\x001\x00A\x00\ndiff --git a/x b/x\n@@ -x +1 @@\n", "malformed hunk header"},
		{"unexpected hunk line", "\x00a\x001\x00A\x00\ndiff --git a/x b/x\n@@ -1 +1 @@\n context\n", "unexpected line in hunk"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotErr error
			for _, err := range Parse(strings.NewReader(tt.input)) {
				gotErr = err
			}
			if gotErr == nil || !strings.Contains(gotErr.Error(), tt.want) {
				t.Errorf("got error %v, want it to contain %q", gotErr, tt.want)
			}
		})
	}
}

func TestParseTruncatedHunk(t *testing.T) {
	input := "\x00a\x001\x00A\x00\ndiff --git a/x b/x\n@@ -0,0 +1,3 @@\n+only one\n"
	var gotErr error
	for _, err := range Parse(strings.NewReader(input)) {
		gotErr = err
	}
	if !errors.Is(gotErr, io.ErrUnexpectedEOF) {
		t.Errorf("got %v, want io.ErrUnexpectedEOF", gotErr)
	}
}

func TestParseStopsWhenConsumerStops(t *testing.T) {
	input := "\x00a\x001\x00A\x00\n\x00b\x002\x00B\x00\n"
	n := 0
	for range Parse(strings.NewReader(input)) {
		n++
		break
	}
	if n != 1 {
		t.Errorf("expected 1 iteration, got %d", n)
	}
}
