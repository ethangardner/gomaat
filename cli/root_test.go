package cli

import (
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
	origLog, origOut, origRows, origGroup, origTeam := logFile, outFile, maxRows, groupFile, teamMapFile
	t.Cleanup(func() {
		logFile, outFile, maxRows, groupFile, teamMapFile = origLog, origOut, origRows, origGroup, origTeam
	})
}

// validLogFixture returns the path to a minimal, valid log file in the
// format documented in internal/parser/git.go.
func validLogFixture(t *testing.T) string {
	t.Helper()
	return testhelpers.WriteTempFile(t, "valid.log", "--abc123--2024-01-01--Jane Doe\n1\t2\tfoo.go\n\n")
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
