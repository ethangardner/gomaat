package grouper

import (
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/testhelpers"
)

func TestLoad(t *testing.T) {
	spec := `
# comment line
src/api   => API
src/core  => Core
^src/gen/ => Generated
`
	groups, err := load(strings.NewReader(spec))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
}

func TestLoadInvalidRegex(t *testing.T) {
	_, err := load(strings.NewReader("^[invalid => Bad"))
	if err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

func TestApply(t *testing.T) {
	spec := "src/api => API\nsrc/core => Core\n"
	groups, _ := load(strings.NewReader(spec))

	commits := []model.Commit{
		{Entity: "src/api/handler.go"},
		{Entity: "src/core/engine.go"},
		{Entity: "vendor/lib/foo.go"}, // no match → excluded
	}
	result := Apply(commits, groups)

	if len(result) != 2 {
		t.Fatalf("expected 2 commits after grouping, got %d", len(result))
	}
	if result[0].Entity != "API" {
		t.Errorf("expected entity 'API', got %q", result[0].Entity)
	}
	if result[1].Entity != "Core" {
		t.Errorf("expected entity 'Core', got %q", result[1].Entity)
	}
}

func TestApplyRegexPattern(t *testing.T) {
	spec := "^src/gen/ => Generated\n"
	groups, _ := load(strings.NewReader(spec))

	commits := []model.Commit{
		{Entity: "src/gen/proto.go"},
		{Entity: "src/other/foo.go"},
	}
	result := Apply(commits, groups)

	if len(result) != 1 || result[0].Entity != "Generated" {
		t.Errorf("expected 1 commit mapped to 'Generated', got %v", result)
	}
}

func TestLoadFile(t *testing.T) {
	path := testhelpers.WriteTempFile(t, "groups.txt", "src/api => API\n")

	groups, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(groups))
	}
}

func TestApplyEmptyGroups(t *testing.T) {
	commits := []model.Commit{{Entity: "anything.go"}}
	result := Apply(commits, nil)
	if len(result) != 1 {
		t.Errorf("empty groups should return all commits unchanged, got %d", len(result))
	}
}
