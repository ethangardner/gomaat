package analysis

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/ethangardner/gomaat/internal/gitdiff"
	"github.com/ethangardner/gomaat/internal/model"
)

// minEditSimilarity is the token similarity (Dice coefficient, 0..1) at or
// above which a line that replaces another in the same hunk is treated as an
// edit of that line — it keeps the original line's provenance — rather than
// as the original being removed and new code written in its place.
const minEditSimilarity = 0.6

type ReworkResult struct {
	Entity   string
	Added    int     // judged lines added to the entity
	Reworked int     // of those, lines removed or rewritten within the window
	Ratio    float64 // Reworked / Added, as a percentage
}

// Rework reports, per entity, the share of added lines that were removed or
// substantially rewritten within opts.ReworkWindow of landing.
//
// commits must be in chronological (oldest-first) order along a single line
// of history, as produced by
//
//	git log --reverse --first-parent --diff-merges=first-parent -p -U0 --no-renames
//
// so that applying each commit's hunks in turn reproduces every file's
// contents. Rework tracks, for every line, the commit (entity and time) that
// introduced it. A line counts as:
//
//   - added, if it is non-blank and its window has fully elapsed by
//     opts.ReworkTimeNow (younger lines cannot be judged yet, so they are
//     left out of both counts). A merge's first-parent diff counts as landing
//     the merged branch's lines at the merge's time, so merge-commit, squash
//     and rebase workflows are measured alike;
//   - reworked, if a later commit within the window removes it without that
//     commit re-adding the same text elsewhere (a move, compared ignoring
//     whitespace, even across files) and without replacing it in place with
//     a line at least minEditSimilarity token-similar (a small edit).
//
// A moved or lightly edited line keeps its original provenance, so it is
// still judged against its original landing time. Lines that predate the
// analyzed history are untracked and never counted.
func Rework(commits iter.Seq2[gitdiff.Commit, error], opts model.Options) ([]ReworkResult, error) {
	now := opts.ReworkTimeNow
	if now.IsZero() {
		now = time.Now()
	}
	t := reworkTracker{
		window:   int64(opts.ReworkWindow / time.Second),
		now:      now.Unix(),
		files:    map[string][]lineOrigin{},
		added:    map[string]int{},
		reworked: map[string]int{},
	}
	for c, err := range commits {
		if err != nil {
			return nil, err
		}
		t.apply(c)
	}
	return t.results(), nil
}

func FormatRework(results []ReworkResult, _ model.Options) [][]string {
	out := [][]string{{"entity", "added-lines", "reworked-lines", "rework-ratio"}}
	for _, r := range results {
		out = append(out, []string{r.Entity, fmt.Sprint(r.Added), fmt.Sprint(r.Reworked), fmt.Sprintf("%.2f", r.Ratio)})
	}
	return out
}

// lineOrigin records which entity and when (unix seconds) a counted line was
// added. The zero value marks a line that isn't counted (blank, pre-history,
// or too young to judge).
type lineOrigin struct {
	entity string
	when   int64
}

type reworkTracker struct {
	window, now int64
	files       map[string][]lineOrigin // current provenance of every line, per path
	added       map[string]int
	reworked    map[string]int
}

// hunkLines is one hunk's deleted and added lines, awaiting matching.
type hunkLines struct {
	path    string
	deleted []deletedLine
	added   []addedLine
}

type deletedLine struct {
	origin    lineOrigin
	text, key string
	matched   bool
}

type addedLine struct {
	idx       int // position in the file's post-commit line slice
	text, key string
	matched   bool
}

func (t *reworkTracker) apply(c gitdiff.Commit) {
	hunks := t.splice(c)
	t.matchMoves(hunks)
	t.matchEdits(hunks)
	t.tally(hunks, c.Time.Unix())
}

// splice applies c's hunks to every touched file's line provenance and
// returns the hunks' lines for matching.
func (t *reworkTracker) splice(c gitdiff.Commit) []hunkLines {
	var hunks []hunkLines
	for _, f := range c.Files {
		if f.Binary {
			delete(t.files, f.Path)
			continue
		}
		lines, hs := spliceHunks(t.files[f.Path], f)
		hunks = append(hunks, hs...)
		if f.Removed {
			delete(t.files, f.Path)
		} else {
			t.files[f.Path] = lines
		}
	}
	return hunks
}

// inherit marks a and d as the same line, carrying d's provenance to a.
func (t *reworkTracker) inherit(path string, a *addedLine, d *deletedLine) {
	a.matched, d.matched = true, true
	t.files[path][a.idx] = d.origin
}

// matchMoves pairs lines removed and re-added with the same text (ignoring
// whitespace) anywhere in the commit. Since --no-renames reports a rename as
// a whole file deleted and re-added, this also carries provenance across
// renames.
func (t *reworkTracker) matchMoves(hunks []hunkLines) {
	removedByKey := map[string][]*deletedLine{}
	for i := range hunks {
		for j := range hunks[i].deleted {
			if d := &hunks[i].deleted[j]; d.key != "" {
				removedByKey[d.key] = append(removedByKey[d.key], d)
			}
		}
	}
	for i := range hunks {
		for j := range hunks[i].added {
			a := &hunks[i].added[j]
			if q := removedByKey[a.key]; a.key != "" && len(q) > 0 {
				t.inherit(hunks[i].path, a, q[0])
				removedByKey[a.key] = q[1:]
			}
		}
	}
}

