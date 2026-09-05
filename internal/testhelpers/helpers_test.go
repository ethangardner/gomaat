package testhelpers

import (
	"os"
	"testing"
)

func TestWriteTempFile(t *testing.T) {
	path := WriteTempFile(t, "sample.txt", "hello")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading temp file: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected content %q, got %q", "hello", string(data))
	}
}
