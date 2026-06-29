package analysis

import "testing"

func assertFormattedRows(t *testing.T, rows [][]string, wantHeader string, wantCount int) {
	t.Helper()
	if rows[0][0] != wantHeader {
		t.Fatalf("header: got %q, want %q", rows[0][0], wantHeader)
	}
	if len(rows) != wantCount {
		t.Fatalf("row count: got %d, want %d", len(rows), wantCount)
	}
}
