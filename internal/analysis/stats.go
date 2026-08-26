package analysis

import (
	"math"
	"slices"
)

// quantile computes the p-th quantile (0<=p<=1) of sorted using linear
// interpolation between closest ranks — R's/numpy's default "type 7"
// method, chosen because it's the de facto standard for descriptive stats
// and gomaat has no prior convention to match. sorted must already be
// sorted ascending and non-empty.
func quantile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 1 {
		return sorted[0]
	}
	h := p * float64(n-1)
	lo := int(math.Floor(h))
	hi := lo + 1
	if hi >= n {
		return sorted[lo]
	}
	frac := h - float64(lo)
	return sorted[lo] + frac*(sorted[hi]-sorted[lo])
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// sampleStddev computes the sample standard deviation (n-1, Bessel's
// correction). Returns 0 for n<2: a single observation (or none) has no
// defined sample variance, and 0 (rather than NaN) keeps the CSV column
// numeric and keeps callers from needing a special case.
func sampleStddev(values []float64, m float64) float64 {
	if len(values) < 2 {
		return 0
	}
	var sumSq float64
	for _, v := range values {
		d := v - m
		sumSq += d * d
	}
	return math.Sqrt(sumSq / float64(len(values)-1))
}

// describe computes count/min/q1/median/q3/max/mean/stddev for one named
// metric. values need not be pre-sorted; describe clones before sorting so
// the caller's slice is left untouched. For an empty values slice, describe
// returns a result with Count 0 and every other field at its zero value
// (0.0) — rather than NaN — so FormatStatistics never has to special-case
// an empty metric.
func describe(metric string, values []float64) StatisticsResult {
	if len(values) == 0 {
		return StatisticsResult{Metric: metric}
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)

	m := mean(sorted)
	n := len(sorted)
	return StatisticsResult{
		Metric: metric,
		Count:  n,
		Min:    sorted[0],
		Q1:     quantile(sorted, 0.25),
		Median: quantile(sorted, 0.5),
		Q3:     quantile(sorted, 0.75),
		Max:    sorted[n-1],
		Mean:   m,
		Stddev: sampleStddev(sorted, m),
	}
}
