package analysis

import (
	"regexp"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestDefects(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "foo.go", Message: "fix: null pointer crash"},
		{Rev: "r2", Entity: "foo.go", Message: "add new feature"},
		{Rev: "r3", Entity: "foo.go", Message: "fix bug in feature"},
		{Rev: "r4", Entity: "bar.go", Message: "add bar"},
		{Rev: "r5", Entity: "bar.go", Message: "refactor bar"},
	}

	re := regexp.MustCompile(`(?i)fix|bug`)
	isBugfix := func(c model.Commit) bool { return re.MatchString(c.Message) }

	results := Defects(commits, isBugfix, model.Options{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// sorted by defect-ratio desc: foo.go (2/3 ~ 0.67) > bar.go (0/2 = 0)
	if results[0].Entity != "foo.go" || results[0].BugfixRevisions != 2 || results[0].TotalRevisions != 3 {
		t.Errorf("result 0: got %+v, want {foo.go 2 3 ...}", results[0])
	}
	if got, want := results[0].DefectRatio, 2.0/3.0; got != want {
		t.Errorf("result 0 defect ratio: got %v, want %v", got, want)
	}

	if results[1].Entity != "bar.go" || results[1].BugfixRevisions != 0 || results[1].TotalRevisions != 2 {
		t.Errorf("result 1: got %+v, want {bar.go 0 2 ...}", results[1])
	}
	if results[1].DefectRatio != 0 {
		t.Errorf("result 1 defect ratio: got %v, want 0", results[1].DefectRatio)
	}
}

func TestDefectsNoMatches(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "foo.go", Message: "add feature"},
	}
	isBugfix := func(model.Commit) bool { return false }

	results := Defects(commits, isBugfix, model.Options{})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].BugfixRevisions != 0 || results[0].TotalRevisions != 1 || results[0].DefectRatio != 0 {
		t.Errorf("got %+v, want {foo.go 0 1 0}", results[0])
	}
}

func TestDefectsEmpty(t *testing.T) {
	results := Defects(nil, func(model.Commit) bool { return true }, model.Options{})
	assertEmptyResults(t, results)
}

func TestFormatDefects(t *testing.T) {
	results := []DefectsResult{
		{Entity: "foo.go", BugfixRevisions: 2, TotalRevisions: 3, DefectRatio: 2.0 / 3.0},
	}
	rows := FormatDefects(results, model.Options{})
	assertFormattedRows(t, rows, "entity", 2)
	r := rows[1]
	if r[0] != "foo.go" || r[1] != "2" || r[2] != "3" || r[3] != "0.67" {
		t.Errorf("row 1: got %v, want [foo.go 2 3 0.67]", r)
	}
}
