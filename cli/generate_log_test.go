package cli

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestMatchesExcludePattern(t *testing.T) {
	tests := []struct {
		path    string
		pattern string
		want    bool
	}{
		// directory prefix
		{"vendor/github.com/foo/bar.go", "vendor/", true},
		{"vendor/foo.go", "vendor/", true},
		{"src/vendor/foo.go", "vendor/", false},
		// glob against base name
		{"src/api/types.pb.go", "*.pb.go", true},
		{"src/api/types.go", "*.pb.go", false},
		// glob against full path
		{"src/generated/types.pb.go", "src/generated/*.pb.go", true},
		{"src/other/types.pb.go", "src/generated/*.pb.go", false},
		// exact match
		{"go.sum", "go.sum", true},
		{"go.mod", "go.sum", false},
	}

	for _, tt := range tests {
		got := matchesExcludePattern(tt.path, tt.pattern)
		if got != tt.want {
			t.Errorf("matchesExcludePattern(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestFilterExcludes(t *testing.T) {
	input := strings.Join([]string{
		"--abc123--2024-01-15--Alice",
		"5\t3\tsrc/foo.go",
		"2\t1\tvendor/github.com/lib/lib.go",
		"1\t0\tsrc/types.pb.go",
		"",
		"--def456--2024-02-01--Bob",
		"3\t2\tsrc/bar.go",
		"4\t0\tsrc/api/gen.pb.go",
		"",
	}, "\n")

	var out bytes.Buffer
	if err := filterExcludesStream(strings.NewReader(input), &out, []string{"vendor/", "*.pb.go"}); err != nil {
		t.Fatalf("filterExcludesStream: %v", err)
	}
	result := out.String()

	kept := []string{"src/foo.go", "src/bar.go", "--abc123", "--def456"}
	for _, s := range kept {
		if !strings.Contains(result, s) {
			t.Errorf("expected %q to be kept in output, but it was removed:\n%s", s, result)
		}
	}

	removed := []string{"vendor/github.com/lib/lib.go", "src/types.pb.go", "src/api/gen.pb.go"}
	for _, s := range removed {
		if strings.Contains(result, s) {
			t.Errorf("expected %q to be excluded from output, but it was kept:\n%s", s, result)
		}
	}
}

func TestFilterExcludesNoPatterns(t *testing.T) {
	input := "5\t3\tvendor/foo.go\n"
	var out bytes.Buffer
	if err := filterExcludesStream(strings.NewReader(input), &out, nil); err != nil {
		t.Fatalf("filterExcludesStream: %v", err)
	}
	if out.String() != input {
		t.Errorf("filterExcludesStream with no patterns should return input unchanged")
	}
}

func TestNumstatLineMatchesExclude(t *testing.T) {
	tests := []struct {
		line     string
		excludes []string
		want     bool
	}{
		{"5\t3\tvendor/foo.go", []string{"vendor/"}, true},
		{"5\t3\tsrc/foo.go", []string{"vendor/"}, false},
		// commit header lines are never excluded
		{"--abc--2024-01-01--Alice", []string{"vendor/"}, false},
		// blank lines are never excluded
		{"", []string{"vendor/"}, false},
	}

	for _, tt := range tests {
		got := numstatLineMatchesExclude(tt.line, tt.excludes)
		if got != tt.want {
			t.Errorf("numstatLineMatchesExclude(%q, %v) = %v, want %v", tt.line, tt.excludes, got, tt.want)
		}
	}
}

func TestFilterExcludesStreamPreservesTrailingNewline(t *testing.T) {
	input := "1\t0\tvendor/foo.go\n2\t0\tsrc/keep.go\n"

	var out bytes.Buffer
	if err := filterExcludesStream(strings.NewReader(input), &out, []string{"vendor/"}); err != nil {
		t.Fatalf("filterExcludesStream: %v", err)
	}

	if out.String() != "2\t0\tsrc/keep.go\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestFilterExcludesStreamWriterError(t *testing.T) {
	err := filterExcludesStream(strings.NewReader("1\t0\tsrc/keep.go\n"), errWriter{}, nil)
	if err == nil {
		t.Fatal("expected writer error, got nil")
	}
}

type errWriter struct{}

func (errWriter) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func TestFilterExcludesStreamReaderError(t *testing.T) {
	err := filterExcludesStream(errReader{}, &bytes.Buffer{}, nil)
	if err == nil {
		t.Fatal("expected reader error, got nil")
	}
}

type errReader struct{}

func (errReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}

func TestFilterExcludesStreamCarriageReturn(t *testing.T) {
	input := "1\t0\tvendor/foo.go\r\n2\t0\tsrc/keep.go\r\n"

	var out bytes.Buffer
	if err := filterExcludesStream(strings.NewReader(input), &out, []string{"vendor/"}); err != nil {
		t.Fatalf("filterExcludesStream: %v", err)
	}

	if out.String() != "2\t0\tsrc/keep.go\r\n" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
