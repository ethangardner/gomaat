package teammapper

import (
	"strings"
	"testing"

	"gomaat/internal/model"
	"gomaat/internal/testhelpers"
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

func TestApplyEmptyLookup(t *testing.T) {
	commits := []model.Commit{{Author: "Alice"}}
	result := Apply(commits, nil)
	if len(result) != 1 {
		t.Errorf("empty lookup should return all commits unchanged, got %d", len(result))
	}
}
