package parser

import (
	"os"
	"strings"
	"testing"
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

	tests := []struct {
		idx        int
		rev        string
		date       string
		author     string
		entity     string
		locAdded   int
		locDeleted int
	}{
		{0, "abc123", "2024-01-15", "Alice", "src/foo.go", 10, 5},
		{1, "abc123", "2024-01-15", "Alice", "src/bar.go", 3, 0},
		{2, "def456", "2024-01-16", "Bob", "image.png", 0, 0}, // binary
		{3, "def456", "2024-01-16", "Bob", "src/foo.go", 2, 1},
	}
	for _, tt := range tests {
		c := commits[tt.idx]
		if c.Rev != tt.rev || c.Date != tt.date || c.Author != tt.author ||
			c.Entity != tt.entity || c.LocAdded != tt.locAdded || c.LocDeleted != tt.locDeleted {
			t.Errorf("commit[%d]: got {%s %s %s %s %d %d}, want {%s %s %s %s %d %d}",
				tt.idx, c.Rev, c.Date, c.Author, c.Entity, c.LocAdded, c.LocDeleted,
				tt.rev, tt.date, tt.author, tt.entity, tt.locAdded, tt.locDeleted)
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
	f, err := os.CreateTemp("", "gomaat-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(sampleLog)
	f.Close()

	commits, err := ParseFile(f.Name())
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
