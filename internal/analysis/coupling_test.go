package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

// looseOpts disables all thresholds so every pair is included.
var looseOpts = model.Options{
	MinRevs:          1,
	MinSharedRevs:    1,
	MinCoupling:      0,
	MaxCoupling:      100,
	MaxChangesetSize: 30,
}

// couplingCommits: a.go and b.go co-change twice; a.go and c.go co-change once.
//
//	moduleRevs: a.go=3, b.go=2, c.go=1
//	pairShared: {a.go,b.go}=2, {a.go,c.go}=1
var couplingCommits = []model.Commit{
	{Rev: "r1", Entity: "a.go"}, {Rev: "r1", Entity: "b.go"},
	{Rev: "r2", Entity: "a.go"}, {Rev: "r2", Entity: "b.go"},
	{Rev: "r3", Entity: "a.go"}, {Rev: "r3", Entity: "c.go"},
}

func TestCouplingBasic(t *testing.T) {
	results := Coupling(couplingCommits, looseOpts)

	if len(results) != 2 { // 2 pairs
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// {a.go,b.go}: avg=2.5 → degree=80, avgRevs=3 (ceil)
	if results[0].Entity != "a.go" || results[0].Coupled != "b.go" || results[0].Degree != 80 || results[0].AvgRevs != 3 {
		t.Errorf("result 0: got %v, want {a.go b.go 80 3}", results[0])
	}
	// {a.go,c.go}: avg=2 → degree=50, avgRevs=2
	if results[1].Entity != "a.go" || results[1].Coupled != "c.go" || results[1].Degree != 50 || results[1].AvgRevs != 2 {
		t.Errorf("result 1: got %v, want {a.go c.go 50 2}", results[1])
	}

	assertFormattedRows(t, FormatCoupling(results, looseOpts), "entity", 3)
}

func TestCouplingVerbose(t *testing.T) {
	opts := looseOpts
	opts.VerboseResults = true
	results := Coupling(couplingCommits, opts)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Verify formatter
	rows := FormatCoupling(results, opts)
	if len(rows[0]) != 7 {
		t.Fatalf("verbose header should have 7 columns, got %d", len(rows[0]))
	}
	// {a.go,b.go}: revA=3, revB=2, shared=2
	r := rows[1]
	if r[4] != "3" || r[5] != "2" || r[6] != "2" {
		t.Errorf("verbose row: got revA=%s revB=%s shared=%s, want 3 2 2", r[4], r[5], r[6])
	}
}

func TestCouplingMinSharedRevs(t *testing.T) {
	opts := looseOpts
	opts.MinSharedRevs = 2
	results := Coupling(couplingCommits, opts)

	// Only {a.go,b.go} has 2 shared revisions; {a.go,c.go} has 1 → filtered
	if len(results) != 1 {
		t.Errorf("expected 1 result with MinSharedRevs=2, got %d", len(results))
	}
}

func TestCouplingMaxChangesetSize(t *testing.T) {
	// All 3 files change in the same commit; with MaxChangesetSize=2 the commit is skipped.
	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go"},
		{Rev: "r1", Entity: "b.go"},
		{Rev: "r1", Entity: "c.go"},
	}
	opts := looseOpts
	opts.MaxChangesetSize = 2
	results := Coupling(commits, opts)

	if len(results) != 0 {
		t.Errorf("expected no results when changeset exceeds max size, got %d", len(results))
	}
}

func TestCouplingEmpty(t *testing.T) {
	results := Coupling(nil, looseOpts)
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty input, got %d", len(results))
	}
}

func TestCouplingTiebreakerByAvgRevs(t *testing.T) {
	// a.go+b.go co-change twice, c.go+d.go co-change four times.
	// Both pairs reach degree=100; avgRevs differs (2 vs 4).
	// Sort must place the higher-avgRevs pair first (exercises AvgRevs tie-breaker branch).
	commits := []model.Commit{
		{Rev: "r1", Entity: "a.go"}, {Rev: "r1", Entity: "b.go"},
		{Rev: "r2", Entity: "a.go"}, {Rev: "r2", Entity: "b.go"},
		{Rev: "r3", Entity: "c.go"}, {Rev: "r3", Entity: "d.go"},
		{Rev: "r4", Entity: "c.go"}, {Rev: "r4", Entity: "d.go"},
		{Rev: "r5", Entity: "c.go"}, {Rev: "r5", Entity: "d.go"},
		{Rev: "r6", Entity: "c.go"}, {Rev: "r6", Entity: "d.go"},
	}
	results := Coupling(commits, looseOpts)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Degree != 100 || results[1].Degree != 100 {
		t.Errorf("expected both degrees=100, got %d and %d", results[0].Degree, results[1].Degree)
	}
	if results[0].AvgRevs <= results[1].AvgRevs {
		t.Errorf("expected higher avgRevs first, got %d then %d", results[0].AvgRevs, results[1].AvgRevs)
	}
}

func TestSumOfCouplingTiebreaker(t *testing.T) {
	// a.go and b.go both appear in the same two commits → equal SOC.
	// Tie-breaker must sort alphabetically by entity name (exercises entity tie-breaker branch).
	commits := []model.Commit{
		{Rev: "r1", Entity: "b.go"}, {Rev: "r1", Entity: "a.go"},
		{Rev: "r2", Entity: "b.go"}, {Rev: "r2", Entity: "a.go"},
	}
	results := SumOfCoupling(commits, looseOpts)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Soc != results[1].Soc {
		t.Fatalf("expected equal SOC values for tie-breaker test, got %d and %d", results[0].Soc, results[1].Soc)
	}
	if results[0].Entity != "a.go" {
		t.Errorf("expected a.go first (alphabetical), got %q", results[0].Entity)
	}
}

func TestSumOfCoupling(t *testing.T) {
	// r1: a.go+b.go, r2: a.go+b.go, r3: a.go+c.go
	// soc: a.go=3, b.go=2, c.go=1
	results := SumOfCoupling(couplingCommits, looseOpts)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Entity != "a.go" || results[0].Soc != 3 {
		t.Errorf("result 0: got %v, want {a.go 3}", results[0])
	}
	if results[1].Entity != "b.go" || results[1].Soc != 2 {
		t.Errorf("result 1: got %v, want {b.go 2}", results[1])
	}
	if results[2].Entity != "c.go" || results[2].Soc != 1 {
		t.Errorf("result 2: got %v, want {c.go 1}", results[2])
	}

	assertFormattedRows(t, FormatSumOfCoupling(results, looseOpts), "entity", 4)
}
