// Package fileutil holds file-handling helpers shared by gomaat's loaders.
package fileutil

import (
	"fmt"
	"io"
	"os"
)

// Load opens path and hands the open file to parse; what names the file in error messages.
func Load[T any](path, what string, parse func(io.Reader) (T, error)) (T, error) {
	f, err := os.Open(path)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("opening %s file: %w", what, err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error closing %s file %s: %v\n", what, path, closeErr)
		}
	}()
	return parse(f)
}
