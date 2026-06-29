package testhelpers

import (
	"os"
	"path/filepath"
	"testing"
)

// WriteTempFile writes content to a temp file inside t.TempDir() and returns
// the path. The file is automatically removed when the test completes.
func WriteTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
