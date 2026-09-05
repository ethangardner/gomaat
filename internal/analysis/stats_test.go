package analysis

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestQuantileExactRanks(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5}
	cases := map[float64]float64{0.25: 2, 0.5: 3, 0.75: 4}
	for p, want := range cases {
		if got := quantile(sorted, p); !floatEquals(got, want) {
			t.Errorf("quantile(%v, %v) = %v, want %v", sorted, p, got, want)
		}
	}
}

func TestQuantileInterpolation(t *testing.T) {
	sorted := []float64{1, 2, 3, 4}
	cases := map[float64]float64{0.25: 1.75, 0.5: 2.5, 0.75: 3.25}
	for p, want := range cases {
		if got := quantile(sorted, p); !floatEquals(got, want) {
			t.Errorf("quantile(%v, %v) = %v, want %v", sorted, p, got, want)
		}
	}
}

func TestQuantileSingleValue(t *testing.T) {
	if got := quantile([]float64{7}, 0.5); got != 7 {
		t.Errorf("quantile single value = %v, want 7", got)
	}
}

func TestQuantileAtUpperBound(t *testing.T) {
	// p=1.0 puts the interpolation index exactly at the last element
	// (hi == n), which must clamp to sorted[lo] rather than index out of range.
	sorted := []float64{1, 2, 3, 4, 5}
	if got := quantile(sorted, 1.0); got != 5 {
		t.Errorf("quantile(sorted, 1.0) = %v, want 5", got)
	}
}

func TestMeanEmpty(t *testing.T) {
	if got := mean(nil); got != 0 {
		t.Errorf("mean(nil) = %v, want 0", got)
	}
}

func TestSampleStddevEdgeCases(t *testing.T) {
	if got := sampleStddev(nil, 0); got != 0 {
		t.Errorf("sampleStddev(nil) = %v, want 0", got)
	}
	if got := sampleStddev([]float64{5}, 5); got != 0 {
		t.Errorf("sampleStddev(single) = %v, want 0", got)
	}
	want := math.Sqrt(2.5)
	if got := sampleStddev([]float64{1, 2, 3, 4, 5}, 3); !floatEquals(got, want) {
		t.Errorf("sampleStddev = %v, want %v", got, want)
	}
}

func TestDescribeEmpty(t *testing.T) {
	got := describe("x", nil)
	want := StatisticsResult{Metric: "x"}
	if got != want {
		t.Errorf("describe(empty) = %+v, want %+v", got, want)
	}
}

func TestDescribeSingleValue(t *testing.T) {
	got := describe("x", []float64{7})
	if got.Count != 1 || got.Min != 7 || got.Q1 != 7 || got.Median != 7 || got.Q3 != 7 || got.Max != 7 || got.Mean != 7 || got.Stddev != 0 {
		t.Errorf("describe(single) = %+v, want all-7 with stddev 0", got)
	}
}

func TestDescribeKnownDataset(t *testing.T) {
	got := describe("x", []float64{5, 1, 3, 2, 4})
	if got.Count != 5 {
		t.Errorf("Count = %d, want 5", got.Count)
	}
	if got.Min != 1 || got.Max != 5 || got.Mean != 3 {
		t.Errorf("Min/Max/Mean = %v/%v/%v, want 1/5/3", got.Min, got.Max, got.Mean)
	}
	if got.Q1 != 2 || got.Median != 3 || got.Q3 != 4 {
		t.Errorf("Q1/Median/Q3 = %v/%v/%v, want 2/3/4", got.Q1, got.Median, got.Q3)
	}
	if !floatEquals(got.Stddev, math.Sqrt(2.5)) {
		t.Errorf("Stddev = %v, want %v", got.Stddev, math.Sqrt(2.5))
	}
}
