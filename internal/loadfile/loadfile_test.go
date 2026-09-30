package loadfile

import (
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/testhelpers"
)

func readAll(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}

func TestParse(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "sample.txt", "hello world")
	got, err := Parse(path, "sample", readAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestParseFileNotFound(t *testing.T) {
	_, err := Parse(filepath.Join(t.TempDir(), "does-not-exist.txt"), "sample", readAll)
	if err == nil || !strings.Contains(err.Error(), "opening sample file") {
		t.Fatalf("expected opening sample file error, got %v", err)
	}
}

func TestParseParserError(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "sample.txt", "data")
	parseErr := errors.New("parse failed")
	_, err := Parse(path, "sample", func(io.Reader) (string, error) { return "", parseErr })
	if !errors.Is(err, parseErr) {
		t.Errorf("expected parse error, got %v", err)
	}
}

// Excel's "CSV UTF-8" export and some Windows editors start files with a
// UTF-8 byte order mark, which would otherwise stick to the first value.
func TestParseStripsUTF8BOM(t *testing.T) {
	tests := []struct{ name, content, want string }{
		{"leading BOM", "\uFEFFAlice\nBob\n", "Alice\nBob\n"},
		{"no BOM", "Alice\n", "Alice\n"},
		{"BOM only", "\uFEFF", ""},
		{"BOM later in the file is kept", "Alice\uFEFF\n", "Alice\uFEFF\n"},
		{"empty file", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := testhelpers.WriteTempFile(t, "sample.txt", tt.content)
			got, err := Parse(path, "sample", readAll)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
