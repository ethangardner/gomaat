# 9. Author identity and noise filtering happen at log generation, keyed by full hash

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Using `%aN` with no mailmap, one person committing from several git configs appears as several authors. That corrupts `authors`, `fragmentation`, `entity-ownership`, `main-dev`, and `communication`. Bot accounts and mass-reformat commits skew churn and coupling, and there was no way to exclude them (#44).

## Decision

`generate-log` gains `--use-mailmap`, `--exclude-author`, and `--ignore-revs-file`. The resulting log is already clean, so no analysis needs to know about these filters.

- **Full hashes.** The log's rev field switched from `%h` to `%H`, so `--ignore-revs-file` matches `.git-blame-ignore-revs`-style lists, which use full SHAs, without ambiguity. `Rev` is opaque downstream, so only its value changed.
- **`--exclude-author` matches the author name only, case-sensitively.** `*` is the only wildcard. Brackets are literal, because `filepath.Match` would treat `dependabot[bot]` as a character class, and that's the main account this flag exists to exclude.
- **One streaming pass.** Author, rev, and path filters run together in one header-aware filter over git's stdout, which is never buffered. Directory excludes are pushed down to git as pathspecs.

## Consequences

- Every existing analysis gets more accurate without code changes.
- Old logs generated with `%h` still parse, but can't be matched against a full-hash ignore list.
- The filter's header detection must stay in step with `internal/parser`'s. Any log-format change (e.g. #60) has to update both.

## Sources

- Issue #44, PR #73 (commit `ecd7c5a`)
- Planning session for #44–#47 ("Switch to %H (full hash) everywhere", "Name only, case-sensitive")
