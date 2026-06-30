package analysis

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/ethangardner/gomaat/internal/model"
)

type CommunicationResult struct {
	Author   string
	Peer     string
	Shared   int
	Average  int
	Strength int
}

// Communication maps team collaboration needs based on shared entities.
//
// Algorithm (mirrors the original Clojure implementation):
//  1. Find distinct authors per entity.
//  2. For each entity, generate all 2-selections (with replacement) of its authors:
//     [A,A], [A,B], [B,A], [B,B] etc.
//  3. Count frequencies across all entities.
//  4. Self-pairs [A,A] give A's total entity count ("total commits").
//  5. Non-self pairs [A,B] give the count of entities both A and B touched ("shared").
//  6. average = ceil((self_A + self_B) / 2)
//  7. strength = int((shared / average) * 100)
func Communication(commits []model.Commit, _ model.Options) []CommunicationResult {
	entityAuthors := groupAuthorsByEntity(commits)
	freqs := countPairFrequencies(entityAuthors)
	results := computeResults(freqs)

	slices.SortFunc(results, func(a, b CommunicationResult) int {
		if c := cmp.Compare(b.Strength, a.Strength); c != 0 {
			return c
		}
		return cmp.Compare(b.Author, a.Author)
	})

	return results
}

func groupAuthorsByEntity(commits []model.Commit) map[string]map[string]struct{} {
	entityAuthors := map[string]map[string]struct{}{}
	for _, c := range commits {
		if _, ok := entityAuthors[c.Entity]; !ok {
			entityAuthors[c.Entity] = map[string]struct{}{}
		}
		entityAuthors[c.Entity][c.Author] = struct{}{}
	}
	return entityAuthors
}

func countPairFrequencies(authorsByEntity map[string]map[string]struct{}) map[pairKey]int {
	freqs := map[pairKey]int{}
	for _, authors := range authorsByEntity {
		// all ordered pairs including self-pairs
		for a := range authors {
			for b := range authors {
				freqs[pairKey{a, b}]++
			}
		}
	}
	return freqs
}

func computeResults(freqs map[pairKey]int) []CommunicationResult {
	var results []CommunicationResult
	for key, shared := range freqs {
		me, peer := key.a, key.b
		if me == peer {
			continue
		}
		myTotal := freqs[pairKey{me, me}]
		peerTotal := freqs[pairKey{peer, peer}]
		avg := int(math.Ceil((float64(myTotal) + float64(peerTotal)) / 2.0))
		if avg == 0 {
			continue
		}
		strength := int((float64(shared) / float64(avg)) * 100.0)
		results = append(results, CommunicationResult{me, peer, shared, avg, strength})
	}
	return results
}

func FormatCommunication(results []CommunicationResult, _ model.Options) [][]string {
	out := [][]string{{"author", "peer", "shared", "average", "strength"}}
	for _, r := range results {
		out = append(out, []string{
			r.Author, r.Peer,
			fmt.Sprint(r.Shared),
			fmt.Sprint(r.Average),
			fmt.Sprint(r.Strength),
		})
	}
	return out
}
