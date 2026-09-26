package cli

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gitRun runs git in dir with extra environment variables and returns its
// stdout, failing the test on error.
func gitRun(t *testing.T, dir string, env []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.Output()
	if err != nil {
		var stderr []byte
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			stderr = exitErr.Stderr
		}
		t.Fatalf("git %v: %v\n%s", args, err, stderr)
	}
	return string(out)
}

// realTempDir returns t.TempDir() with symlinks and Windows 8.3 short names
// resolved, so it compares equal to paths git reports (macOS's /var is a
// symlink to /private/var; Windows runners' temp dir is under RUNNER~1).
func realTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// initGitRepo creates an empty repo on branch main with a test identity.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	gitRun(t, dir, nil, "init", "-b", "main")
	gitRun(t, dir, nil, "config", "user.email", "test@example.com")
	gitRun(t, dir, nil, "config", "user.name", "Test")
}

// writeRepoFile writes content to name (relative to dir), creating parent
// directories as needed.
func writeRepoFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// commitAll stages everything and commits it. A non-empty date pins both the
// author and committer dates.
func commitAll(t *testing.T, dir, msg, date string) {
	t.Helper()
	var env []string
	if date != "" {
		env = []string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}
	}
	gitRun(t, dir, nil, "add", "-A")
	gitRun(t, dir, env, "commit", "-m", msg)
}

func TestStreamGitEmptyArgs(t *testing.T) {
	err := streamGit("", nil, func(io.Reader) error { return nil })
	if err == nil {
		t.Fatal("expected error with empty args, got nil")
	}
	if !strings.Contains(err.Error(), "no git arguments provided") {
		t.Errorf("got %v, want no git arguments provided", err)
	}
}

func TestStreamGitPassesStdout(t *testing.T) {
	var got string
	err := streamGit("", []string{"--version"}, func(r io.Reader) error {
		b, err := io.ReadAll(r)
		got = string(b)
		return err
	})
	if err != nil {
		t.Fatalf("streamGit: %v", err)
	}
	if !strings.HasPrefix(got, "git version") {
		t.Errorf("got %q, want git version output", got)
	}
}

func TestStreamGitCommandFailure(t *testing.T) {
	dir := t.TempDir() // not a repo
	err := streamGit(dir, []string{"log"}, func(r io.Reader) error {
		_, err := io.Copy(io.Discard, r)
		return err
	})
	if err == nil {
		t.Fatal("expected error outside a git repo, got nil")
	}
	for _, want := range []string{"git log failed", "not a git repository", "Command: git -C " + dir + " log"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestStreamGitConsumerError(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	for i := range 50 {
		writeRepoFile(t, dir, "f.txt", strings.Repeat("line\n", i+1))
		commitAll(t, dir, "c", "")
	}

	boom := errors.New("boom")
	err := streamGit(dir, []string{"log", "-p"}, func(io.Reader) error { return boom })
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "processing git log output") {
		t.Errorf("got %v, want %v", err, boom)
	}
}
