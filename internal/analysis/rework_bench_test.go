package analysis

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/ethangardner/gomaat/internal/gitdiff"
	"github.com/ethangardner/gomaat/internal/model"
)

func BenchmarkRework(b *testing.B) {
	commits := makeBenchPatches(10_000, 10)
	opts := model.Options{ReworkWindow: 14 * 24 * time.Hour, ReworkTimeNow: day(20_000)}
	for b.Loop() {
		if _, err := Rework(commitsOf(commits...), opts); err != nil {
			b.Fatal(err)
		}
	}
}

// makeBenchPatches builds a valid single-line history of numRevs commits, one
// per day, each replacing up to 3 lines of one of numFiles files with up to 5
// new ones at a random position. Files grow to roughly 1k lines, and roughly
// a third of added lines are near-duplicates of deleted ones, so the move and
// edit matching paths are exercised too.
func makeBenchPatches(numRevs, numFiles int) []gitdiff.Commit {
	r := rand.New(rand.NewPCG(1, 2))
	lengths := make([]int, numFiles)
	commits := make([]gitdiff.Commit, 0, numRevs)
	for rev := range numRevs {
		f := r.IntN(numFiles)
		pos := r.IntN(lengths[f] + 1)
		h := gitdiff.Hunk{OldStart: pos}
		for range min(r.IntN(4), lengths[f]-pos) {
			h.Deleted = append(h.Deleted, fmt.Sprintf("value%d := compute(input, %d)", r.IntN(50), r.IntN(50)))
		}
		if len(h.Deleted) > 0 {
			h.OldStart = pos + 1
		}
		for range r.IntN(6) {
			h.Added = append(h.Added, fmt.Sprintf("value%d := compute(input, %d)", r.IntN(50), r.IntN(50)))
		}
		lengths[f] += len(h.Added) - len(h.Deleted)
		commits = append(commits, commitOn(rev, fileDiff(fmt.Sprintf("pkg/file%d.go", f), h)))
	}
	return commits
}
