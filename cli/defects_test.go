package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/testhelpers"
)

func defectsLogFixture(t *testing.T) string {
	t.Helper()
	log := nulRecord("r1", "2024-01-01", "Alice", "fix: null pointer crash", "1\t1\tfoo.go") +
		nulRecord("r2", "2024-01-02", "Alice", "add new feature", "2\t0\tfoo.go") +
		nulRecord("r3", "2024-01-03", "Bob", "add bar", "1\t0\tbar.go")
	return testhelpers.WriteTempFile(t, "defects.log", log)
}

func TestDefectsCmdRunE(t *testing.T) {
	resetFlags(t)
	logFile = defectsLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "foo.go") || !strings.Contains(data, "bar.go") {
		t.Errorf("expected foo.go and bar.go in output, got: %q", data)
	}
}

func TestDefectsCmdRunECustomPattern(t *testing.T) {
	resetFlags(t)
	logFile = defectsLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	if err := cmd.Flags().Set("bugfix-pattern", "^fix"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("bugfix-pattern", "") })

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "foo.go,1,2") {
		t.Errorf("expected foo.go with 1 bugfix rev of 2 total, got: %q", data)
	}
}

func TestDefectsCmdRunEConventionalCommitType(t *testing.T) {
	resetFlags(t)
	logFile = defectsLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	if err := cmd.Flags().Set("conventional-commit-type", "fix"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("conventional-commit-type", "") })

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "foo.go,1,2") {
		t.Errorf("expected foo.go with 1 bugfix rev of 2 total, got: %q", data)
	}
}

func TestDefectsCmdRunERejectsBothFlags(t *testing.T) {
	resetFlags(t)
	logFile = defectsLogFixture(t)

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	if err := cmd.Flags().Set("bugfix-pattern", "fix"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("conventional-commit-type", "fix"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("bugfix-pattern", "")
		_ = cmd.Flags().Set("conventional-commit-type", "")
	})

	err = cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error when both flags are set, got nil")
	}
	if !strings.Contains(err.Error(), "only one of") {
		t.Errorf("expected mutual-exclusivity error, got: %v", err)
	}
}

func TestDefectsCmdRunEInvalidRegex(t *testing.T) {
	resetFlags(t)
	logFile = defectsLogFixture(t)

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	if err := cmd.Flags().Set("bugfix-pattern", "("); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("bugfix-pattern", "") })

	err = cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
	if !strings.Contains(err.Error(), "--bugfix-pattern") {
		t.Errorf("expected error to mention --bugfix-pattern, got: %v", err)
	}
}

func TestDefectsCmdRunEMissingLogFlag(t *testing.T) {
	resetFlags(t)
	logFile = ""

	cmd, _, err := rootCmd.Find([]string{"defects"})
	if err != nil {
		t.Fatalf("finding defects command: %v", err)
	}
	err = cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing --log, got nil")
	}
	if !strings.Contains(err.Error(), "--log") {
		t.Errorf("expected error to mention --log, got: %v", err)
	}
}
