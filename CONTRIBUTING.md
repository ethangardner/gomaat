# Contributing to gomaat

Thanks for considering a contribution. gomaat is a Go port of
[code-maat](https://github.com/adamtornhill/code-maat) that mines `git log`
output to surface design insights. See [AGENTS.md](AGENTS.md) for the full
architecture and pipeline breakdown — this file covers the conventions you
need to open a pull request.

## Adding a new analysis

Every analysis follows the same compute/format/register pattern:

```go
func XXX(commits []model.Commit, opts model.Options) T    // compute typed results
func FormatXXX(results T, opts model.Options) [][]string  // render to CSV rows
```

New analyses live in `internal/analysis/` and are registered as a subcommand
in `cli/root.go`'s `init()`. See AGENTS.md's ["Analysis
functions"](AGENTS.md#analysis-functions) and ["CLI
structure"](AGENTS.md#cli-structure-cli-package) sections for the details —
including which helper (`simpleCmd` vs. `newCouplingCmd`) to register with.

## Branch naming

Branches are merged into `main` via pull request, named by kind of change:

- `task/<issue-number>-short-slug` — a specific issue or small task
- `feature/short-slug` — new functionality
- `refac/short-slug` — refactors with no behavior change

## Before opening a pull request

Run `make check` (fmt + vet + lint + test — mirrors CI in
`.github/workflows/verify.yml`). If your change is user-facing (a new/changed/
removed CLI flag, subcommand, or output column, or a user-visible bug fix),
add an entry under `[Unreleased]` in `CHANGELOG.md` per the convention
documented in AGENTS.md.

## License

By contributing, you agree your contributions will be licensed under this
project's [MIT License](LICENSE).
