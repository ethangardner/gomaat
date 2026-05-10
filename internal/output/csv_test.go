package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	rows := [][]string{
		{"entity", "n-revs"},
		{"foo.go", "10"},
		{"bar.go", "5"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "entity,n-revs") {
		t.Errorf("header missing from output: %q", out)
	}
	if !strings.Contains(out, "foo.go,10") || !strings.Contains(out, "bar.go,5") {
		t.Errorf("data rows missing from output: %q", out)
	}
}

func TestWriteRowLimit(t *testing.T) {
	rows := [][]string{
		{"entity"},
		{"a.go"},
		{"b.go"},
		{"c.go"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, rows, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "c.go") {
		t.Errorf("c.go should be truncated by row limit, got: %q", out)
	}
}

func TestWriteFile(t *testing.T) {
	f, err := os.CreateTemp("", "gomaat-out-*.csv")
	if err != nil {
		t.Fatal(err)
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(name) }()

	rows := [][]string{{"entity"}, {"foo.go"}}
	if err := WriteFile(name, rows, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(name)
	if !strings.Contains(string(data), "foo.go") {
		t.Errorf("expected foo.go in output file, got: %q", string(data))
	}
}

func TestWriteEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil, 0); err != nil {
		t.Fatalf("unexpected error on empty input: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}
