# 15. Noisy CI signals inform rather than gate

- **Status:** Accepted
- **Date:** 2026-08-25 (coverage); 2026-08-30 (smoke-test trend declined); 2026-09-26 (benchmarks, fuzzing)

## Context

Several CI checks produce measurements rather than pass/fail answers: coverage percentage, benchmark deltas, and the smoke test's diff of analysis output between the base and head builds. On shared runners, benchmark deltas are noisy. There isn't enough history to know what normal smoke-test churn looks like, or what coverage floor makes sense.

## Decision

Deterministic checks gate the build: fmt, vet, lint, `go mod tidy` drift, tests (with `-race`, on Linux, macOS, and Windows), a changelog entry, the goreleaser snapshot build, and workflow lint. Measurements are reported but don't fail the build:

- **Coverage:** `go test -coverprofile` plus `go tool cover -func` on every run. No minimum threshold (#16).
- **Benchmarks:** six alternating base/head runs with a `benchstat` comparison in the job summary. The job fails only if a benchmark itself fails on head, never on a delta (#104).
- **Smoke test:** a sorted diff of every analysis's output, posted as a sticky PR comment and job summary. It isn't persisted across runs or gated. #24 proposed persisting it as a trend and was closed as won't do: every approach (structured artifacts, a CI-committed history file, a post-merge job) adds permissions, bot commits, or triggers, and that cost isn't justified for a solo-maintainer repo at this scale.
- **Fuzzing:** 30 s on PRs that touch `internal/parser` or `internal/gitdiff`, and 10 min weekly. A failure uploads the input so it can be committed under `testdata/fuzz/` as a regression seed.

## Consequences

- No flaky red builds from runner noise.
- Regressions in these measurements depend on a reviewer reading the summary or comment.
- Revisit #24 if the smoke test fires often enough that reading PR comments stops being enough, or if gating becomes a goal. Revisit coverage if a floor is needed for a release criterion.

## Sources

- Issues #16, #24 (closing comment), #101, #103, #104; PRs #27, #109, #110, #111
- AGENTS.md CI section
