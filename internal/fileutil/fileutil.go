package fileutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
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

// LoadLines reads non-empty, non-comment (#) lines from r into a set.
func LoadLines(r io.Reader) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return set, nil
}
