package parser

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

// nulRecord builds one commit record in the NUL-delimited format generate-log
// produces (see git.go's ParseFile doc comment), so tests can construct
// fixtures without hand-escaping NUL bytes inline.
func nulRecord(rev, date, author, message string, numstatLines ...string) string {
	var sb strings.Builder
	sb.WriteByte(0)
	sb.WriteString(rev)
	sb.WriteByte(0)
	sb.WriteString(date)
	sb.WriteByte(0)
	sb.WriteString(author)
	sb.WriteByte(0)
	sb.WriteString(message)
	sb.WriteString("\n") // %B's own trailing newline
	sb.WriteByte(0)
	sb.WriteString("\n") // git's separator newline before numstat output
	for _, l := range numstatLines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	sb.WriteString("\n") // blank line before the next record
	return sb.String()
}

var sampleLog = nulRecord("abc123", "2024-01-15", "Alice", "add foo and bar",
	"10\t5\tsrc/foo.go",
	"3\t0\tsrc/bar.go",
) + nulRecord("def456", "2024-01-16", "Bob", "tweak foo",
	"-\t-\timage.png",
	"2\t1\tsrc/foo.go",
)

func TestParseReader(t *testing.T) {
	commits, err := ParseReader(strings.NewReader(sampleLog))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 4 {
		t.Fatalf("expected 4 commits, got %d", len(commits))
	}

	wants := []model.Commit{
		{Rev: "abc123", Date: "2024-01-15", Author: "Alice", Entity: "src/foo.go", Message: "add foo and bar", LocAdded: 10, LocDeleted: 5},
		{Rev: "abc123", Date: "2024-01-15", Author: "Alice", Entity: "src/bar.go", Message: "add foo and bar", LocAdded: 3},
		{Rev: "def456", Date: "2024-01-16", Author: "Bob", Entity: "image.png", Message: "tweak foo"}, // binary
		{Rev: "def456", Date: "2024-01-16", Author: "Bob", Entity: "src/foo.go", Message: "tweak foo", LocAdded: 2, LocDeleted: 1},
	}
	for i, want := range wants {
		if commits[i] != want {
			t.Errorf("commit[%d]: got %+v, want %+v", i, commits[i], want)
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
	input := nulRecord("abc123", "2024-01-15", "Alice", "empty commit")
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

func TestParseReaderOldFormatRejected(t *testing.T) {
	// The pre-#60 log format ("--rev--date--author" header lines) contains
	// no NUL bytes at all, so it must be rejected with an explicit,
	// actionable error rather than silently parsing to zero commits.
	oldFormat := "--abc123--2024-01-15--Alice\n10\t5\tsrc/foo.go\n"
	_, err := ParseReader(strings.NewReader(oldFormat))
	if err == nil {
		t.Fatal("expected error for old-format log input, got nil")
	}
	if !strings.Contains(err.Error(), "generate-log") {
		t.Errorf("expected error to mention regenerating via generate-log, got: %v", err)
	}
}

func TestParseReaderMessageWithBlankLinesAndTrailers(t *testing.T) {
	message := "subject line\n\nbody paragraph one.\n\nbody paragraph two.\n\nReviewed-by: Carol\nFixes: #42"
	input := nulRecord("abc123", "2024-01-15", "Alice", message, "1\t1\tsrc/foo.go")

	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != message {
		t.Errorf("message: got %q, want %q", commits[0].Message, message)
	}
}

func TestParseReaderMessageWithEmbeddedDashLine(t *testing.T) {
	// Adversarial case: a message body containing a line that looks exactly
	// like the OLD format's commit header. The NUL-delimited framing must
	// not be fooled by this — the whole message should round-trip intact
	// and no extra/malformed commits should appear.
	message := "refactor parser\n\n--zzz999--2020-01-01--Nobody\nthat line above is just message text, not a new commit"
	input := nulRecord("abc123", "2024-01-15", "Alice", message, "1\t1\tsrc/foo.go")

	commits, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != message {
		t.Errorf("message: got %q, want %q", commits[0].Message, message)
	}
	if commits[0].Rev != "abc123" {
		t.Errorf("rev: got %q, want %q (embedded dash-line must not be parsed as a new header)", commits[0].Rev, "abc123")
	}
}

func TestParseReaderMalformedInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantCommits []model.Commit
	}{
		{
			name:        "incomplete trailing group is dropped",
			input:       "\x00abc123\x002024-01-15\x00Alice", // missing message + numstat fields
			wantCommits: nil,
		},
		{
			name:  "malformed numstat line is skipped",
			input: nulRecord("abc123", "2024-01-15", "Alice", "msg", "not-a-numstat-line", "3\t0\tsrc/bar.go"),
			wantCommits: []model.Commit{
				{Rev: "abc123", Date: "2024-01-15", Author: "Alice", Entity: "src/bar.go", Message: "msg", LocAdded: 3},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commits, err := ParseReader(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(commits) != len(tt.wantCommits) {
				t.Fatalf("got %d commits, want %d", len(commits), len(tt.wantCommits))
			}
			for i, want := range tt.wantCommits {
				if commits[i] != want {
					t.Errorf("commit[%d]: got %+v, want %+v", i, commits[i], want)
				}
			}
		})
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestParseReaderReadError(t *testing.T) {
	_, err := ParseReader(errReader{})
	if err == nil {
		t.Fatal("expected error from reader, got nil")
	}
}
