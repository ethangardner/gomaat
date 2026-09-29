package testhelpers

import (
	"fmt"
	"testing"
)

// fakeTB records a Fatalf instead of stopping the test, so the helper's
// failure path can be asserted on.
type fakeTB struct {
	testing.TB
	msg string
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Fatalf(format string, args ...any) {
	f.msg = fmt.Sprintf(format, args...)
}

func identity(s string) string { return s }

func TestAssertStableOrderPassesWhenEveryRunMatches(t *testing.T) {
	calls := 0
	run := func() []string { calls++; return []string{"a", "b"} }

	tb := &fakeTB{}
	AssertStableOrder(tb, "by name", []string{"a", "b"}, run, identity)

	if tb.msg != "" {
		t.Errorf("expected no failure, got %q", tb.msg)
	}
	if calls != stableOrderRuns {
		t.Errorf("expected %d runs, got %d", stableOrderRuns, calls)
	}
}

func TestAssertStableOrderFailsWhenALaterRunDiffers(t *testing.T) {
	calls := 0
	run := func() []string {
		calls++
		if calls == 3 {
			return []string{"b", "a"}
		}
		return []string{"a", "b"}
	}

	tb := &fakeTB{}
	AssertStableOrder(tb, "by name", []string{"a", "b"}, run, identity)

	want := "run 3: expected rows sorted by name [a b], got [b a]"
	if tb.msg != want {
		t.Errorf("expected failure %q, got %q", want, tb.msg)
	}
	if calls != 3 {
		t.Errorf("expected to stop after the failing run, got %d runs", calls)
	}
}
