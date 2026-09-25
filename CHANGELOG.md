# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html) from `v1.0.0` onward. The pre-1.0 `v0.1.0-alphaN` tags predate that guarantee and could include breaking changes to flags or output columns without a major-version bump.

**Convention:** entries land under `[Unreleased]` in the same PR as the change they describe — new/changed/removed CLI flags, subcommands, or output columns, and user-visible bug fixes. Internal-only refactors are omitted unless they change the public Go API. When a release is tagged, `[Unreleased]` is renamed to the version and date, and a new empty `[Unreleased]` section is added above it.

## [Unreleased]

### Added

- `--use-mailmap` flag on `generate-log` to resolve author identities via a `.mailmap` file at the repo root, so the same person committing under different names/emails collapses to one canonical author. ([#44](https://github.com/ethangardner/gomaat/issues/44))
- `--exclude-author` flag on `generate-log` to drop commits by author name (repeatable, supports `*` globs), for filtering out bot accounts (dependabot, renovate, CI accounts). ([#44](https://github.com/ethangardner/gomaat/issues/44))
- `--ignore-revs-file` flag on `generate-log` to drop commits listed in a file, using the same format as `git blame --ignore-revs-file` — useful for excluding a single mass-reformat commit from churn/coupling numbers. ([#44](https://github.com/ethangardner/gomaat/issues/44))
- `rework` subcommand: per file, the share of added lines removed or substantially rewritten within `--rework-window` (default `14d`) of landing, as `entity, added-lines, reworked-lines, rework-ratio`. Moved lines and small edits (≥60% token-similar) are not counted as rework. Merge commits count as landing their branch's lines at the merge time. Unlike other analyses it reads the repository directly (`--path`, `--after`, `--before`, `--exclude`, positional pathspecs) instead of a `--log` file. See the README for the exact definition, known limitations, and measured runtime. ([#65](https://github.com/ethangardner/gomaat/issues/65))
- `bus-factor` subcommand: per entity, the fewest authors who together own more than 50% of its lines added, as `entity, bus-factor, top-owners, ownership`, sorted riskiest (`1`) first. ([#58](https://github.com/ethangardner/gomaat/issues/58))

### Changed

- `generate-log`'s rev field now uses the full commit hash (`%H`) instead of the abbreviated `%h`, so `--ignore-revs-file` can match unambiguously. `Rev` is an opaque string everywhere it's consumed, so this only changes the value in output, not its meaning. ([#44](https://github.com/ethangardner/gomaat/issues/44))

### Fixed

- Errors are printed once instead of twice. Cobra already reports the error with an `Error:` prefix above the usage text, and `Execute` was printing it a second time after the usage.

## [v1.0.0] - 2026-09-05

### Changed

- Updated to Go 1.27.1. ([#38](https://github.com/ethangardner/gomaat/pull/38))

### Documentation

- Documented a recipe for tracking metrics (e.g. `statistics`, `summary`) over time by generating one log per period with `generate-log --after/--before` and combining the results. ([#37](https://github.com/ethangardner/gomaat/pull/37))

## [v0.1.0-alpha.7] - 2026-08-30

### Added

- `--version` flag reporting the built binary version. ([#13](https://github.com/ethangardner/gomaat/pull/13))
- `statistics` subcommand: descriptive statistics (count/min/q1/median/q3/max/mean/stddev) for core metrics such as files/lines per commit and revisions/authors/sum-of-coupling per entity. ([#15](https://github.com/ethangardner/gomaat/pull/15))
- `--format`/`-f json` output option, alongside the existing CSV output. ([#33](https://github.com/ethangardner/gomaat/pull/33))
- `--before` flag on `generate-log` to bound history by date. ([#29](https://github.com/ethangardner/gomaat/pull/29))

### Fixed

- `WriteJSON` now returns an error on a header/row width mismatch instead of silently producing malformed output. ([#35](https://github.com/ethangardner/gomaat/pull/35))

### Documentation

- Documented the `statistics` command and `--version` flag in the README. ([#36](https://github.com/ethangardner/gomaat/pull/36))

## [v0.1.0-alpha.6] - 2026-08-01

### Changed

- Improved `--exclude` filtering performance on large repositories. ([#12](https://github.com/ethangardner/gomaat/pull/12))

## [v0.1.0-alpha.5] - 2026-07-31

### Changed

- Internal readability refactor and Go-proverbs cleanup; no user-facing behavior change. ([#9](https://github.com/ethangardner/gomaat/pull/9), [#10](https://github.com/ethangardner/gomaat/pull/10), [#11](https://github.com/ethangardner/gomaat/pull/11))

## [v0.1.0-alpha.4] - 2026-06-29

### Changed

- Split each analysis into a compute step (`XXX`) and a format step (`FormatXXX`), separating typed results from CSV rendering. This is an internal Go API change for anyone importing `internal/analysis` directly; CLI behavior is unaffected. ([#1](https://github.com/ethangardner/gomaat/pull/1))
- Deduplicated shared logic across analysis and CLI code; no user-facing behavior change. ([#3](https://github.com/ethangardner/gomaat/pull/3), [#4](https://github.com/ethangardner/gomaat/pull/4), [#6](https://github.com/ethangardner/gomaat/pull/6), [#7](https://github.com/ethangardner/gomaat/pull/7), [#8](https://github.com/ethangardner/gomaat/pull/8))
- Added a CI workflow that dogfoods gomaat by running it against this repo's own history. ([#2](https://github.com/ethangardner/gomaat/pull/2))

## [v0.1.0-alpha.3] - 2026-06-11

### Fixed

- Made `cloc`-reported paths relative and had `cloc` locate the repository root correctly. ([43809c6](https://github.com/ethangardner/gomaat/commit/43809c65a44fd84892e12df5790a48be96a3c758), [6603d5e](https://github.com/ethangardner/gomaat/commit/6603d5ea54b4ccc4338deb32f3e50a8b2117f0be))
- Fixed log handling and error handling on Windows. ([4c04570](https://github.com/ethangardner/gomaat/commit/4c04570b98f146c71af25839faa5412429e57207))

## [v0.1.0-alpha.2] - 2026-05-13

### Added

- `cloc` subcommand, limited to git-tracked files. ([2555db7](https://github.com/ethangardner/gomaat/commit/2555db7a60030455eda69e1a29c7037377494754), [c4e2bb5](https://github.com/ethangardner/gomaat/commit/c4e2bb5eba5a0874fddf08a9f485c2a5dba31fac))

### Changed

- Standardized table formatting and adopted a global flag pattern shared across subcommands. ([b2b267c](https://github.com/ethangardner/gomaat/commit/b2b267c564425bb84e43b1374e975de0ebc33108), [0849c6e](https://github.com/ethangardner/gomaat/commit/0849c6e4edc2183e20c2fa36d56b8b3fa0385fb8))

## [v0.1.0-alpha] - 2026-05-11

### Added

- Initial Go port of [code-maat](https://github.com/adamtornhill/code-maat): `authors`, `revisions`, `summary`, `identity`, `abs-churn`, `author-churn`, `entity-churn`, `entity-ownership`, `main-dev`, `refactoring-main-dev`, `entity-effort`, `main-dev-by-revs`, `fragmentation`, `communication`, `age`, `coupling`, `soc`, and `generate-log`.
- `--exclude` glob filtering for `generate-log`.
- CI: linting and a goreleaser-based release workflow.

[Unreleased]: https://github.com/ethangardner/gomaat/compare/v1.0.0...HEAD
[v1.0.0]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.7...v1.0.0
[v0.1.0-alpha.7]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.6...v0.1.0-alpha.7
[v0.1.0-alpha.6]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.5...v0.1.0-alpha.6
[v0.1.0-alpha.5]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.4...v0.1.0-alpha.5
[v0.1.0-alpha.4]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.3...v0.1.0-alpha.4
[v0.1.0-alpha.3]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha.2...v0.1.0-alpha.3
[v0.1.0-alpha.2]: https://github.com/ethangardner/gomaat/compare/v0.1.0-alpha...v0.1.0-alpha.2
[v0.1.0-alpha]: https://github.com/ethangardner/gomaat/releases/tag/v0.1.0-alpha
