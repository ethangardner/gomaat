# 8. No time-decay (`--half-life`) weighting

- **Status:** Accepted
- **Date:** 2026-09-13

## Context

Every analysis weights a commit from years ago the same as one from last week. Issue #45 proposed `--half-life <days>`, which would weight each commit by `0.5^(age / half_life)` relative to `--age-time-now`. It would apply to `revisions`, `coupling`, `soc`, and the ownership analyses (`entity-ownership`, `fragmentation`, `main-dev`, `refactoring-main-dev`, `main-dev-by-revs`).

PR #75 implemented it:

- Plain number of days as the input format.
- `--age-time-now` promoted to a global flag so decay and `age` share one "now".
- Output byte-for-byte unchanged when the flag is off.
- Affected result fields changed from `int` to `float64`, so counts become weighted sums.

## Decision

Not merged. PR #75 and issue #45 were closed with "Decided not to do this."

Reason: most of the benefit is already available without adding `--half-life`. To focus on recent activity, generate a log for a window with `generate-log --after/--before`. To see whether a file is heating up or cooling down, run an analysis over full history and over a recent window, then join the two outputs (see the README's "Tracking Metrics Over Time" recipe). Both approaches keep counts as integers that are easy to explain, and they work for every command, not just the eight that decay would have touched.

Not adding it also avoided what PR #75 would have changed: the public result types in `internal/analysis`, which would have gone from `int` to `float64`; the meaning of `n-revs` and coupling counts, which would have become weighted sums when the flag was on; and `--age-time-now`, which would have become a global flag.

## Consequences

- Counts stay integers, and every analysis means the same thing it did in code-maat.
- "Recent hotspot" questions are answered by windowing the log, not by weighting inside the analysis.
- A window drops everything before its cutoff, where decay would only fade it. Two cases lose the most: `coupling` on a slow-moving repo, where a short window may not reach `--min-shared-revs`, and ownership, where a past main developer drops out entirely. If either becomes a real need, revisit this.
- `whoknows` (#57, recency-weighted ownership) is still open and would need its own answer to the same question.

## Sources

- Issue #45, PR #75 (closed, not merged)
- Planning session for #44–#47 (input format and `main-dev` scope decisions)
