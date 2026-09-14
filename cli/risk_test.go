package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/testhelpers"
)

// coupledLogFixture returns a log file establishing strong temporal coupling
// between a.go and b.go: 5 shared revisions, avg revs 5, degree 100 — well
// above risk's default thresholds (MinRevs=5, MinSharedRevs=5,
// --min-coupling=50).
func coupledLogFixture(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	for i := 1; i <= 5; i++ {
		n := strconv.Itoa(i)
		b.WriteString("--r" + n + "--2024-01-0" + n + "--Jane Doe\n1\t0\ta.go\n1\t0\tb.go\n\n")
	}
	return testhelpers.WriteTempFile(t, "coupled.log", b.String())
}

func TestRiskExactlyOneOfStagedOrDiff(t *testing.T) {
	resetFlags(t)
	logFile = validLogFixture(t)

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("staged", "false")
		_ = cmd.Flags().Set("diff", "")
	})

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error when neither --staged nor --diff is set")
	}

	if err := cmd.Flags().Set("staged", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("diff", "main"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error when both --staged and --diff are set")
	}
}

func TestRiskRequiresLog(t *testing.T) {
	resetFlags(t)
	logFile = ""

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	if err := cmd.Flags().Set("staged", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Flags().Set("staged", "false") })

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for missing --log, got nil")
	}
}

func TestRiskStagedFlagsMissingCoupledFile(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFiles(t, dir, "a.go", "b.go")

	// Stage a change to a.go only; b.go stays untouched.
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "add", "a.go").Run(); err != nil {
		t.Fatal(err)
	}

	logFile = coupledLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	if err := cmd.Flags().Set("staged", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("staged", "false")
		_ = cmd.Flags().Set("path", ".")
	})

	runErr := cmd.RunE(cmd, nil)
	if runErr == nil {
		t.Fatal("expected non-nil error (findings present) for a staged change missing its coupled partner")
	}

	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "a.go") || !strings.Contains(data, "b.go") {
		t.Errorf("expected a.go/b.go in output, got: %q", data)
	}
}

func TestRiskStagedNoConcernsWhenBothChanged(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFiles(t, dir, "a.go", "b.go")

	for _, f := range []string{"a.go", "b.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := exec.Command("git", "-C", dir, "add", "a.go", "b.go").Run(); err != nil {
		t.Fatal(err)
	}

	logFile = coupledLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	if err := cmd.Flags().Set("staged", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("staged", "false")
		_ = cmd.Flags().Set("path", ".")
	})

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("expected no error when both coupled files changed, got: %v", err)
	}
}

func TestRiskDiffAgainstRef(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitFiles(t, dir, "a.go", "b.go")

	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "add", "-A").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "-m", "modify a.go").Run(); err != nil {
		t.Fatal(err)
	}

	logFile = coupledLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	if err := cmd.Flags().Set("diff", "HEAD~1"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("diff", "")
		_ = cmd.Flags().Set("path", ".")
	})

	runErr := cmd.RunE(cmd, nil)
	if runErr == nil {
		t.Fatal("expected non-nil error (findings present) for a diff missing the coupled partner")
	}

	data := readOutputFile(t, outFile)
	if !strings.Contains(data, "a.go") || !strings.Contains(data, "b.go") {
		t.Errorf("expected a.go/b.go in output, got: %q", data)
	}
}

func TestRiskNonRepoPath(t *testing.T) {
	resetFlags(t)
	logFile = coupledLogFixture(t)
	dir := t.TempDir()

	cmd, _, err := rootCmd.Find([]string{"risk"})
	if err != nil {
		t.Fatalf("finding risk command: %v", err)
	}
	if err := cmd.Flags().Set("staged", "true"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Flags().Set("staged", "false")
		_ = cmd.Flags().Set("path", ".")
	})

	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for non-repo path, got nil")
	}
}
