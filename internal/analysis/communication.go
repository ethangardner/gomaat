package analysis

import (
	"fmt"
	"math"
	"sort"

	"godemaat/internal/model"
)

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
func Communication(commits []model.Commit, _ model.Options) [][]string {
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
	type commRow struct {
		author   string
		peer     string
		shared   int
		average  int
		strength int
	}
	var rows []commRow
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
		rows = append(rows, commRow{me, peer, shared, avg, strength})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].strength != rows[j].strength {
			return rows[i].strength > rows[j].strength
		}
		return rows[i].author > rows[j].author // desc, matching original
	})

	out := [][]string{{"author", "peer", "shared", "average", "strength"}}
	for _, r := range rows {
		out = append(out, []string{
			r.author, r.peer,
			fmt.Sprint(r.shared),
			fmt.Sprint(r.average),
			fmt.Sprint(r.strength),
		})
	}
	return out
}
