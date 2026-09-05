package output

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// failWriter always fails. encoding/csv buffers small writes, so a write
// only reaches failWriter once a field is large enough to force a flush
// (see the size checks in the tests below).
type failWriter struct{}

func (failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestWrite(t *testing.T) {
	rows := [][]string{
		{"entity", "n-revs"},
		{"foo.go", "10"},
		{"bar.go", "5"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "entity,n-revs") {
		t.Errorf("header missing from output: %q", out)
	}
	if !strings.Contains(out, "foo.go,10") || !strings.Contains(out, "bar.go,5") {
		t.Errorf("data rows missing from output: %q", out)
	}
}

func TestWriteRowLimit(t *testing.T) {
	rows := [][]string{
		{"entity"},
		{"a.go"},
		{"b.go"},
		{"c.go"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, rows, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "c.go") {
		t.Errorf("c.go should be truncated by row limit, got: %q", out)
	}
}

func TestWriteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.csv")

	rows := [][]string{{"entity"}, {"foo.go"}}
	if err := WriteFile(path, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "foo.go") {
		t.Errorf("expected foo.go in output file, got: %q", string(data))
	}
}

func TestWriteEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil, 0); err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestWriteHeaderError(t *testing.T) {
	rows := [][]string{{strings.Repeat("x", 5000)}}
	if err := Write(failWriter{}, rows, 0); err == nil {
		t.Fatal("expected error writing header, got nil")
	} else if !strings.Contains(err.Error(), "writing header") {
		t.Errorf("expected error to mention writing header, got: %v", err)
	}
}

func TestWriteRowsError(t *testing.T) {
	rows := [][]string{{"entity"}, {strings.Repeat("y", 5000)}}
	if err := Write(failWriter{}, rows, 0); err == nil {
		t.Fatal("expected error writing rows, got nil")
	} else if !strings.Contains(err.Error(), "writing rows") {
		t.Errorf("expected error to mention writing rows, got: %v", err)
	}
}

func TestWriteFileBadPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist", "out.csv")
	if err := WriteFile(path, [][]string{{"entity"}}, 0); err == nil {
		t.Fatal("expected error for unwritable path, got nil")
	}
}
