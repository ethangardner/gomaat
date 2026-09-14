package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitDistinctFiles writes and commits files with distinct content, so
// gocloc's default MD5-based duplicate-file detection doesn't merge them
// (unlike cli/cloc_test.go's commitFiles helper, which writes identical
// content to every path — fine for cloc's own tests, but not for hotspots'
// tests, which need each file to carry its own, distinguishable LOC count).
func commitDistinctFiles(t *testing.T, dir string, files ...string) {
	t.Helper()
	for i, f := range files {
		p := filepath.Join(dir, f)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		content := "package main\n\nfunc main() {}\n// unique marker " + strings.Repeat("x", i+1) + "\n"
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := exec.Command("git", "-C", dir, "add", "-A").Run(); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "-m", "initial").Run(); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}

func hotspotsLogFixture(t *testing.T) string {
	t.Helper()
	content := "--r1--2024-01-01--Jane Doe\n1\t0\thot.go\n\n" +
		"--r2--2024-01-02--Jane Doe\n1\t0\thot.go\n\n" +
		"--r3--2024-01-03--Jane Doe\n1\t0\tcold.go\n\n"
	p := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestHotspotsRunERanksByChurnAndSize(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitDistinctFiles(t, dir, "hot.go", "cold.go")

	logFile = hotspotsLogFixture(t)
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd := newHotspotsCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := readOutputFile(t, outFile)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 data rows, got %d: %v", len(lines), lines)
	}
	if lines[0] != "entity,revisions,lines,hotspot-score,fractal-value" {
		t.Errorf("unexpected header: %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "hot.go,2,") {
		t.Errorf("expected hot.go (2 revs) ranked first, got: %s", lines[1])
	}
	if !strings.HasPrefix(lines[2], "cold.go,1,") {
		t.Errorf("expected cold.go (1 rev) ranked second, got: %s", lines[2])
	}
}

func TestHotspotsRunEMissingLogFlag(t *testing.T) {
	resetFlags(t)
	logFile = ""

	cmd := newHotspotsCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for missing --log, got nil")
	}
	if !strings.Contains(err.Error(), "--log") {
		t.Errorf("expected error to mention --log, got: %v", err)
	}
}

func TestHotspotsRunENoTrackedFiles(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	logFile = hotspotsLogFixture(t)

	cmd := newHotspotsCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error for repo with no tracked files, got nil")
	}
	if !strings.Contains(err.Error(), "no git-tracked files found") {
		t.Errorf("expected 'no git-tracked files found' error, got: %v", err)
	}
}

func TestHotspotsRunEExcludesFileMissingFromDisk(t *testing.T) {
	resetFlags(t)
	dir := t.TempDir()
	initGitRepo(t, dir)
	commitDistinctFiles(t, dir, "hot.go")

	// Log references a file that no longer exists on disk; it should be
	// silently excluded from the report rather than erroring.
	content := "--r1--2024-01-01--Jane Doe\n1\t0\thot.go\n\n" +
		"--r2--2024-01-02--Jane Doe\n1\t0\tdeleted.go\n\n"
	logPath := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	logFile = logPath
	outFile = filepath.Join(t.TempDir(), "out.csv")

	cmd := newHotspotsCmd()
	if err := cmd.Flags().Set("path", dir); err != nil {
		t.Fatal(err)
	}
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := readOutputFile(t, outFile)
	if strings.Contains(out, "deleted.go") {
		t.Errorf("expected deleted.go to be excluded, got: %s", out)
	}
	if !strings.Contains(out, "hot.go") {
		t.Errorf("expected hot.go present, got: %s", out)
	}
}

func TestHotspotsRunEBadFormat(t *testing.T) {
	resetFlags(t)
	outputFormat = "yaml"
	logFile = hotspotsLogFixture(t)

	cmd := newHotspotsCmd()
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected error for invalid --format, got nil")
	} else if !strings.Contains(err.Error(), "--format") {
		t.Errorf("expected error to mention --format, got: %v", err)
	}
}
