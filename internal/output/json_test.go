package output

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rows := [][]string{
		{"entity", "n-revs"},
		{"foo.go", "10"},
		{"bar.go", "5"},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	want := []map[string]string{
		{"entity": "foo.go", "n-revs": "10"},
		{"entity": "bar.go", "n-revs": "5"},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d records, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		for k, v := range want[i] {
			if got[i][k] != v {
				t.Errorf("record %d: expected %s=%q, got %q", i, k, v, got[i][k])
			}
		}
	}
}

func TestWriteJSONRowLimit(t *testing.T) {
	rows := [][]string{
		{"entity"},
		{"a.go"},
		{"b.go"},
		{"c.go"},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 records after limit, got %d: %v", len(got), got)
	}
}

func TestWriteJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	rows := [][]string{{"entity"}, {"foo.go"}}
	if err := WriteJSONFile(path, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}

	var got []map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if len(got) != 1 || got[0]["entity"] != "foo.go" {
		t.Errorf("expected [{entity: foo.go}], got %v", got)
	}
}

func TestWriteJSONRowShorterThanHeader(t *testing.T) {
	rows := [][]string{
		{"entity", "n-revs"},
		{"foo.go"},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows, 0); err == nil {
		t.Fatal("expected error for row shorter than header, got nil")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output written on error, got %q", buf.String())
	}
}

func TestWriteJSONRowLongerThanHeader(t *testing.T) {
	rows := [][]string{
		{"entity"},
		{"foo.go", "10"},
	}
	var buf bytes.Buffer
	if err := WriteJSON(&buf, rows, 0); err == nil {
		t.Fatal("expected error for row longer than header, got nil")
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output written on error, got %q", buf.String())
	}
}

func TestWriteJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, nil, 0); err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}
