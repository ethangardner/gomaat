package model

import "testing"

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	if opts.MinRevs != 5 {
		t.Errorf("MinRevs: got %d, want 5", opts.MinRevs)
	}
	if opts.MinSharedRevs != 5 {
		t.Errorf("MinSharedRevs: got %d, want 5", opts.MinSharedRevs)
	}
	if opts.MinCoupling != 30 {
		t.Errorf("MinCoupling: got %f, want 30", opts.MinCoupling)
	}
	if opts.MaxCoupling != 100 {
		t.Errorf("MaxCoupling: got %f, want 100", opts.MaxCoupling)
	}
	if opts.MaxChangesetSize != 30 {
		t.Errorf("MaxChangesetSize: got %d, want 30", opts.MaxChangesetSize)
	}
	if opts.AgeTimeNow.IsZero() {
		t.Error("AgeTimeNow should be set to current time, got zero value")
	}
}
