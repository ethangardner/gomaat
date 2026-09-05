package teammapper

import (
	"path/filepath"
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

func TestApplyEmptyLookup(t *testing.T) {
	commits := []model.Commit{{Author: "Alice"}}
	result := Apply(commits, nil)
	if len(result) != 1 {
		t.Errorf("empty lookup should return all commits unchanged, got %d", len(result))
	}
}
