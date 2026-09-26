package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

func TestMatchesExcludePattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"vendor/github.com/foo/bar.go", "vendor/", true},
		{"vendor/foo.go", "vendor/", true},
		{"src/vendor/foo.go", "vendor/", false},
		{"src/api/types.pb.go", "*.pb.go", true},
		{"src/api/types.go", "*.pb.go", false},
		{"src/generated/types.pb.go", "src/generated/*.pb.go", true},
		{"src/other/types.pb.go", "src/generated/*.pb.go", false},
		{"go.sum", "go.sum", true},
		{"go.mod", "go.sum", false},
	}

	for _, tt := range tests {
		got := matchesExcludePattern(tt.path, tt.pattern)
		if got != tt.want {
			t.Errorf("matchesExcludePattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestBuildPathspecArgs(t *testing.T) {
	tests := []struct {
		name     string
		includes []string
		excludes []string
		want     []string
	}{
		{"nil excludes", nil, nil, nil},
		{"empty excludes", nil, []string{}, nil},
		{"only glob patterns", nil, []string{"*.pb.go"}, nil},
		{"single dir pattern", nil, []string{"vendor/"}, []string{"--", ".", ":(exclude,literal)vendor/"}},
		{
			"multiple dir patterns",
			nil,
			[]string{"vendor/", "data/"},
			[]string{"--", ".", ":(exclude,literal)vendor/", ":(exclude,literal)data/"},
		},
		{
			"mixed dir and glob patterns",
			nil,
			[]string{"vendor/", "*.pb.go"},
			[]string{"--", ".", ":(exclude,literal)vendor/"},
		},
		{"includes only", []string{"src/a.go"}, nil, []string{"--", "src/a.go"}},
		{"includes with only glob excludes", []string{"src/"}, []string{"*.pb.go"}, []string{"--", "src/"}},
		{
			"includes replace the default . when excluding",
			[]string{"src/", "cmd/"},
			[]string{"src/gen/"},
			[]string{"--", "src/", "cmd/", ":(exclude,literal)src/gen/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPathspecArgs(tt.includes, tt.excludes)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildPathspecArgs(%v, %v) = %v, want %v", tt.includes, tt.excludes, got, tt.want)
			}
		})
	}
}

func TestRunGitLogExcludesDirectory(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	writeRepoFile(t, dir, "data/big.csv", "a,b,c\n")
	writeRepoFile(t, dir, "src_main.go", "package main\n")
	commitAll(t, dir, "initial", "")

	var out bytes.Buffer
	if err := runGitLog(dir, "", "", logFilters{Excludes: []string{"data/"}}, &out); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "src_main.go") {
		t.Errorf("expected src_main.go in output, got:\n%s", result)
	}
	if strings.Contains(result, "data/") || strings.Contains(result, "big.csv") {
		t.Errorf("expected data/ to be excluded from output, got:\n%s", result)
	}
}

func TestRunGitLogExcludesMergeCommits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeRepoFile(t, dir, "main.go", "package main\n")
	commitAll(t, dir, "initial", "")

	gitRun(t, dir, nil, "checkout", "-b", "feature")
	writeRepoFile(t, dir, "feature.go", "package main\n")
	commitAll(t, dir, "add feature", "")

	gitRun(t, dir, nil, "checkout", "main")
	writeRepoFile(t, dir, "other.go", "package main\n")
	commitAll(t, dir, "add other", "")

	gitRun(t, dir, nil, "merge", "feature", "--no-ff", "-m", "merge feature")

	mergeHeader := "--" + strings.TrimSpace(gitRun(t, dir, nil, "rev-parse", "HEAD")) + "--"

	var out bytes.Buffer
	if err := runGitLog(dir, "", "", logFilters{}, &out); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}

	result := out.String()
	if strings.Contains(result, mergeHeader) {
		t.Errorf("expected merge commit %q to be excluded, got:\n%s", mergeHeader, result)
	}
	for _, want := range []string{"main.go", "feature.go", "other.go"} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q from a regular commit in output, got:\n%s", want, result)
		}
	}
}

