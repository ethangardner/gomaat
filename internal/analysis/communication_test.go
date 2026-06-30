package analysis

import (
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

func TestCommunication(t *testing.T) {
	// foo.go touched by Alice+Bob → shared entity
	// bar.go touched by Alice only
	// baz.go touched by Bob only
	//
	// freqs: AA=2, BB=2, AB=1, BA=1
	// Each non-self pair: shared=1, avg=ceil((2+2)/2)=2, strength=50
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "foo.go", Author: "Bob"},
		{Entity: "bar.go", Author: "Alice"},
		{Entity: "baz.go", Author: "Bob"},
	}

	results := Communication(commits, model.Options{})
	if len(results) != 2 { // Alice→Bob and Bob→Alice
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Both pairs should have strength=50; sorted desc by author name: Bob first
	if results[0].Author != "Bob" || results[0].Peer != "Alice" || results[0].Shared != 1 || results[0].Average != 2 || results[0].Strength != 50 {
		t.Errorf("result 0: got %v, want {Bob Alice 1 2 50}", results[0])
	}
	if results[1].Author != "Alice" || results[1].Peer != "Bob" {
		t.Errorf("result 1: got %v, want {Alice Bob ...}", results[1])
	}

	assertFormattedRows(t, FormatCommunication(results, model.Options{}), "author", 3)
}

func TestCommunicationSingleAuthor(t *testing.T) {
	// Only one author touching any entities → no non-self pairs
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "bar.go", Author: "Alice"},
	}
	results := Communication(commits, model.Options{})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}

	assertFormattedRows(t, FormatCommunication(results, model.Options{}), "author", 1)
}

func TestGroupAuthorsByEntity(t *testing.T) {
	commits := []model.Commit{
		{Entity: "a.go", Author: "Alice"},
		{Entity: "a.go", Author: "Bob"},
		{Entity: "b.go", Author: "Alice"},
	}
	res := groupAuthorsByEntity(commits)
	if len(res) != 2 {
		t.Errorf("expected 2 entities, got %d", len(res))
	}
	if len(res["a.go"]) != 2 {
		t.Errorf("expected 2 authors for a.go, got %d", len(res["a.go"]))
	}
	if _, ok := res["a.go"]["Alice"]; !ok {
		t.Error("expected Alice for a.go")
	}
	if _, ok := res["a.go"]["Bob"]; !ok {
		t.Error("expected Bob for a.go")
	}
	if len(res["b.go"]) != 1 {
		t.Errorf("expected 1 author for b.go, got %d", len(res["b.go"]))
	}
}

func TestCountPairFrequencies(t *testing.T) {
	authorsByEntity := map[string]map[string]struct{}{
		"a.go": {"Alice": {}, "Bob": {}},
		"b.go": {"Alice": {}},
	}
	freqs := countPairFrequencies(authorsByEntity)

	// Alice: a.go, b.go -> 2
	// Bob: a.go -> 1
	// Shared: a.go -> 1
	if freqs[pairKey{"Alice", "Alice"}] != 2 {
		t.Errorf("expected Alice total 2, got %d", freqs[pairKey{"Alice", "Alice"}])
	}
	if freqs[pairKey{"Bob", "Bob"}] != 1 {
		t.Errorf("expected Bob total 1, got %d", freqs[pairKey{"Bob", "Bob"}])
	}
	if freqs[pairKey{"Alice", "Bob"}] != 1 {
		t.Errorf("expected Alice-Bob shared 1, got %d", freqs[pairKey{"Alice", "Bob"}])
	}
	if freqs[pairKey{"Bob", "Alice"}] != 1 {
		t.Errorf("expected Bob-Alice shared 1, got %d", freqs[pairKey{"Bob", "Alice"}])
	}
}

func TestComputeResults(t *testing.T) {
	freqs := map[pairKey]int{
		{"Alice", "Alice"}: 4,
		{"Bob", "Bob"}:     2,
		{"Alice", "Bob"}:   1,
		{"Bob", "Alice"}:   1,
	}
	// avg = ceil((4+2)/2) = 3
	// strength = (1/3)*100 = 33
	results := computeResults(freqs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	found := false
	for _, r := range results {
		if r.Author == "Alice" && r.Peer == "Bob" {
			found = true
			if r.Shared != 1 || r.Average != 3 || r.Strength != 33 {
				t.Errorf("unexpected Alice-Bob result: %+v", r)
			}
		}
	}
	if !found {
		t.Error("Alice-Bob result not found")
	}
}

func TestCommunicationSortByStrength(t *testing.T) {
	// Alice touches foo, bar, baz, qux.  Bob touches foo, bar.  Charlie touches qux.
	// Alice-Bob shared=2, avg=ceil((4+2)/2)=3, strength=66
	// Alice-Charlie shared=1, avg=ceil((4+1)/2)=3, strength=33
	// Sort must place strength=66 pairs before strength=33 pairs (exercises return c branch).
	commits := []model.Commit{
		{Entity: "foo.go", Author: "Alice"},
		{Entity: "foo.go", Author: "Bob"},
		{Entity: "bar.go", Author: "Alice"},
		{Entity: "bar.go", Author: "Bob"},
		{Entity: "baz.go", Author: "Alice"},
		{Entity: "qux.go", Author: "Alice"},
		{Entity: "qux.go", Author: "Charlie"},
	}
	results := Communication(commits, model.Options{})
	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}
	if results[0].Strength != 66 {
		t.Errorf("expected first result strength=66, got %d", results[0].Strength)
	}
	if results[len(results)-1].Strength != 33 {
		t.Errorf("expected last result strength=33, got %d", results[len(results)-1].Strength)
	}
}

func TestComputeResultsEdgeCases(t *testing.T) {
	// Single author
	freqs := map[pairKey]int{
		{"Alice", "Alice"}: 2,
	}
	res := computeResults(freqs)
	if len(res) != 0 {
		t.Errorf("expected 0 results for single author, got %d", len(res))
	}

	// Disconnected authors
	freqs = map[pairKey]int{
		{"Alice", "Alice"}: 2,
		{"Bob", "Bob"}:     2,
	}
	res = computeResults(freqs)
	if len(res) != 0 {
		t.Errorf("expected 0 results for disconnected authors, got %d", len(res))
	}
}
