package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

// resetFlags snapshots the package-level persistent flag vars and restores
// them after the test, since runAnalysis reads these globals directly and
// they would otherwise leak between test cases.
func resetFlags(t *testing.T) {
	t.Helper()
	origLog, origRepo, origOut, origRows, origGroup, origTeam, origFormat := logFile, repoPath, outFile, maxRows, groupFile, teamMapFile, outputFormat
	origAfter, origBefore, origExcludes, origExcludeAuthors, origIgnoreRevsFile, origUseMailmap := after, before, excludes, excludeAuthors, ignoreRevsFile, useMailmap
	origHalfLife, origAgeTimeNow := halfLifeDays, ageTimeNow
	t.Cleanup(func() {
		logFile, repoPath, outFile, maxRows, groupFile, teamMapFile, outputFormat = origLog, origRepo, origOut, origRows, origGroup, origTeam, origFormat
		after, before, excludes, excludeAuthors, ignoreRevsFile, useMailmap = origAfter, origBefore, origExcludes, origExcludeAuthors, origIgnoreRevsFile, origUseMailmap
		halfLifeDays, ageTimeNow = origHalfLife, origAgeTimeNow
	})
}

// validLogFixture returns the path to a minimal, valid log file in the
// format documented in internal/parser/git.go.
func validLogFixture(t *testing.T) string {
	t.Helper()
	return testhelpers.WriteTempFile(t, "valid.log", "--abc123--2024-01-01--Jane Doe\n1\t2\tfoo.go\n\n")
}

func readOutputFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	return string(data)
}

func TestRunAnalysisMissingLogFlag(t *testing.T) {
	resetFlags(t)
	logFile = ""

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error for missing --log, got nil")
	}
	if !strings.Contains(err.Error(), "--log") {
		t.Errorf("expected error to mention --log, got: %v", err)
	}
}

func TestRunAnalysisLogFileNotFound(t *testing.T) {
	resetFlags(t)
	logFile = filepath.Join(t.TempDir(), "does-not-exist.log")

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error for missing log file, got nil")
	}
}

func TestRunAnalysisBadGroupFile(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	groupFile = testhelpers.WriteTempFile(t, "bad-group.txt", "^[invalid => Bad")

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error for invalid --group regex, got nil")
	}
}

func TestRunAnalysisBadTeamMapFile(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	teamMapFile = filepath.Join(t.TempDir(), "does-not-exist.csv")

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error for missing --team-map-file, got nil")
	}
}

func TestRunAnalysisBadFormat(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	outputFormat = "yaml"

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error for invalid --format, got nil")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Errorf("expected error to mention --format, got: %v", err)
	}
}

func TestRunAnalysisJSONFormat(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	outputFormat = "json"
	outFile = filepath.Join(t.TempDir(), "out.json")

	if err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	var got []map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected at least one JSON record, got none")
	}
}

func TestRunAnalysisValidGroupFile(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	groupFile = testhelpers.WriteTempFile(t, "groups.txt", `^foo\.go$ => FooGroup`+"\n")
	outFile = filepath.Join(t.TempDir(), "out.csv")

	if err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readOutputFile(t, outFile), "FooGroup") {
		t.Errorf("expected entity remapped to FooGroup")
	}
}

func TestRunAnalysisValidTeamMapFile(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	teamMapFile = testhelpers.WriteTempFile(t, "teams.csv", "Jane Doe,TeamA\n")
	outFile = filepath.Join(t.TempDir(), "out.csv")

	if err := runAnalysis(analysis.AuthorChurn, analysis.FormatAuthorChurn, model.Options{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readOutputFile(t, outFile), "TeamA") {
		t.Errorf("expected author remapped to TeamA")
	}
}

func TestRunAnalysisWritesToStdout(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	runErr := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	_ = w.Close()
	os.Stdout = origStdout

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "foo.go") {
		t.Errorf("expected foo.go on stdout, got: %q", buf.String())
	}
}

func TestSimpleCmdRunE(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"authors"})
	if err != nil {
		t.Fatalf("finding authors command: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(readOutputFile(t, outFile), "foo.go") {
		t.Errorf("expected foo.go in output")
	}
}

func TestCouplingCmdRunE(t *testing.T) {
	resetFlags(t)
	logFile = testhelpers.WriteTempFile(t, "coupling.log", strings.Join([]string{
		"--abc123--2024-01-01--Jane Doe",
		"1\t0\tfoo.go",
		"1\t0\tbar.go",
		"",
	}, "\n"))
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"coupling"})
	if err != nil {
		t.Fatalf("finding coupling command: %v", err)
	}
	for _, flag := range []struct{ name, val string }{
		{"min-revs", "1"},
		{"min-shared-revs", "1"},
		{"min-coupling", "0"},
	} {
		if err := cmd.Flags().Set(flag.name, flag.val); err != nil {
			t.Fatal(err)
		}
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "foo.go") || !strings.Contains(data, "bar.go") {
		t.Errorf("expected coupled entities in output, got: %q", data)
	}
}