func TestRunGitLogBeforeFiltersCommits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeRepoFile(t, dir, "old.go", "content\n")
	commitAll(t, dir, "add old.go", "2020-01-01T00:00:00")
	writeRepoFile(t, dir, "new.go", "content\n")
	commitAll(t, dir, "add new.go", "2025-01-01T00:00:00")

	var out bytes.Buffer
	if err := runGitLog(dir, "", "2022-01-01", logFilters{}, &out); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}

	result := out.String()
	if !strings.Contains(result, "old.go") {
		t.Errorf("expected old.go (before cutoff) in output, got:\n%s", result)
	}
	if strings.Contains(result, "new.go") {
		t.Errorf("expected new.go (after cutoff) to be excluded, got:\n%s", result)
	}
}

func TestFilterExcludes(t *testing.T) {
	input := strings.Join([]string{
		"--abc123--2024-01-15--Alice",
		"5\t3\tsrc/foo.go",
		"2\t1\tvendor/github.com/lib/lib.go",
		"1\t0\tsrc/types.pb.go",
		"",
		"--def456--2024-02-01--Bob",
		"3\t2\tsrc/bar.go",
		"4\t0\tsrc/api/gen.pb.go",
		"",
	}, "\n")

	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, []string{"vendor/", "*.pb.go"}, nil, nil); err != nil {
		t.Fatalf("filterLog: %v", err)
	}
	result := out.String()

	kept := []string{"src/foo.go", "src/bar.go", "--abc123", "--def456"}
	for _, s := range kept {
		if !strings.Contains(result, s) {
			t.Errorf("expected %q to be kept in output, but it was removed:\n%s", s, result)
		}
	}

	removed := []string{"vendor/github.com/lib/lib.go", "src/types.pb.go", "src/api/gen.pb.go"}
	for _, s := range removed {
		if strings.Contains(result, s) {
			t.Errorf("expected %q to be excluded from output, but it was kept:\n%s", s, result)
		}
	}
}

func TestFilterExcludesNoPatterns(t *testing.T) {
	input := "5\t3\tvendor/foo.go\n"
	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, nil, nil, nil); err != nil {
		t.Fatalf("filterLog: %v", err)
	}
	if out.String() != input {
		t.Errorf("filterLog with no patterns should return input unchanged")
	}
}

func TestNumstatLineMatchesExclude(t *testing.T) {
	tests := []struct {
		line     string
		excludes []string
		want     bool
	}{
		{"5\t3\tvendor/foo.go", []string{"vendor/"}, true},
		{"5\t3\tsrc/foo.go", []string{"vendor/"}, false},
		// commit header lines are never excluded
		{"--abc--2024-01-01--Alice", []string{"vendor/"}, false},
		// blank lines are never excluded
		{"", []string{"vendor/"}, false},
	}

	for _, tt := range tests {
		got := numstatLineMatchesExclude(tt.line, tt.excludes)
		if got != tt.want {
			t.Errorf("numstatLineMatchesExclude(%q, %v) = %v, want %v", tt.line, tt.excludes, got, tt.want)
		}
	}
}

func TestFilterExcludesStreamPreservesTrailingNewline(t *testing.T) {
	input := "1\t0\tvendor/foo.go\n2\t0\tsrc/keep.go\n"

	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, []string{"vendor/"}, nil, nil); err != nil {
		t.Fatalf("filterLog: %v", err)
	}

	if out.String() != "2\t0\tsrc/keep.go\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestFilterExcludesStreamWriterError(t *testing.T) {
	err := filterLog(strings.NewReader("1\t0\tsrc/keep.go\n"), errWriter{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected writer error, got nil")
	}
}

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestFilterExcludesStreamReaderError(t *testing.T) {
	err := filterLog(errReader{}, &bytes.Buffer{}, nil, nil, nil)
	if err == nil {
		t.Fatal("expected reader error, got nil")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestFilterExcludesStreamCarriageReturn(t *testing.T) {
	input := "1\t0\tvendor/foo.go\r\n2\t0\tsrc/keep.go\r\n"

	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, []string{"vendor/"}, nil, nil); err != nil {
		t.Fatalf("filterLog: %v", err)
	}

	if out.String() != "2\t0\tsrc/keep.go\r\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestFilterLogExcludesAuthor(t *testing.T) {
	input := strings.Join([]string{
		"--aaa111--2024-01-15--Alice",
		"5\t3\tsrc/foo.go",
		"",
		"--bbb222--2024-02-01--dependabot[bot]",
		"3\t2\tgo.mod",
		"",
		"--ccc333--2024-03-01--renovate-bot",
		"1\t1\tgo.sum",
		"",
	}, "\n")

	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, nil, []string{"dependabot[bot]", "renovate*"}, nil); err != nil {
		t.Fatalf("filterLog: %v", err)
	}
	result := out.String()

	if !strings.Contains(result, "Alice") || !strings.Contains(result, "src/foo.go") {
		t.Errorf("expected Alice's commit to be kept, got:\n%s", result)
	}
	for _, s := range []string{"dependabot", "go.mod", "renovate-bot", "go.sum"} {
		if strings.Contains(result, s) {
			t.Errorf("expected %q to be excluded from output, but it was kept:\n%s", s, result)
		}
	}
}

