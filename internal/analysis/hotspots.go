package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type HotspotResult struct {
	Entity    string
	Revisions int
	Lines     int
	Score     float64
	Fractal   float64
}

// Hotspots joins revision counts and fragmentation (both derived from
// commits) with current lines-of-code per file (locByFile, keyed by
// repo-relative path) to rank entities by churn x size. Entities missing
// from either side of the join — files deleted from disk but still present
// in history, or files on disk with no revisions in this log — are
// excluded rather than zero-filled, since either dimension being fabricated
// would produce a misleading score.
func Hotspots(commits []model.Commit, locByFile map[string]int, opts model.Options) []HotspotResult {
	revisions := Revisions(commits, opts)
	fragmentation := Fragmentation(commits, opts)

	fractalByEntity := make(map[string]float64, len(fragmentation))
	for _, f := range fragmentation {
		fractalByEntity[f.Entity] = f.Fractal
	}

	type joinedRow struct {
		entity    string
		revisions int
		lines     int
		fractal   float64
	}
	var joined []joinedRow
	maxRevisions, maxLines := 0, 0
	for _, r := range revisions {
		lines, ok := locByFile[r.Entity]
		if !ok {
			continue
		}
		joined = append(joined, joinedRow{r.Entity, r.Revs, lines, fractalByEntity[r.Entity]})
		if r.Revs > maxRevisions {
			maxRevisions = r.Revs
		}
		if lines > maxLines {
			maxLines = lines
		}
	}

	results := make([]HotspotResult, 0, len(joined))
	for _, r := range joined {
		var score float64
		if maxRevisions > 0 && maxLines > 0 {
			raw := (float64(r.revisions) / float64(maxRevisions)) * (float64(r.lines) / float64(maxLines)) * 100
			score = math.Round(raw*100) / 100
		}
		results = append(results, HotspotResult{
			Entity:    r.entity,
			Revisions: r.revisions,
			Lines:     r.lines,
			Score:     score,
			Fractal:   r.fractal,
		})
	}

	slices.SortFunc(results, func(a, b HotspotResult) int {
		if c := cmp.Compare(b.Score, a.Score); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})

	return results
}

func FormatHotspots(results []HotspotResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "revisions", "lines", "hotspot-score", "fractal-value"}}
	for _, r := range results {
		out = append(out, []string{
			r.Entity,
			fmt.Sprint(r.Revisions),
			fmt.Sprint(r.Lines),
			fmt.Sprintf("%.2f", r.Score),
			fmt.Sprintf("%.2f", r.Fractal),
		})
	}
	return out
}
