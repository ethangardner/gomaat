# 11. Rework's edit-vs-rewrite test: token Dice at 0.6 with a punctuation fallback

- **Status:** Accepted
- **Date:** 2026-09-24

## Context

When a commit replaces a line, `rework` must decide whether it's a small edit (the line survives) or a rewrite (the original counts as reworked). Review on PR #85 asked to explore other similarity algorithms and thresholds.

About 50k real in-hunk replaced/replacing line pairs were pulled from kubernetes (Go), uswds (SCSS/JS), and site-scanning-engine (TS). They were scored with six metrics, and a uniform random sample of 160 was hand-labeled as edit or rewrite:

| Metric | Threshold | Agreement with labels |
|---|---|---|
| Token Dice (before) | 0.6 | 85.6% |
| Token Dice + punctuation fallback | 0.6 | 90.0% |
| Normalized Levenshtein | 0.5 | 90.6% |
| Jaccard, char bigrams, token LCS | best for each | ≤ 87.5% |

Almost half of plain Dice's misses were lines with no words (`)` → `),`, `}` → `},`), which always scored 0 and counted as rework. They're 4.4% of all pairs and 7.2% in uswds.

## Decision

- Use Dice similarity over word tokens with a threshold of 0.6. When either line has no words, compare punctuation tokens instead.
- **Levenshtein rejected:** about as accurate, but O(n·m) per pair, and it scores shared structure as similarity (unrelated hex blobs come out about 70% alike).
- **Threshold stays at 0.6:** 0.5 was within noise. Of the 8 labeled pairs in [0.5, 0.6), 5 were edits and 3 were rewrites. The code comment on `minEditSimilarity` records this.

## Consequences

- The remaining known misses: one changed token on a 2–3 token line, and sibling list or map entries that share naming prefixes. No cheap metric fixed these.
- The labels are one person's judgment, and a sample of 160 can't separate results within a few points. Change the threshold or metric only with a larger labeled sample.
- The tokenizers are a direct-call iterator on purpose. An indirect call through a func variable moved iterator state to the heap (+60% allocs/op).

## Sources

- PR #85 review; commit `da38af3`
- `internal/analysis/rework.go` (`minEditSimilarity`)