// matchEdits pairs, in order, each hunk's remaining replaced and replacing
// lines that are still similar enough to be the same line.
func (t *reworkTracker) matchEdits(hunks []hunkLines) {
	for i := range hunks {
		h := &hunks[i]
		dels := unmatched(h.deleted, func(d *deletedLine) bool { return d.matched || d.key == "" })
		adds := unmatched(h.added, func(a *addedLine) bool { return a.matched || a.key == "" })
		for k := range min(len(dels), len(adds)) {
			if tokenSimilarity(dels[k].text, adds[k].text) >= minEditSimilarity {
				t.inherit(h.path, adds[k], dels[k])
			}
		}
	}
}

// tally counts what's left after matching, which is genuinely new code and
// genuinely removed code, for a commit landing at when (unix seconds).
func (t *reworkTracker) tally(hunks []hunkLines, when int64) {
	countable := when+t.window <= t.now
	for _, h := range hunks {
		if countable {
			for _, a := range h.added {
				if !a.matched && a.key != "" {
					t.files[h.path][a.idx] = lineOrigin{h.path, when}
					t.added[h.path]++
				}
			}
		}
		for _, d := range h.deleted {
			if !d.matched && d.origin.entity != "" && when-d.origin.when <= t.window {
				t.reworked[d.origin.entity]++
			}
		}
	}
}

// spliceHunks applies f's zero-context hunks to a file's line provenance,
// returning the post-commit provenance (added lines as zero-value
// placeholders) and each hunk's lines for matching.
func spliceHunks(old []lineOrigin, f gitdiff.FileDiff) ([]lineOrigin, []hunkLines) {
	next := make([]lineOrigin, 0, len(old))
	hunks := make([]hunkLines, 0, len(f.Hunks))
	cursor := 0
	for _, h := range f.Hunks {
		// With no deleted lines, OldStart is the line the insertion follows.
		start := h.OldStart - 1
		if len(h.Deleted) == 0 {
			start = h.OldStart
		}
		start = max(start, cursor)
		end := start + len(h.Deleted)
		if end > len(old) {
			// Lines from before the analyzed history (e.g. with --after) were
			// never seen; they're untracked.
			old = append(old, make([]lineOrigin, end-len(old))...)
		}
		next = append(next, old[cursor:start]...)

		hl := hunkLines{path: f.Path}
		for i, text := range h.Deleted {
			hl.deleted = append(hl.deleted, deletedLine{origin: old[start+i], text: text, key: normalizeLine(text)})
		}
		for _, text := range h.Added {
			hl.added = append(hl.added, addedLine{idx: len(next), text: text, key: normalizeLine(text)})
			next = append(next, lineOrigin{})
		}
		hunks = append(hunks, hl)
		cursor = end
	}
	if cursor < len(old) {
		next = append(next, old[cursor:]...)
	}
	return next, hunks
}

func (t *reworkTracker) results() []ReworkResult {
	results := make([]ReworkResult, 0, len(t.added))
	for entity, added := range t.added {
		reworked := t.reworked[entity]
		results = append(results, ReworkResult{entity, added, reworked, float64(reworked) / float64(added) * 100})
	}
	slices.SortFunc(results, func(a, b ReworkResult) int {
		if c := cmp.Compare(b.Reworked, a.Reworked); c != 0 {
			return c
		}
		return cmp.Compare(a.Entity, b.Entity)
	})
	return results
}

// unmatched returns pointers to the elements of s that skip rejects.
func unmatched[T any](s []T, skip func(*T) bool) []*T {
	var out []*T
	for i := range s {
		if !skip(&s[i]) {
			out = append(out, &s[i])
		}
	}
	return out
}

// normalizeLine collapses all whitespace runs, so re-indented or re-aligned
// lines compare equal. Blank lines normalize to "".
func normalizeLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// tokenSimilarity is the Dice coefficient of a's and b's word-token
// multisets: 2*|shared tokens| / (|a tokens| + |b tokens|). Punctuation is
// ignored — on short lines it would otherwise dominate, making e.g.
// "legacy()" and "fresh()" look 67% alike.
func tokenSimilarity(a, b string) float64 {
	counts := map[string]int{}
	na, nb := 0, 0
	for tok := range wordTokens(a) {
		counts[tok]++
		na++
	}
	shared := 0
	for tok := range wordTokens(b) {
		if counts[tok] > 0 {
			counts[tok]--
			shared++
		}
		nb++
	}
	if na+nb == 0 {
		return 0
	}
	return 2 * float64(shared) / float64(na+nb)
}

// wordTokens yields the identifier/number runs in s.
func wordTokens(s string) iter.Seq[string] {
	return strings.FieldsFuncSeq(s, func(r rune) bool {
		return r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
