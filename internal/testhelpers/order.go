package testhelpers

import (
	"slices"
	"testing"
)

// stableOrderRuns is how many times AssertStableOrder calls run. Go
// randomizes map iteration per range, so an unstable sort over map-built
// rows matches want by chance at most once in a handful of runs; 20 makes a
// lucky pass vanishingly rare.
const stableOrderRuns = 20

// AssertStableOrder calls run stableOrderRuns times and fails unless the key
// of every returned row, in order, equals want on each run. It's for testing
// that a sort breaks ties deterministically when its input comes from map
// iteration, where a single run can pass by luck.
//
// rule names the expected order (e.g. "by entity then coupled") and appears
// in the failure message. key projects a row to what the order is checked
// on. It stops at the first run that differs.
func AssertStableOrder[R any, K comparable](t testing.TB, rule string, want []K, run func() []R, key func(R) K) {
	t.Helper()
	for i := range stableOrderRuns {
		rows := run()
		got := make([]K, len(rows))
		for j, r := range rows {
			got[j] = key(r)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("run %d: expected rows sorted %s %v, got %v", i+1, rule, want, got)
			return
		}
	}
}
