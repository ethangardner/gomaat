# 10. `rework` streams the repository's patch history instead of reading a log file

- **Status:** Accepted
- **Date:** 2026-09-23

## Context

`rework` (#65) measures the share of added lines that are removed or substantially rewritten within a window of landing. A numstat log says how many lines changed, not which ones, so it can't answer that. The analysis needs line-level diffs.

Two sub-questions came up while building it:

1. **Merges.** On this repo, 31 of 59 first-parent commits were PR merges, so a first-parent walk that ignored merges missed most of the code.
2. **Streaming vs buffering git's output.** Measured patch output and peak memory while streaming:

| Repository | `git log -p -U0` output | `rework` peak memory (streaming) |
|---|---|---|
| gomaat | 0.5 MB | 31 MB |
| uswds, full history | 85 MB | 247 MB |
| kubernetes since 2025-06, no vendor | 160 MB | 398 MB |

Buffering would add the whole output to peak memory, and more once it's converted to Go strings.

## Decision

- `rework` bypasses `runAnalysis`. It runs `git log --reverse --first-parent --diff-merges=first-parent -p -U0 --no-renames` with pinned `--src-prefix=a/ --dst-prefix=b/` (so `diff.noprefix` can't change path parsing). It streams the output through `internal/gitdiff.Parse` into `analysis.Rework`, which takes an `iter.Seq2[gitdiff.Commit, error]`.
- Merged lines count as landing at the merge's date. That makes squash, rebase, and merge-commit repos comparable, and editing on a branch before it merges isn't rework.
- Log-only flags (`--log`, `--group`, `--team-map-file`) give an error instead of being silently ignored.
- `--before` doubles as the "now" for deciding which windows have elapsed, so historical runs are reproducible. Lines younger than the window are excluded from both counts.
- Streaming is kept. `streamGit` is shared with `generate-log`, which already streamed.

## Consequences

- This is the one analysis that needs a live repository, so it doesn't work on a shared log file. It's an exception to [ADR 0002](0002-log-file-as-pipeline-input.md).
- Memory stays flat per commit. Kubernetes-scale windows run in about 13 s and 400 MB.
- `-g` and `-p` aren't supported for `rework`.

## Sources

- Issue #65; PRs #83, #84, #88, #89, #90 (commits `8bd0063`, `8d5842f`, `2a9cfa7`, `043450b`)
- Rework session, 2026-09-24 ("Count at merge time"; "is the git stream necessary?")
