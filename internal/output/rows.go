package output

// splitHeaderData splits rows into a header (the first row) and the
// remaining data rows, clamped to at most limit rows when limit > 0. ok is
// false when rows is empty, signaling the caller should write nothing.
func splitHeaderData(rows [][]string, limit int) (header []string, data [][]string, ok bool) {
	if len(rows) == 0 {
		return nil, nil, false
	}
	data = rows[1:]
	if limit > 0 && limit < len(data) {
		data = data[:limit]
	}
	return rows[0], data, true
}
