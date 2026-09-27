# 3. Split every analysis into a typed compute step and a format step

- **Status:** Accepted
- **Date:** 2026-06-14

## Context

The analyses were ported from code-maat, where each one produced output rows directly. That made results hard to assert on in tests and impossible for one analysis to reuse another's results without reparsing strings.

## Decision

Every analysis in `internal/analysis` has two functions:

```go
func XXX(commits []model.Commit, opts model.Options) T
func FormatXXX(results T, opts model.Options) [][]string
```

`T` is usually a slice of a small per-row struct. The first row returned by `FormatXXX` is always the header. `runAnalysis`, `simpleCmd`, and `newCouplingCmd` in `cli/root.go` are generic over `T`, so the compiler checks that a compute function and its formatter agree on the result type.

Shared behavior is expressed with generics instead of per-type copies. For example, `formatRows` renders any result slice, and `loadfile.Parse[T]` loads every input file. That package was renamed from `fileutil` to follow Go's advice against catch-all `util` package names.

## Consequences

- Tests assert on typed data, not CSV strings.
- Analyses compose. For example, `statistics` reuses `Revisions`, `Authors`, and `SumOfCoupling`, and `bus-factor` and `knowledge-loss` share the per-entity-author sums.
- Output formats (CSV, JSON) are handled once, over `[][]string`. See [ADR 0007](0007-json-output-shape.md).
- A new analysis must follow the pattern and register in `cli/root.go`'s `init()`. The `add-new-analysis` skill and CONTRIBUTING.md describe how.

## Sources

- PR #1, commit `8e9c21b` (why the command helpers are generic)
- PR #98, commit `06fe109` (`fileutil` → `loadfile`)
- AGENTS.md "Analysis functions"
