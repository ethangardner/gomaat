package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

var riskCoupling = []CouplingResult{
	{Entity: "a.go", Coupled: "b.go", Degree: 80},
	{Entity: "c.go", Coupled: "d.go", Degree: 60},
}

func TestRiskFlagsMissingCoupledFile(t *testing.T) {
	results := Risk(riskCoupling, []string{"a.go"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ChangedEntity != "a.go" || results[0].MissingEntity != "b.go" || results[0].Coupling != 80 {
		t.Errorf("got %+v, want {a.go b.go 80}", results[0])
	}
}

func TestRiskChecksBothDirections(t *testing.T) {
	// Changing the "Coupled" side of a pair should flag the "Entity" side as missing.
	results := Risk(riskCoupling, []string{"b.go"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ChangedEntity != "b.go" || results[0].MissingEntity != "a.go" {
		t.Errorf("got %+v, want {b.go a.go}", results[0])
	}
}

func TestRiskNotFlaggedWhenBothSidesChanged(t *testing.T) {
	results := Risk(riskCoupling, []string{"a.go", "b.go"})
	for _, r := range results {
		if r.ChangedEntity == "a.go" || r.MissingEntity == "a.go" || r.ChangedEntity == "b.go" || r.MissingEntity == "b.go" {
			t.Errorf("a.go/b.go pair should not be flagged when both changed, got %+v", r)
		}
	}
}

func TestRiskNoCouplingData(t *testing.T) {
	results := Risk(nil, []string{"a.go"})
	assertEmptyResults(t, results)
}

func TestRiskChangedFileNotInCouplingTable(t *testing.T) {
	results := Risk(riskCoupling, []string{"z.go"})
	assertEmptyResults(t, results)
}

func TestRiskSortedByCouplingDesc(t *testing.T) {
	results := Risk(riskCoupling, []string{"a.go", "c.go"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Coupling < results[1].Coupling {
		t.Errorf("expected descending coupling order, got %d then %d", results[0].Coupling, results[1].Coupling)
	}
	if results[0].ChangedEntity != "a.go" {
		t.Errorf("expected higher-degree pair (a.go/b.go) first, got %+v", results[0])
	}
}

func TestFormatRisk(t *testing.T) {
	results := Risk(riskCoupling, []string{"a.go"})
	assertFormattedRows(t, FormatRisk(results, model.Options{}), "changed-entity", 2)
}

func TestFormatRiskEmpty(t *testing.T) {
	assertFormattedRows(t, FormatRisk(nil, model.Options{}), "changed-entity", 1)
}
