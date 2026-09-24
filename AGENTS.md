# AGENTS.md

Guidance for agents working in this repository.

## What this is

gomaat is a Go port of [code-maat](https://github.com/adamtornhill/code-maat) — it mines `git log` output to surface design insights (temporal coupling, churn, authorship, fragmentation, communication patterns, etc.), inspired by *Your Code as a Crime Scene* and *Software Design X-Rays*.

## Commands

```bash
make build         # compile to ./bin/gomaat
make check         # fmt + vet + lint + test (mirrors CI)
make test          # go test -cover ./...
make test-verbose  # go test -v -cover ./...
make fmt           # gofmt all packages
make vet           # go vet ./...
make lint          # golangci-lint run
make tidy          # go mod tidy && go mod verify
make watchtest     # rerun tests on file change (requires entr)
```

Run a single test:
```bash
go test ./internal/analysis/ -run TestCoupling -v
```

CI (`.github/workflows/verify.yml`) runs `go vet`, `gofmt -l .` (must be empty), `golangci-lint run`, and `go test -coverprofile=coverage.out ./...` followed by `go tool cover -func=coverage.out` to report per-function coverage. This is visibility only — there's no enforced minimum threshold yet. Run `golangci-lint run` locally before finishing if it's available — there's no repo-specific golangci config, so default rules apply.

## Changelog

`CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Add an entry under `[Unreleased]` in the same PR as any user-facing change — a new/changed/removed CLI flag, subcommand, or output column, or a user-visible bug fix. Skip it for internal-only refactors unless they change the public Go API (e.g. `internal/analysis` function signatures). On tagging a release, `[Unreleased]` is renamed to the version and date.

## Architecture

### Pipeline

Every analysis subcommand follows the same pipeline, orchestrated by `runAnalysis` in `cli/root.go`:

```
log file --[parser.ParseFile]--> []model.Commit
        --[grouper.Apply]-->      (optional, if -g given: remap Entity -> group name, drop unmatched)
        --[teammapper.Apply]-->   (optional, if -p given: remap Author -> team, drop unmapped)
        --[analysis.XXX]-->       typed result (e.g. []XXXResult)
        --[analysis.FormatXXX]--> [][]string  (header row + data rows)
        --[output.Write/WriteJSON]--> CSV or JSON (per --format/-f) to stdout or file
```

- `internal/parser`: parses the custom log format produced by `generate-log`
  (`--rev--date--author` header lines followed by `numstat` lines: `added\tdeleted\tpath`).
  `--no-renames` is baked in deliberately — renames are tracked as delete+add to avoid inflating coupling between old/new paths.
- `internal/model`: shared `Commit` struct and `Options` (all CLI flags that affect analysis, e.g. `MinRevs`, `MinCoupling`, `MaxChangesetSize`, `AgeTimeNow`, `VerboseResults`).
- `internal/grouper`: maps file paths to architectural group names via prefix or `^`-prefixed regex rules; commits matching no group are dropped.
- `internal/teammapper`: maps author names to team names via CSV; commits for unmapped authors are dropped.
- `internal/output`: thin CSV/JSON writer shared by every subcommand; first row of `[][]string` is the header (also used as JSON object keys), `-r/--rows` caps data rows. `--format/-f` selects `csv` (default) or `json`; `cloc` supports it too via the same `writeRows` dispatch in `cli/root.go`, but `generate-log` rejects anything but `csv` since it streams raw git-log text, not tabular rows.

### Analysis functions

Every analysis lives in `internal/analysis/` and is split into a compute step and a format step:

```go
func XXX(commits []model.Commit, opts model.Options) T    // compute typed results
func FormatXXX(results T, opts model.Options) [][]string  // render to CSV rows
```

`T` is usually `[]XXXResult`, a slice of a small per-row struct (e.g. `CouplingResult`), but can be a single struct (`Summary` -> `SummaryResult`) or `[]model.Commit` (`Identity`). The first row returned by `FormatXXX` is always the CSV header. Keeping the typed result separate from formatting lets tests assert on the data directly and lets other analyses reuse the computed results.

`runAnalysis`, `simpleCmd`, and `newCouplingCmd` in `cli/root.go` are generic over `T`: they call `XXX` to get the typed result, then `FormatXXX` to get CSV rows. New analyses should follow this pattern and be registered in `cli/root.go`'s `init()`:
- Most subcommands need no extra flags — register with `simpleCmd(use, short, analysis.Fn, analysis.FormatFn)`.
- Coupling-style subcommands (those needing the `--min-revs`/`--min-shared-revs`/`--min-coupling`/`--max-coupling`/`--max-changeset-size` thresholds) use `newCouplingCmd`.
- `age` and `generate-log`/`cloc` are registered individually because they need bespoke flags.

### CLI structure (`cli/` package)

- `root.go`: cobra root command, persistent flags (`--log/-l`, `--outfile/-o`, `--rows/-r`, `--group/-g`, `--team-map-file/-p`, `--format/-f`), and subcommand registration.
- `generate_log.go`: runs the canonical `git log --all --numstat --date=short --pretty=format:'--%h--%ad--%aN' --no-renames --no-merges [...]` and streams it through `--exclude` glob filtering. `--no-merges` is baked in deliberately too — it keeps a combined merge diff from being double-counted against the commits it merges.
- `cloc.go`: wraps `gocloc` over `git ls-files` output (so it respects gitignore), with the same `--exclude` filtering logic as `generate-log`.
- `git.go`: `streamGit` runs a git subcommand and hands its stdout to a consumer as it's produced; `generate-log` uses it. `buildPathspecArgs` builds the trailing `-- <pathspec>...` arguments from includes and directory excludes.

Both `generate-log --exclude` and `cloc --exclude` share `matchesExcludePattern`: patterns ending in `/` match path prefixes, everything else is matched as a glob against both the full path and the basename.

## Development

When doing development, you must always do simplification passes and use modern Go idioms. Strive to be DRY, always write tests with code changes, and use generics instead of multiple formatters for different types.