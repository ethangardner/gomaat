package parser

import (
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
