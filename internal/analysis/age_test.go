package analysis

import (
	"testing"
	"time"

	"gomaat/internal/model"
)

func TestMonthsBetween(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected int
	}{
		{"2024-01-01", "2024-07-01", 6},
		{"2024-01-01", "2025-01-01", 12},
		{"2024-01-01", "2024-01-31", 0}, // same month
		{"2024-06-01", "2024-01-01", 0}, // future from → clamp to 0
	}
	for _, tt := range tests {
		from, _ := time.Parse("2006-01-02", tt.from)
		to, _ := time.Parse("2006-01-02", tt.to)
		got := monthsBetween(from, to)
		if got != tt.expected {
			t.Errorf("monthsBetween(%s, %s) = %d, want %d", tt.from, tt.to, got, tt.expected)
		}
	}
}

func TestAge(t *testing.T) {
	now := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Entity: "new.go", Date: "2024-06-01"}, // 1 month
		{Entity: "old.go", Date: "2024-01-01"}, // 6 months
	}
	opts := model.Options{AgeTimeNow: now}
	rows := Age(commits, opts)

	if rows[0][0] != "entity" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// sorted youngest first
	if rows[1][0] != "new.go" || rows[1][1] != "1" {
		t.Errorf("row 1: got %v, want [new.go 1]", rows[1])
	}
	if rows[2][0] != "old.go" || rows[2][1] != "6" {
		t.Errorf("row 2: got %v, want [old.go 6]", rows[2])
	}
}

func TestAgeUsesLatestDate(t *testing.T) {
	// Entity appears in multiple commits; age should reflect the most recent one
	now := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Entity: "foo.go", Date: "2024-01-01"}, // older
		{Entity: "foo.go", Date: "2024-06-01"}, // newer → 1 month ago
	}
	opts := model.Options{AgeTimeNow: now}
	rows := Age(commits, opts)
	if rows[1][1] != "1" {
		t.Errorf("expected age 1 (most recent commit), got %q", rows[1][1])
	}
}
