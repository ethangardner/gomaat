package parser

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

const sampleLog = `--abc123--2024-01-15--Alice
10	5	src/foo.go
3	0	src/bar.go

--def456--2024-01-16--Bob
-	-	image.png
2	1	src/foo.go
`

func TestParseReader(t *testing.T) {
	commits, err := ParseReader(strings.NewReader(sampleLog))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 4 {
		t.Fatalf("expected 4 commits, got %d", len(commits))
	}

	wants := []model.Commit{
		{Rev: "abc123", Date: "2024-01-15", Author: "Alice", Entity: "src/foo.go", LocAdded: 10, LocDeleted: 5},
		{Rev: "abc123", Date: "2024-01-15", Author: "Alice", Entity: "src/bar.go", LocAdded: 3},
		{Rev: "def456", Date: "2024-01-16", Author: "Bob", Entity: "image.png"}, // binary
		{Rev: "def456", Date: "2024-01-16", Author: "Bob", Entity: "src/foo.go", LocAdded: 2, LocDeleted: 1},
	}
	for i, want := range wants {
		if commits[i] != want {
			t.Errorf("commit[%d]: got %v, want %v", i, commits[i], want)
		}
	}
}

func TestParseReaderEmpty(t *testing.T) {
	commits, err := ParseReader(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}

func TestParseFile(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "test.log", sampleLog)

	commits, err := ParseFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 4 {
		t.Errorf("expected 4 commits, got %d", len(commits))
	}
}

func TestParseReaderHeaderOnly(t *testing.T) {
	input := "--abc123--2024-01-15--Alice\n"
	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits (no file lines), got %d", len(commits))
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := ParseFile(filepath.Join(t.TempDir(), "does-not-exist.log"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseReaderMalformedHeader(t *testing.T) {
	// A line starting with "--" that doesn't split into 4 parts is skipped,
	// and any numstat lines that follow it are dropped since currentRev is
	// never set.
	input := "--not-enough-dashes\n10\t5\tsrc/foo.go\n"
	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for malformed header, got %d", len(commits))
	}
}

func TestParseReaderNumstatBeforeHeader(t *testing.T) {
	input := "10\t5\tsrc/foo.go\n--abc123--2024-01-15--Alice\n3\t0\tsrc/bar.go\n"
	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 || commits[0].Entity != "src/bar.go" {
		t.Errorf("expected only src/bar.go to be parsed, got %v", commits)
	}
}

func TestParseReaderMalformedNumstatLine(t *testing.T) {
	input := "--abc123--2024-01-15--Alice\nnot-a-numstat-line\n3\t0\tsrc/bar.go\n"
	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 || commits[0].Entity != "src/bar.go" {
		t.Errorf("expected only src/bar.go to be parsed, got %v", commits)
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestParseReaderScannerError(t *testing.T) {
	_, err := ParseReader(errReader{})
	if err == nil {
		t.Fatal("expected error from scanner, got nil")
	}
}
