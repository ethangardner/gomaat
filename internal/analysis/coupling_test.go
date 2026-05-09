package analysis

import (
	"testing"

	"gomaat/internal/model"
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
	rows := Coupling(couplingCommits, looseOpts)

	if rows[0][0] != "entity" {
		t.Fatalf("expected header row, got %v", rows[0])
	}
	if len(rows) != 3 { // header + 2 pairs
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	// {a.go,b.go}: avg=2.5 → degree=80, avgRevs=3 (ceil)
	r1 := rows[1]
	if r1[0] != "a.go" || r1[1] != "b.go" || r1[2] != "80" || r1[3] != "3" {
		t.Errorf("row 1: got %v, want [a.go b.go 80 3]", r1)
	}
	// {a.go,c.go}: avg=2 → degree=50, avgRevs=2
	r2 := rows[2]
	if r2[0] != "a.go" || r2[1] != "c.go" || r2[2] != "50" || r2[3] != "2" {
		t.Errorf("row 2: got %v, want [a.go c.go 50 2]", r2)
	}
}

func TestCouplingVerbose(t *testing.T) {
	opts := looseOpts
	opts.VerboseResults = true
	rows := Coupling(couplingCommits, opts)

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
	rows := Coupling(couplingCommits, opts)

	// Only {a.go,b.go} has 2 shared revisions; {a.go,c.go} has 1 → filtered
	if len(rows) != 2 { // header + 1 pair
		t.Errorf("expected 2 rows with MinSharedRevs=2, got %d", len(rows))
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
	rows := Coupling(commits, opts)

	if len(rows) != 1 { // header only
		t.Errorf("expected no pairs when changeset exceeds max size, got %d rows", len(rows))
	}
}

func TestCouplingEmpty(t *testing.T) {
	rows := Coupling(nil, looseOpts)
	if len(rows) != 1 {
		t.Errorf("expected header-only result for empty input, got %d rows", len(rows))
	}
}

func TestSumOfCoupling(t *testing.T) {
	// r1: a.go+b.go, r2: a.go+b.go, r3: a.go+c.go
	// soc: a.go=3, b.go=2, c.go=1
	rows := SumOfCoupling(couplingCommits, looseOpts)

	if rows[0][0] != "entity" {
		t.Fatalf("expected header, got %v", rows[0])
	}
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[1][0] != "a.go" || rows[1][1] != "3" {
		t.Errorf("row 1: got %v, want [a.go 3]", rows[1])
	}
	if rows[2][0] != "b.go" || rows[2][1] != "2" {
		t.Errorf("row 2: got %v, want [b.go 2]", rows[2])
	}
	if rows[3][0] != "c.go" || rows[3][1] != "1" {
		t.Errorf("row 3: got %v, want [c.go 1]", rows[3])
	}
}
