// Package loadfile opens input files and hands them to a parser, so every
// gomaat loader reports open and close errors the same way.
package loadfile

import (
	"fmt"
	"io"
	"os"
)

// Parse opens path and hands the open file to parse; what names the file in error messages.
func Parse[T any](path, what string, parse func(io.Reader) (T, error)) (T, error) {
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
