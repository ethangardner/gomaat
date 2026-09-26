package fileutil

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

func TestLoad(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "sample.txt", "hello world")
	got, err := Load(path, "sample", readAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello world" {
		t.Errorf("got %q, want %q", got, "hello world")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "does-not-exist.txt"), "sample", readAll)
	if err == nil || !strings.Contains(err.Error(), "opening sample file") {
		t.Fatalf("expected opening sample file error, got %v", err)
	}
}

func TestLoadParseError(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "sample.txt", "data")
	parseErr := errors.New("parse failed")
	_, err := Load(path, "sample", func(io.Reader) (string, error) { return "", parseErr })
	if !errors.Is(err, parseErr) {
		t.Errorf("expected parse error, got %v", err)
	}
}
