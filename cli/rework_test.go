package cli

import (
	"encoding/json"
	"errors"
	"iter"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/gitdiff"
)

func TestParseWindow(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"14d", 14 * 24 * time.Hour, false},
		{"2w", 14 * 24 * time.Hour, false},
		{"36h", 36 * time.Hour, false},
		{"90m", 90 * time.Minute, false},
		{"0d", 0, true},
		{"-1d", 0, true},
		{"0s", 0, true},
		{"abc", 0, true},
		{"d", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseWindow(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseWindow(%q) = %v, want error", tt.in, got)
				}
				if !strings.Contains(err.Error(), "--rework-window") {
					t.Errorf("error %q should mention --rework-window", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWindow(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("parseWindow(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func seqOf(cs []gitdiff.Commit, err error) iter.Seq2[gitdiff.Commit, error] {
	return func(yield func(gitdiff.Commit, error) bool) {
		for _, c := range cs {
			if !yield(c, nil) {
				return
			}
		}
		if err != nil {
			yield(gitdiff.Commit{}, err)
		}
	}
}

func TestExcludeFiles(t *testing.T) {
	files := func(paths ...string) []gitdiff.FileDiff {
		var fs []gitdiff.FileDiff
		for _, p := range paths {
			fs = append(fs, gitdiff.FileDiff{Path: p})
		}
		return fs
	}
	boom := errors.New("boom")
	in := []gitdiff.Commit{
		{Rev: "a", Files: files("vendor/lib.go", "src/main.go", "src/api/gen.pb.go")},
		{Rev: "b", Files: files("vendor/only.go")},
	}

	var gotPaths [][]string
	var gotErr error
	for c, err := range excludeFiles(seqOf(in, boom), []string{"vendor/", "*.pb.go"}) {
		if err != nil {
			gotErr = err
			break
		}
		var paths []string
		for _, f := range c.Files {
			paths = append(paths, f.Path)
		}
		gotPaths = append(gotPaths, paths)
	}

	// commit b keeps its (now empty) slot so the history stays intact
	if want := [][]string{{"src/main.go"}, nil}; !reflect.DeepEqual(gotPaths, want) {
		t.Errorf("got %v, want %v", gotPaths, want)
	}
	if !errors.Is(gotErr, boom) {
		t.Errorf("got err %v, want %v", gotErr, boom)
	}
}

func TestExcludeFilesNoPatternsIsIdentity(t *testing.T) {
	in := []gitdiff.Commit{{Rev: "a", Files: []gitdiff.FileDiff{{Path: "vendor/x.go"}}}}
	var got []gitdiff.Commit
	for c, err := range excludeFiles(seqOf(in, nil), nil) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, c)
	}
	if !reflect.DeepEqual(got, in) {
		t.Errorf("got %v, want %v", got, in)
	}
}

// reworkFixtureRepo builds a repo where churned.go has a block added on day 0
// and deleted on day 3 (rework), plus a line removed on day 30 (outside the
// default 14-day window), and stable.go is added on day 0 and never touched.
func reworkFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeRepoFile(t, dir, "churned.go", "package x\n\nfunc keep() {}\nfunc tmpA() int { return 1 }\nfunc tmpB() int { return 2 }\n")
	writeRepoFile(t, dir, "stable.go", "package x\nfunc stable() {}\n")
	commitAll(t, dir, "day 0", "2024-01-01T12:00:00Z")

	writeRepoFile(t, dir, "churned.go", "package x\n\nfunc keep() {}\n")
	commitAll(t, dir, "day 3: drop tmp funcs", "2024-01-04T12:00:00Z")

	writeRepoFile(t, dir, "churned.go", "package x\n\n")
	commitAll(t, dir, "day 30: drop keep", "2024-01-31T12:00:00Z")
	return dir
}

// runReworkCmd runs the rework subcommand against dir with extra flags and
// positional args, returning what it wrote to the output file.
func runReworkCmd(t *testing.T, dir string, flags map[string]string, args ...string) (string, error) {
	t.Helper()
	resetFlags(t)
	outFile = filepath.Join(t.TempDir(), "out")

	cmd := newReworkCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("before", "2024-03-01"); err != nil {
		t.Fatal(err)
	}
	for k, v := range flags {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.RunE(cmd, args); err != nil {
		return "", err
	}
	return readOutputFile(t, outFile), nil
}

func TestReworkCmd(t *testing.T) {
	dir := reworkFixtureRepo(t)
	header := "entity,added-lines,reworked-lines,rework-ratio\n"

	tests := []struct {
		name  string
		flags map[string]string
		args  []string
		want  string
	}{
		{
			name: "block deleted within window is reworked, untouched file is not",
			want: header + "churned.go,4,2,50.00\nstable.go,2,0,0.00\n",
		},
		{
			name:  "wider window also catches the day-30 deletion",
			flags: map[string]string{"rework-window": "5w"},
			want:  header + "churned.go,4,3,75.00\nstable.go,2,0,0.00\n",
		},
		{
			name: "positional pathspec scopes the analysis",
			args: []string{"stable.go"},
			want: header + "stable.go,2,0,0.00\n",
		},
		{
			name:  "glob exclude drops matching files",
			flags: map[string]string{"exclude": "churned*"},
			want:  header + "stable.go,2,0,0.00\n",
		},
		{
			name:  "after skips earlier history, leaving nothing tracked",
			flags: map[string]string{"after": "2024-01-02"},
			want:  header,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runReworkCmd(t, dir, tt.flags, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// A branch commit on day 0 merged into main on day 20 lands on day 20: a
// deletion on day 30 is 10 days after landing, so it's rework even though
// it's 30 days after the line was first written on the branch.
func TestReworkCmdCountsMergedLinesAtMergeTime(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeRepoFile(t, dir, "base.go", "package x\n")
	commitAll(t, dir, "base", "2024-01-01T00:00:00Z")

	gitRun(t, dir, nil, "checkout", "-b", "feature")
	writeRepoFile(t, dir, "feat.go", "func a() {}\nfunc b() {}\n")
	commitAll(t, dir, "feature", "2024-01-01T12:00:00Z")

	gitRun(t, dir, nil, "checkout", "main")
	merged := "2024-01-21T00:00:00Z"
	gitRun(t, dir, []string{"GIT_AUTHOR_DATE=" + merged, "GIT_COMMITTER_DATE=" + merged}, "merge", "--no-ff", "-m", "merge feature", "feature")

	writeRepoFile(t, dir, "feat.go", "func a() {}\n")
	commitAll(t, dir, "drop b", "2024-01-31T00:00:00Z")

	got, err := runReworkCmd(t, dir, nil, "feat.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "entity,added-lines,reworked-lines,rework-ratio\nfeat.go,2,1,50.00\n"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// With diff.noprefix set, git would print "--- b/x.go" for a file at b/x.go,
// and the parser would strip the "b/" as if it were git's prefix.
func TestReworkCmdIgnoresUserDiffPrefixConfig(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	gitRun(t, dir, nil, "config", "diff.noprefix", "true")

	writeRepoFile(t, dir, "b/x.go", "func a() {}\nfunc b() {}\n")
	commitAll(t, dir, "add", "2024-01-01T00:00:00Z")
	writeRepoFile(t, dir, "b/x.go", "func a() {}\n")
	commitAll(t, dir, "drop b", "2024-01-02T00:00:00Z")

	got, err := runReworkCmd(t, dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := "entity,added-lines,reworked-lines,rework-ratio\nb/x.go,2,1,50.00\n"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestReworkCmdJSON(t *testing.T) {
	dir := reworkFixtureRepo(t)
	resetFlags(t)
	outputFormat = "json"

	got, err := runReworkCmd(t, dir, nil, "stable.go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(got), &rows); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(rows) != 1 || rows[0]["entity"] != "stable.go" {
		t.Errorf("got %v, want one stable.go row", rows)
	}
}

func TestReworkCmdErrors(t *testing.T) {
	dir := reworkFixtureRepo(t)
	tests := []struct {
		name    string
		setup   func()
		flags   map[string]string
		wantErr string
	}{
		{"bad window", nil, map[string]string{"rework-window": "soon"}, "--rework-window"},
		{"bad before", nil, map[string]string{"before": "yesterday"}, "--before"},
		{"bad after", nil, map[string]string{"after": "yesterday"}, "--after"},
		{"bad format", func() { outputFormat = "yaml" }, nil, "--format"},
		{"log flag set", func() { logFile = "x.log" }, nil, "--log"},
		{"group flag set", func() { groupFile = "g.txt" }, nil, "--group"},
		{"team map flag set", func() { teamMapFile = "t.csv" }, nil, "--team-map-file"},
		{"not a repo", nil, map[string]string{"path": t.TempDir()}, "git log failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			if tt.setup != nil {
				tt.setup()
			}
			if _, err := runReworkCmd(t, dir, tt.flags); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("got err %v, want one mentioning %q", err, tt.wantErr)
			}
		})
	}
}

func TestReworkCmdRegistered(t *testing.T) {
	if cmd, _, err := rootCmd.Find([]string{"rework"}); err != nil || cmd.Name() != "rework" {
		t.Fatalf("rework subcommand not registered: %v", err)
	}
}
