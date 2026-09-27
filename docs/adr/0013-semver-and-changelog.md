# 13. SemVer from v1.0.0, and a curated, CI-enforced CHANGELOG

- **Status:** Accepted
- **Date:** 2026-09-03 (changelog); 2026-09-05 (v1.0.0); 2026-09-26 (CI enforcement)

## Context

The project cut seven `v0.1.0-alpha` tags and no stable release. Perpetual alpha is a risk for anyone scripting CI against gomaat's flags and output columns (#21, #25). goreleaser's release notes are a flat list of commit subjects, including merges and housekeeping, and nobody can review them before a tag is cut (#22).

## Decision

- **Versioning:** Semantic Versioning from `v1.0.0` (tagged 2026-09-05). The pre-1.0 alpha tags predate that guarantee.
- **Changelog:** `CHANGELOG.md` in Keep a Changelog format. Entries go under `[Unreleased]` in the same PR as any user-facing change: a new, changed, or removed flag, subcommand, or output column, or a user-visible bug fix. Internal refactors are omitted unless they change the public Go API. On tagging, `[Unreleased]` is renamed to the version and date.
- **Enforcement:** `changelog.yml` fails a PR that changes non-test Go source under `cli/`, `cmd/`, or `internal/` without touching `CHANGELOG.md`, unless the PR has the `skip-changelog` label.
- **Curation help:** the `update-changelog` skill categorizes merged PRs by reading their diffs, not their commit subjects.
- **Deprecation policy (#26):** deferred ("deferring this for later"). There's no documented deprecation window for flags or output columns yet.

## Consequences

- Any breaking change to a flag or output column now requires a major version bump.
- Without a deprecation policy, it's undefined whether a flag can be removed with a warning first or must be removed in a major release. Decide this before the first breaking change after 1.0.
- The CI check catches a forgotten entry but can't tell whether a change is user-facing. Applying `skip-changelog` is a judgment call.

## Sources

- Issues #21, #22, #25, #26, #101; PRs #39, #40, #109
- v1.0.0 prep session, 2026-09-05
