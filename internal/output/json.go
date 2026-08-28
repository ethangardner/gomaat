package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// WriteJSON writes rows as a JSON array of objects to w. The first row is
// treated as the header and used as the object keys for each data row. If
// limit > 0, at most limit data rows (excluding the header) are written.
func WriteJSON(w io.Writer, rows [][]string, limit int) error {
	header, data, ok := splitHeaderData(rows, limit)
	if !ok {
		return nil
	}

	records := make([]map[string]string, len(data))
	for i, row := range data {
		record := make(map[string]string, len(header))
		for j, col := range header {
			if j < len(row) {
				record[col] = row[j]
			}
		}
		records[i] = record
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(records); err != nil {
		return fmt.Errorf("writing json: %w", err)
	}
	return nil
}

// WriteJSONFile writes rows as a JSON array of objects to the named file.
func WriteJSONFile(path string, rows [][]string, limit int) error {
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
	return WriteJSON(f, rows, limit)
}
