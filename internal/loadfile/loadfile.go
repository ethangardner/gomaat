// Package loadfile opens input files and hands them to a parser, so every
// gomaat loader reports open and close errors the same way.
package loadfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

// utf8BOM is the byte order mark Excel's "CSV UTF-8" export and some Windows
// editors put at the start of a file.
var utf8BOM = []byte("\uFEFF")

// Parse opens path and hands its contents to parse, without a leading UTF-8
// byte order mark, which would otherwise stick to the first value (a BOM'd
// "Alice" would never match an author named Alice). what names the file in
// error messages.
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
	return parse(skipBOM(f))
}

// skipBOM returns r with a leading UTF-8 byte order mark removed, if any.
func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	if head, _ := br.Peek(len(utf8BOM)); bytes.Equal(head, utf8BOM) {
		_, _ = br.Discard(len(utf8BOM))
	}
	return br
}