func TestFilterLogIgnoresRevs(t *testing.T) {
	input := strings.Join([]string{
		"--aaa111--2024-01-15--Alice",
		"5\t3\tsrc/foo.go",
		"",
		"--bbb222--2024-02-01--Bob",
		"3\t2\tsrc/bar.go",
		"",
	}, "\n")

	var out bytes.Buffer
	if err := filterLog(strings.NewReader(input), &out, nil, nil, map[string]struct{}{"bbb222": {}}); err != nil {
		t.Fatalf("filterLog: %v", err)
	}
	result := out.String()

	if !strings.Contains(result, "src/foo.go") {
		t.Errorf("expected src/foo.go (kept commit) in output, got:\n%s", result)
	}
	if strings.Contains(result, "src/bar.go") || strings.Contains(result, "Bob") {
		t.Errorf("expected the ignored-rev commit to be excluded, got:\n%s", result)
	}
}

func TestFilterLogCombinesAllFilters(t *testing.T) {
	input := strings.Join([]string{
		"--aaa111--2024-01-15--Alice",
		"5\t3\tsrc/foo.go",
		"2\t1\tvendor/lib.go",
		"",
		"--bbb222--2024-02-01--dependabot[bot]",
		"3\t2\tgo.mod",
		"",
		"--ccc333--2024-03-01--Bob",
		"1\t1\tsrc/bar.go",
		"",
	}, "\n")

	var out bytes.Buffer
	err := filterLog(strings.NewReader(input), &out,
		[]string{"vendor/"},
		[]string{"dependabot*"},
		map[string]struct{}{"ccc333": {}},
	)
	if err != nil {
		t.Fatalf("filterLog: %v", err)
	}
	result := out.String()

	if !strings.Contains(result, "src/foo.go") {
		t.Errorf("expected src/foo.go to be kept, got:\n%s", result)
	}
	for _, s := range []string{"vendor/lib.go", "dependabot", "go.mod", "Bob", "src/bar.go"} {
		if strings.Contains(result, s) {
			t.Errorf("expected %q to be excluded from output, but it was kept:\n%s", s, result)
		}
	}
}

func TestMatchesAuthorPattern(t *testing.T) {
	tests := []struct {
		author  string
		pattern string
		want    bool
	}{
		{"dependabot[bot]", "dependabot[bot]", true},
		{"renovate-bot", "renovate*", true},
		{"Alice", "renovate*", false},
		{"alice", "Alice", false}, // case-sensitive
	}

	for _, tt := range tests {
		got := matchesAuthorPattern(tt.author, tt.pattern)
		if got != tt.want {
			t.Errorf("matchesAuthorPattern(%q, %q) = %v, want %v", tt.author, tt.pattern, got, tt.want)
		}
	}
}

func TestLoadIgnoreRevs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ignore-revs")
	content := "# a comment\naaa111\n\nbbb222\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	revs, err := loadIgnoreRevs(path)
	if err != nil {
		t.Fatalf("loadIgnoreRevs: %v", err)
	}

	want := map[string]struct{}{"aaa111": {}, "bbb222": {}}
	if !reflect.DeepEqual(revs, want) {
		t.Errorf("loadIgnoreRevs() = %v, want %v", revs, want)
	}
}

func TestReadIgnoreRevsReadError(t *testing.T) {
	_, err := readIgnoreRevs(iotest.ErrReader(errors.New("boom")))
	if err == nil || !strings.Contains(err.Error(), "reading ignore-revs file") {
		t.Fatalf("expected reading ignore-revs file error, got %v", err)
	}
}

func TestLoadIgnoreRevsMissingFile(t *testing.T) {
	_, err := loadIgnoreRevs(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing ignore-revs file, got nil")
	}
}