func TestAgeCmdRunE(t *testing.T) {
	tests := []struct {
		name       string
		ageTimeNow string
	}{
		{"valid age-time-now", "2024-06-01"},
		{"defaults to now", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			logFile = validLogFixture(t)
			outFile = filepath.Join(t.TempDir(), "out.csv")

			ageCmd, _, err := rootCmd.Find([]string{"age"})
			if err != nil {
				t.Fatalf("finding age command: %v", err)
			}
			ageTimeNow = tt.ageTimeNow

			if err := ageCmd.RunE(ageCmd, nil); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(readOutputFile(t, outFile), "foo.go") {
				t.Errorf("expected foo.go in output")
			}
		})
	}
}

func TestExecuteSuccess(t *testing.T) {
	resetFlags(t)
	logFile = ""
	outFile = ""
	fixture := validLogFixture(t)
	outPath := filepath.Join(t.TempDir(), "out.csv")

	origArgs := os.Args
	t.Cleanup(func() { os.Args = origArgs })
	os.Args = []string{"gomaat", "authors", "--log", fixture, "--outfile", outPath}

	Execute()

	if !strings.Contains(readOutputFile(t, outPath), "foo.go") {
		t.Errorf("expected foo.go in output")
	}
}

func TestRunAnalysisRepoAndLogMutuallyExclusive(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	repoPath = "."

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error when both --log and --repo are set, got nil")
	}
	if !strings.Contains(err.Error(), "--log") || !strings.Contains(err.Error(), "--repo") {
		t.Errorf("expected error to mention both --log and --repo, got: %v", err)
	}
}

func TestRunAnalysisNeitherLogNorRepo(t *testing.T) {
	resetFlags(t)

	err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{})
	if err == nil {
		t.Fatal("expected error when neither --log nor --repo is set, got nil")
	}
	if !strings.Contains(err.Error(), "--log") || !strings.Contains(err.Error(), "--repo") {
		t.Errorf("expected error to mention both --log and --repo, got: %v", err)
	}
}

func TestRunAnalysisRepoPath(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFiles(t, dir, "foo.go", "bar.go")

	// Run via --repo directly.
	resetFlags(t)
	repoPath = dir
	outFile = filepath.Join(t.TempDir(), "via-repo.csv")
	if err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{}); err != nil {
		t.Fatalf("unexpected error via --repo: %v", err)
	}
	viaRepo := readOutputFile(t, outFile)

	// Run via generate-log + -l for comparison.
	resetFlags(t)
	repoPath = ""
	logPath := filepath.Join(t.TempDir(), "generated.log")
	var buf bytes.Buffer
	if err := runGitLog(dir, "", "", logFilters{}, &buf); err != nil {
		t.Fatalf("runGitLog: %v", err)
	}
	if err := os.WriteFile(logPath, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	logFile = logPath
	outFile = filepath.Join(t.TempDir(), "via-log.csv")
	if err := runAnalysis(analysis.Authors, analysis.FormatAuthors, model.Options{}); err != nil {
		t.Fatalf("unexpected error via -l: %v", err)
	}
	viaLog := readOutputFile(t, outFile)

	if viaRepo != viaLog {
		t.Errorf("--repo output differs from generate-log + -l output:\n--repo:\n%s\n-l:\n%s", viaRepo, viaLog)
	}
}

func TestHalfLifeFlagEndToEnd(t *testing.T) {
	resetFlags(t)
	logFile = testhelpers.WriteTempFile(t, "half-life.log", strings.Join([]string{
		"--abc123--2024-01-01--Jane Doe",
		"1\t0\tfoo.go",
		"",
	}, "\n"))
	outFile = filepath.Join(t.TempDir(), "out.csv")
	halfLifeDays = 30
	if err := rootCmd.PersistentFlags().Set("age-time-now", "2024-01-31"); err != nil {
		t.Fatalf("setting age-time-now flag: %v", err)
	}
	t.Cleanup(func() {
		_ = rootCmd.PersistentFlags().Set("age-time-now", "")
	})

	cmd, _, err := rootCmd.Find([]string{"revisions"})
	if err != nil {
		t.Fatalf("finding revisions command: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data := readOutputFile(t, outFile); data != "entity,n-revs\nfoo.go,0.50\n" {
		t.Errorf("unexpected decay-weighted output: %q", data)
	}
}

func TestHalfLifeFlagBadInput(t *testing.T) {
	resetFlags(t)

	err := rootCmd.PersistentFlags().Set("half-life", "not-a-number")
	if err == nil {
		t.Fatal("expected error setting --half-life to a non-numeric value, got nil")
	}
	if !strings.Contains(err.Error(), "half-life") {
		t.Errorf("expected error to mention half-life, got: %v", err)
	}
}

func TestAgeBadTimeNow(t *testing.T) {
	resetFlags(t)

	ageCmd, _, err := rootCmd.Find([]string{"age"})
	if err != nil {
		t.Fatalf("finding age command: %v", err)
	}
	ageTimeNow = "not-a-date"

	err = ageCmd.RunE(ageCmd, nil)
	if err == nil {
		t.Fatal("expected error for malformed --age-time-now, got nil")
	}
	if !strings.Contains(err.Error(), "--age-time-now") {
		t.Errorf("expected error to mention --age-time-now, got: %v", err)
	}
}
