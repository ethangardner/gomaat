package fileutil_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/fileutil"
)

func TestLoad_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	got, err := fileutil.Load(path, "test", func(r io.Reader) (string, error) {
		b, err := io.ReadAll(r)
		return string(b), err
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestLoad_OpenError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.txt")

	_, err := fileutil.Load(path, "custom", func(r io.Reader) (string, error) {
		return "", nil
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "opening custom file:") {
		t.Errorf("expected error to contain 'opening custom file:', got %q", err.Error())
	}
}

func TestLoad_ParseError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	parseErr := errors.New("parse failed")
	_, err := fileutil.Load(path, "test", func(r io.Reader) (string, error) {
		return "", parseErr
	})
	if !errors.Is(err, parseErr) {
		t.Errorf("expected parseErr, got %v", err)
	}
}

func TestLoadLines(t *testing.T) {
	content := `
# Header comment
foo
  bar  

# Another comment
baz
`
	set, err := fileutil.LoadLines(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(set) != 3 {
		t.Fatalf("expected 3 items, got %d", len(set))
	}
	for _, expected := range []string{"foo", "bar", "baz"} {
		if _, ok := set[expected]; !ok {
			t.Errorf("expected set to contain %q", expected)
		}
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read error")
}

func TestLoadLines_Error(t *testing.T) {
	_, err := fileutil.LoadLines(errReader{})
	if err == nil {
		t.Fatal("expected read error, got nil")
	}
}
