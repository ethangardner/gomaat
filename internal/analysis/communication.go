package analysis

import (
	"fmt"
	"math"
	"sort"

	"gomaat/internal/model"
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
	// Collect distinct authors per entity
	entityAuthors := map[string]map[string]struct{}{}
	for _, c := range commits {
		if _, ok := entityAuthors[c.Entity]; !ok {
			entityAuthors[c.Entity] = map[string]struct{}{}
		}
		entityAuthors[c.Entity][c.Author] = struct{}{}
	}

	// Generate all 2-selections (with replacement) and count frequencies.
	// key: "author\x00peer" — self pairs use the same author twice.
	freqs := map[string]int{}
	for _, authors := range entityAuthors {
		authorList := make([]string, 0, len(authors))
		for a := range authors {
			authorList = append(authorList, a)
		}
		// all ordered pairs including self-pairs
		for _, a := range authorList {
			for _, b := range authorList {
				freqs[a+"\x00"+b]++
			}
		}
	}

	// Build results from non-self pairs.
	var results []CommunicationResult
	for key, shared := range freqs {
		parts := splitPairKey(key)
		me, peer := parts[0], parts[1]
		if me == peer {
			continue
		}
		myTotal := freqs[me+"\x00"+me]
		peerTotal := freqs[peer+"\x00"+peer]
		avg := int(math.Ceil((float64(myTotal) + float64(peerTotal)) / 2.0))
		if avg == 0 {
			continue
		}
		strength := int((float64(shared) / float64(avg)) * 100.0)
		results = append(results, CommunicationResult{me, peer, shared, avg, strength})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Strength != results[j].Strength {
			return results[i].Strength > results[j].Strength
		}
		return results[i].Author > results[j].Author // desc, matching original
	})

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
