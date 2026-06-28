package analysis

import (
	"fmt"
	"testing"

	"gomaat/internal/model"
)

// b.Loop() (Go 1.24+) is preferred over the classic `for i := 0; i < b.N; i++` form:
// it excludes setup code from timing automatically and prevents the compiler from
// optimizing away the loop body when b.N is 0 on a dry run.

func BenchmarkCoupling(b *testing.B) {
	commits := makeBenchCommits(10_000, 20)
	opts := model.Options{MaxChangesetSize: 100, MinRevs: 1, MinSharedRevs: 1, MaxCoupling: 100}
	for b.Loop() {
		Coupling(commits, opts)
	}
}

func BenchmarkDedupe(b *testing.B) {
	// 20 entities per revision, ~25% are duplicates
	entities := make([]string, 0, 20)
	for i := range 15 {
		entities = append(entities, fmt.Sprintf("pkg/module%d/file.go", i))
	}
	entities = append(entities, entities[:5]...)
	for b.Loop() {
		cp := append([]string(nil), entities...)
		dedupe(cp)
	}
}

func makeBenchCommits(numRevs, filesPerRev int) []model.Commit {
	commits := make([]model.Commit, 0, numRevs*filesPerRev)
	for r := range numRevs {
		rev := fmt.Sprintf("rev%06d", r)
		for f := range filesPerRev {
			commits = append(commits, model.Commit{
				Rev:    rev,
				Entity: fmt.Sprintf("pkg/module%d/file.go", f%(filesPerRev/2)),
			})
		}
	}
	return commits
}