func TestRunGitLogUseMailmap(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeRepoFile(t, dir, ".mailmap", "Real Name <real@example.com> Test <test@example.com>\n")
	commitAll(t, dir, "add mailmap", "")
	commitFiles(t, dir, "main.go")

	var out bytes.Buffer
	if err := runGitLog(dir, "", "", logFilters{UseMailmap: true}, &out); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}

	if !strings.Contains(out.String(), "Real Name") {
		t.Errorf("expected mailmap-resolved author 'Real Name' in output, got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "--Test\n") {
		t.Errorf("expected raw author 'Test' to be resolved away, got:\n%s", out.String())
	}
}

func TestGenerateLogRunEWithNewFilters(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGenLogRepo(t, dir)
	outFile = filepath.Join(t.TempDir(), "out.log")

	cmd := newGenerateLogCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("exclude-author", "nobody-matches-this"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("use-mailmap", "true"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readOutputFile(t, outFile), "main.go") {
		t.Errorf("expected main.go in output file")
	}
}

func TestGenerateLogRunEIgnoreRevsFileError(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFiles(t, dir, "keep.go")

	cmd := newGenerateLogCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("ignore-revs-file", filepath.Join(dir, "does-not-exist")); err != nil {
		t.Fatal(err)
	}

	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing ignore-revs file, got nil")
	}
	if !strings.Contains(err.Error(), "ignore-revs file") {
		t.Errorf("expected error to mention ignore-revs file, got: %v", err)
	}
}

func TestGenerateLogRejectsNonCSVFormat(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		resetFlags(t)
		outputFormat = format

		cmd := newGenerateLogCmd()
		err := cmd.RunE(cmd, nil)
		if err == nil {
			t.Fatalf("expected error for --format %s, got nil", format)
		}
		if !strings.Contains(err.Error(), "--format") {
			t.Errorf("expected error to mention --format, got: %v", err)
		}
	}
}

func initGenLogRepo(t *testing.T, dir string) {
	t.Helper()
	initGitRepo(t, dir)
	commitFiles(t, dir, "main.go")
}

func TestGenerateLogRunEWritesToFile(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGenLogRepo(t, dir)
	outFile = filepath.Join(t.TempDir(), "out.log")

	cmd := newGenerateLogCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readOutputFile(t, outFile), "main.go") {
		t.Errorf("expected main.go in output file")
	}
}

func TestGenerateLogRunEPropagatesRunGitLogError(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir() // not a git repo

	cmd := newGenerateLogCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for non-repo path, got nil")
	}
	if !strings.Contains(err.Error(), "git log failed") {
		t.Errorf("expected 'git log failed' error, got: %v", err)
	}
}

func TestRunGitLogAfterFiltersCommits(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeRepoFile(t, dir, "old.go", "content\n")
	commitAll(t, dir, "add old.go", "2020-01-01T00:00:00")
	writeRepoFile(t, dir, "new.go", "content\n")
	commitAll(t, dir, "add new.go", "2025-01-01T00:00:00")

	var out bytes.Buffer
	if err := runGitLog(dir, "2022-01-01", "", logFilters{}, &out); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}

	result := out.String()
	if strings.Contains(result, "old.go") {
		t.Errorf("expected old.go (before cutoff) to be excluded, got:\n%s", result)
	}
	if !strings.Contains(result, "new.go") {
		t.Errorf("expected new.go (after cutoff) in output, got:\n%s", result)
	}
}

func TestRunGitLogStartFailsWithoutGitBinary(t *testing.T) {
	t.Setenv("PATH", "")

	err := runGitLog(".", "", "", logFilters{}, io.Discard)
	if err == nil {
		t.Fatal("expected error when git binary is not on PATH, got nil")
	}
	if !strings.Contains(err.Error(), "starting git log") {
		t.Errorf("expected 'starting git log' error, got: %v", err)
	}
}

func TestRunGitLogWriteError(t *testing.T) {
	dir := t.TempDir()
	initGenLogRepo(t, dir)

	err := runGitLog(dir, "", "", logFilters{}, errWriter{})
	if err == nil {
		t.Fatal("expected error from destination write failure, got nil")
	}
	if !strings.Contains(err.Error(), "processing git log output") {
		t.Errorf("expected 'processing git log output' error, got: %v", err)
	}
}

func TestGenerateLogRunEBadOutputPath(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGenLogRepo(t, dir)
	outFile = filepath.Join(t.TempDir(), "does-not-exist", "out.log")

	cmd := newGenerateLogCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for unwritable output path, got nil")
	}
	if !strings.Contains(err.Error(), "creating output file") {
		t.Errorf("expected 'creating output file' error, got: %v", err)
	}
}
