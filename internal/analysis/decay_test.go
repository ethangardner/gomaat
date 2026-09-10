package analysis

import (
	"math"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestDecayWeight(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	dateAt := func(daysAgo int) string {
		return now.AddDate(0, 0, -daysAgo).Format("2006-01-02")
	}

	tests := []struct {
		name         string
		date         string
		halfLifeDays float64
		wantOK       bool
		checkWeight  func(t *testing.T, got float64)
	}{
		{
			name: "commit dated today has weight 1 regardless of half-life",
			date: dateAt(0), halfLifeDays: 30, wantOK: true,
			checkWeight: func(t *testing.T, got float64) {
				if got != 1.0 {
					t.Errorf("got %v, want 1.0", got)
				}
			},
		},
		{
			name: "age equal to half-life halves the weight",
			date: dateAt(90), halfLifeDays: 90, wantOK: true,
			checkWeight: func(t *testing.T, got float64) {
				if math.Abs(got-0.5) > 1e-9 {
					t.Errorf("got %v, want ~0.5", got)
				}
			},
		},
		{
			name: "short half-life heavily discounts an old commit",
			date: dateAt(365), halfLifeDays: 1, wantOK: true,
			checkWeight: func(t *testing.T, got float64) {
				if got > 1e-50 {
					t.Errorf("got %v, want a value extremely close to 0", got)
				}
			},
		},
		{
			name: "long half-life barely discounts an old commit",
			date: dateAt(30), halfLifeDays: 3650, wantOK: true,
			checkWeight: func(t *testing.T, got float64) {
				if got < 0.99 || got > 1.0 {
					t.Errorf("got %v, want close to but at most 1.0", got)
				}
			},
		},
		{
			name: "future-dated commit is clamped to age 0, not weight > 1",
			date: dateAt(-10), halfLifeDays: 30, wantOK: true,
			checkWeight: func(t *testing.T, got float64) {
				if got != 1.0 {
					t.Errorf("got %v, want 1.0", got)
				}
			},
		},
		{
			name: "unparseable date reports ok=false",
			date: "not-a-date", halfLifeDays: 30, wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decayWeight(tt.date, now, tt.halfLifeDays)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && tt.checkWeight != nil {
				tt.checkWeight(t, got)
			}
		})
	}
}

func TestResolveNow(t *testing.T) {
	if got := resolveNow(model.Options{}); got.IsZero() {
		t.Error("expected resolveNow to default to a non-zero time when AgeTimeNow is unset")
	}
	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := resolveNow(model.Options{AgeTimeNow: fixed}); !got.Equal(fixed) {
		t.Errorf("expected resolveNow to return the explicit AgeTimeNow, got %v", got)
	}
}

func TestFormatMetricDisabledIsPlainInt(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{0, "0"}, {1, "1"}, {2.0, "2"}, {100, "100"},
	}
	for _, tt := range tests {
		got := formatMetric(tt.v, model.Options{})
		if got != tt.want {
			t.Errorf("formatMetric(%v, disabled) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestFormatMetricEnabledUsesTwoDecimals(t *testing.T) {
	got := formatMetric(1.5, model.Options{HalfLifeDays: 90})
	if got != "1.50" {
		t.Errorf("formatMetric(1.5, enabled) = %q, want %q", got, "1.50")
	}
}

func TestRevisionsHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Entity: "foo.go", Date: now.Format("2006-01-02")},
		{Rev: "r2", Entity: "foo.go", Date: now.AddDate(0, 0, -90).Format("2006-01-02")},
	}
	opts := model.Options{HalfLifeDays: 90, AgeTimeNow: now}
	results := Revisions(commits, opts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := 1.5 // 1.0 (today) + 0.5 (90 days old, half-life=90)
	if math.Abs(results[0].Revs-want) > 1e-9 {
		t.Errorf("revs: got %v, want %v", results[0].Revs, want)
	}

	rows := FormatRevisions(results, opts)
	if rows[1][1] != "1.50" {
		t.Errorf("formatted revs: got %q, want %q", rows[1][1], "1.50")
	}
}

func TestRevisionsNoHalfLifeMatchesUndecayedInt(t *testing.T) {
	commits := []model.Commit{
		{Rev: "r1", Entity: "foo.go", Date: "2024-01-01"},
		{Rev: "r2", Entity: "foo.go", Date: "2024-01-02"},
	}
	results := Revisions(commits, model.Options{})
	if results[0].Revs != 2 {
		t.Fatalf("revs: got %v, want 2", results[0].Revs)
	}
	rows := FormatRevisions(results, model.Options{})
	if rows[1][1] != "2" {
		t.Errorf("formatted revs: got %q, want %q (byte-identical int formatting when disabled)", rows[1][1], "2")
	}
}

func TestCouplingHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	recentDate := now.Format("2006-01-02")
	oldDate := now.AddDate(0, 0, -90).Format("2006-01-02")

	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go", Date: recentDate}, {Rev: "r1", Entity: "b.go", Date: recentDate},
		{Rev: "r2", Entity: "a.go", Date: oldDate}, {Rev: "r2", Entity: "b.go", Date: oldDate},
	}
	opts := looseOpts
	opts.HalfLifeDays = 90
	opts.AgeTimeNow = now

	results := Coupling(commits, opts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// moduleRevs: a.go=b.go=1.0(recent)+0.5(old)=1.5; shared=1.5; degree=(1.5/1.5)*100=100
	if math.Abs(results[0].Degree-100) > 1e-9 {
		t.Errorf("degree: got %v, want 100", results[0].Degree)
	}
	if results[0].AvgRevs != 2 { // ceil(1.5)
		t.Errorf("avgRevs: got %v, want 2", results[0].AvgRevs)
	}

	rows := FormatCoupling(results, opts)
	if rows[1][2] != "100.00" || rows[1][3] != "2.00" {
		t.Errorf("formatted row: got degree=%q avgRevs=%q, want 100.00 / 2.00", rows[1][2], rows[1][3])
	}
}

func TestSumOfCouplingHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	recentDate := now.Format("2006-01-02")
	oldDate := now.AddDate(0, 0, -90).Format("2006-01-02")

	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go", Date: recentDate}, {Rev: "r1", Entity: "b.go", Date: recentDate},
		{Rev: "r2", Entity: "a.go", Date: oldDate}, {Rev: "r2", Entity: "b.go", Date: oldDate},
	}
	opts := looseOpts
	opts.HalfLifeDays = 90
	opts.AgeTimeNow = now

	results := SumOfCoupling(commits, opts)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if math.Abs(r.Soc-1.5) > 1e-9 {
			t.Errorf("entity %s: soc got %v, want 1.5", r.Entity, r.Soc)
		}
	}
}

func TestEntityOwnershipHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.Format("2006-01-02"), LocAdded: 10},
		{Rev: "r2", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -90).Format("2006-01-02"), LocAdded: 10},
	}
	opts := model.Options{HalfLifeDays: 90, AgeTimeNow: now}
	results := EntityOwnership(commits, opts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	want := 15.0 // 10*1.0 + 10*0.5
	if math.Abs(results[0].Added-want) > 1e-9 {
		t.Errorf("added: got %v, want %v", results[0].Added, want)
	}
}

func TestFragmentationHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	// Alice's revision is recent; Bob's is a year old. A 1-day half-life
	// decays Bob's contribution to ~0, so the entity should read as
	// effectively single-author (fractal ~0) despite having two raw authors.
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.Format("2006-01-02")},
		{Rev: "r2", Author: "Bob", Entity: "foo.go", Date: now.AddDate(0, 0, -365).Format("2006-01-02")},
	}
	opts := model.Options{HalfLifeDays: 1, AgeTimeNow: now}
	results := Fragmentation(commits, opts)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Fractal > 0.05 {
		t.Errorf("fractal: got %v, want close to 0 (old author's revisions decayed away)", results[0].Fractal)
	}
}

func TestMainDevHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -365).Format("2006-01-02"), LocAdded: 100},
		{Rev: "r2", Author: "Bob", Entity: "foo.go", Date: now.Format("2006-01-02"), LocAdded: 10},
	}

	undecayed := MainDev(commits, model.Options{})
	if undecayed[0].Contributor != "Alice" {
		t.Fatalf("undecayed: expected Alice (100 added) as main dev, got %s", undecayed[0].Contributor)
	}

	opts := model.Options{HalfLifeDays: 1, AgeTimeNow: now}
	decayed := MainDev(commits, opts)
	if decayed[0].Contributor != "Bob" {
		t.Fatalf("decayed: expected Bob once Alice's year-old commit decays away, got %s", decayed[0].Contributor)
	}
}

func TestRefactoringMainDevHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -365).Format("2006-01-02"), LocDeleted: 100},
		{Rev: "r2", Author: "Bob", Entity: "foo.go", Date: now.Format("2006-01-02"), LocDeleted: 10},
	}

	undecayed := RefactoringMainDev(commits, model.Options{})
	if undecayed[0].Contributor != "Alice" {
		t.Fatalf("undecayed: expected Alice (100 deleted) as refactoring main dev, got %s", undecayed[0].Contributor)
	}

	opts := model.Options{HalfLifeDays: 1, AgeTimeNow: now}
	decayed := RefactoringMainDev(commits, opts)
	if decayed[0].Contributor != "Bob" {
		t.Fatalf("decayed: expected Bob once Alice's year-old commit decays away, got %s", decayed[0].Contributor)
	}
}

func TestMainDevByRevsHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -365).Format("2006-01-02")},
		{Rev: "r2", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -366).Format("2006-01-02")},
		{Rev: "r3", Author: "Bob", Entity: "foo.go", Date: now.Format("2006-01-02")},
	}

	undecayed := MainDevByRevs(commits, model.Options{})
	if undecayed[0].Contributor != "Alice" {
		t.Fatalf("undecayed: expected Alice (2 revs) as main dev, got %s", undecayed[0].Contributor)
	}

	opts := model.Options{HalfLifeDays: 1, AgeTimeNow: now}
	decayed := MainDevByRevs(commits, opts)
	if decayed[0].Contributor != "Bob" {
		t.Fatalf("decayed: expected Bob once Alice's year-old revisions decay away, got %s", decayed[0].Contributor)
	}
}

// TestEntityEffortIgnoresHalfLife documents that EntityEffort deliberately
// stays undecayed (it's not in --half-life's scope; only MainDevByRevs and
// Fragmentation use the decay-aware revsPerEntityAuthorForOpts).
func TestEntityEffortIgnoresHalfLife(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	commits := []model.Commit{
		{Rev: "r1", Author: "Alice", Entity: "foo.go", Date: now.AddDate(0, 0, -365).Format("2006-01-02")},
	}
	withDecay := EntityEffort(commits, model.Options{HalfLifeDays: 1, AgeTimeNow: now})
	without := EntityEffort(commits, model.Options{})
	if withDecay[0].AuthorRevs != without[0].AuthorRevs || withDecay[0].TotalRevs != without[0].TotalRevs {
		t.Errorf("expected EntityEffort to ignore HalfLifeDays, got decayed=%v undecayed=%v", withDecay[0], without[0])
	}
}
