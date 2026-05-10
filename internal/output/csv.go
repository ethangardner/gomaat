package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// Write writes rows as CSV to w. The first row is treated as the header.
// If limit > 0, at most limit data rows (excluding the header) are written.
func Write(w io.Writer, rows [][]string, limit int) error {
	if len(rows) == 0 {
		return nil
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(rows[0]); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}
	data := rows[1:]
	if limit > 0 && limit < len(data) {
		data = data[:limit]
	}
	if err := cw.WriteAll(data); err != nil {
		return fmt.Errorf("writing rows: %w", err)
	}
	cw.Flush()
	return cw.Error()
}

// WriteFile writes rows as CSV to the named file.
func WriteFile(path string, rows [][]string, limit int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error closing output file %s: %v\n", path, err)
		}
	}(f)
	return Write(f, rows, limit)
}
