package teammapper

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

func TestLoad(t *testing.T) {
	csv := `author,team
Alice Smith,Backend
Bob Jones,Frontend
`
	lookup, err := load(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lookup) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lookup))
	}
	if lookup["Alice Smith"] != "Backend" {
		t.Errorf("expected Backend, got %q", lookup["Alice Smith"])
	}
	if lookup["Bob Jones"] != "Frontend" {
		t.Errorf("expected Frontend, got %q", lookup["Bob Jones"])
	}
}

func TestLoadSkipsHeader(t *testing.T) {
	// header row should not be treated as an author entry
	csv := "author,team\nAlice,Backend\n"
	lookup, _ := load(strings.NewReader(csv))
	if _, ok := lookup["author"]; ok {
		t.Error("header row should be skipped, but 'author' was added to lookup")
	}
}

func TestLoadNoHeader(t *testing.T) {
	csv := "Alice,Backend\nBob,Frontend\n"
	lookup, _ := load(strings.NewReader(csv))
	if len(lookup) != 2 {
		t.Fatalf("expected 2 entries without header, got %d", len(lookup))
	}
}

func TestApply(t *testing.T) {
	lookup := map[string]string{"Alice": "Backend", "Bob": "Frontend"}
	commits := []model.Commit{
		{Author: "Alice", Entity: "foo.go"},
		{Author: "Bob", Entity: "bar.go"},
		{Author: "Carol", Entity: "baz.go"}, // unmapped → excluded
	}
	result := Apply(commits, lookup)

	if len(result) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(result))
	}
	if result[0].Author != "Backend" {
		t.Errorf("expected author 'Backend', got %q", result[0].Author)
	}
	if result[1].Author != "Frontend" {
		t.Errorf("expected author 'Frontend', got %q", result[1].Author)
	}
}

func TestLoadFile(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "teams.csv", "Alice,Backend\nBob,Frontend\n")

	lookup, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lookup) != 2 {
		t.Errorf("expected 2 entries, got %d", len(lookup))
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "does-not-exist.csv"))
	if err == nil {
		t.Fatal("expected error for missing team map file, got nil")
	}
}

func TestLoadMalformedCSV(t *testing.T) {
	// An unterminated quoted field is a CSV syntax error.
	_, err := load(strings.NewReader(`Alice,"Backend` + "\n"))
	if err == nil {
		t.Fatal("expected error for malformed CSV, got nil")
	}
	if !strings.Contains(err.Error(), "reading team map") {
		t.Errorf("expected error to mention reading team map, got: %v", err)
	}
}

func TestLoadSkipsShortRecords(t *testing.T) {
	// Every row has a single field, so encoding/csv's FieldsPerRecord check
	// (derived from the first row) doesn't trip; each row is instead
	// dropped by teammapper's own len(record) < 2 check.
	lookup, err := load(strings.NewReader("Alice\nBob\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lookup) != 0 {
		t.Errorf("expected single-column rows to be skipped, got %v", lookup)
	}
}

func TestLoadRejectsMixedFieldCounts(t *testing.T) {
	// A row missing its team is a typo, not something to skip silently.
	_, err := load(strings.NewReader("Alice,Backend\nBob\n"))
	if err == nil || !strings.Contains(err.Error(), "reading team map") {
		t.Fatalf("expected reading team map error, got %v", err)
	}
}

func TestApplyEmptyLookup(t *testing.T) {
	commits := []model.Commit{{Author: "Alice"}}
	result := Apply(commits, nil)
	if len(result) != 1 {
		t.Errorf("empty lookup should return all commits unchanged, got %d", len(result))
	}
}

func TestLoadAuthors(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain list", "Alice\nBob\n", []string{"Alice", "Bob"}},
		{"header row skipped", "author\nAlice\n", []string{"Alice"}},
		{"comments and blank lines skipped", "# alumni\nAlice\n\n  Bob  \n", []string{"Alice", "Bob"}},
		{"extra columns ignored", "author,left\nAlice,2024-01-01\nBob\n", []string{"Alice", "Bob"}},
		{"bare quotes kept", "Bob \"The Builder\" Smith\n", []string{`Bob "The Builder" Smith`}},
		{"quoted comma", "\"Doe, Jane\"\n", []string{"Doe, Jane"}},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authors, err := loadAuthors(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := slices.Sorted(maps.Keys(authors))
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadAuthorsFile(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "former.txt", "Alice\nBob\n")
	authors, err := LoadAuthorsFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(authors) != 2 {
		t.Errorf("expected 2 authors, got %d", len(authors))
	}
}

func TestLoadAuthorsFileNotFound(t *testing.T) {
	_, err := LoadAuthorsFile(filepath.Join(t.TempDir(), "does-not-exist.txt"))
	if err == nil || !strings.Contains(err.Error(), "opening authors file") {
		t.Fatalf("expected opening authors file error, got %v", err)
	}
}
