package cli

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

// resetFlags snapshots the package-level persistent flag vars and restores
// them after the test, since runAnalysis reads these globals directly and
// they would otherwise leak between test cases.
func resetFlags(t *testing.T) {
	t.Helper()
	origLog, origOut, origRows, origGroup, origTeam, origFormat := logFile, outFile, maxRows, groupFile, teamMapFile, outputFormat
	t.Cleanup(func() {
		logFile, outFile, maxRows, groupFile, teamMapFile, outputFormat = origLog, origOut, origRows, origGroup, origTeam, origFormat
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
			if err := ageCmd.Flags().Set("age-time-now", tt.ageTimeNow); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = ageCmd.Flags().Set("age-time-now", "")
			})

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

// TestExecuteErrorPrintedOnce re-runs itself as a child process, since
// Execute calls os.Exit on failure.
func TestExecuteErrorPrintedOnce(t *testing.T) {
	if os.Getenv("GOMAAT_EXECUTE_CHILD") == "1" {
		resetFlags(t)
		logFile = ""
		os.Args = []string{"gomaat", "authors"}
		Execute()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestExecuteErrorPrintedOnce$")
	cmd.Env = append(os.Environ(), "GOMAAT_EXECUTE_CHILD=1")
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var exitErr *exec.ExitError
	if err := cmd.Run(); !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %v", err)
	}
	if n := strings.Count(stderr.String(), "--log (-l) is required"); n != 1 {
		t.Errorf("expected the error once on stderr, got %d times:\n%s", n, stderr.String())
	}
}

func TestAgeBadTimeNow(t *testing.T) {
	resetFlags(t)

	ageCmd, _, err := rootCmd.Find([]string{"age"})
	if err != nil {
		t.Fatalf("finding age command: %v", err)
	}
	if err := ageCmd.Flags().Set("age-time-now", "not-a-date"); err != nil {
		t.Fatalf("setting age-time-now flag: %v", err)
	}
	t.Cleanup(func() {
		_ = ageCmd.Flags().Set("age-time-now", "")
	})

	err = ageCmd.RunE(ageCmd, nil)
	if err == nil {
		t.Fatal("expected error for malformed --age-time-now, got nil")
	}
	if !strings.Contains(err.Error(), "--age-time-now") {
		t.Errorf("expected error to mention --age-time-now, got: %v", err)
	}
}

func TestParseDateFlag(t *testing.T) {
	fallback := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name    string
		value   string
		want    time.Time
		wantErr bool
	}{
		{"valid date", "2024-06-01", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), false},
		{"empty uses fallback", "", fallback, false},
		{"malformed", "06/01/2024", time.Time{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateFlag("--some-date", tt.value, fallback)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), `--some-date: expected YYYY-MM-DD, got "06/01/2024"`) {
					t.Fatalf("got err %v, want --some-date format error", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBusFactorCmdRunE(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"bus-factor"})
	if err != nil {
		t.Fatalf("finding bus-factor command: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := readOutputFile(t, outFile), "foo.go,1,Jane Doe,100.00"; !strings.Contains(got, want) {
		t.Errorf("expected %q in output, got:\n%s", want, got)
	}
}

func TestKnowledgeLossCmdRunE(t *testing.T) {
	formerFile := func(contents string) func(*testing.T) string {
		return func(t *testing.T) string { return testhelpers.WriteTempFile(t, "former.txt", contents) }
	}
	tests := []struct {
		name    string
		former  func(*testing.T) string // returns the --former-authors value
		wantErr string
		wantRow string
	}{
		{name: "missing --former-authors", former: func(*testing.T) string { return "" }, wantErr: "--former-authors is required"},
		{
			name:    "unreadable --former-authors",
			former:  func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing.txt") },
			wantErr: "opening authors file",
		},
		{name: "former author", former: formerFile("author\nJane Doe\n"), wantRow: "foo.go,1,1,100.00"},
		{name: "former author absent from log", former: formerFile("Ghost\n"), wantRow: "foo.go,0,1,0.00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t)
			logFile = validLogFixture(t)
			outFile = filepath.Join(t.TempDir(), "out.csv")

			cmd, _, err := rootCmd.Find([]string{"knowledge-loss"})
			if err != nil {
				t.Fatalf("finding knowledge-loss command: %v", err)
			}
			if err := cmd.Flags().Set("former-authors", tt.former(t)); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = cmd.Flags().Set("former-authors", "")
			})

			err = cmd.RunE(cmd, nil)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := readOutputFile(t, outFile); !strings.Contains(got, tt.wantRow) {
				t.Errorf("expected %q in output, got:\n%s", tt.wantRow, got)
			}
		})
	}
}
