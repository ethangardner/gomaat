# 2. A pre-generated log file is the input to every log-based analysis

- **Status:** Accepted
- **Date:** 2026-05-08 (original design); reaffirmed 2026-09-13 when `--repo` was declined

## Context

gomaat has two steps: `generate-log` writes a numstat log, then each analysis reads it with `-l/--log`. Issue #47 proposed a `--repo <path>` flag, mutually exclusive with `--log`, that would shell out to git and skip the intermediate file (`gomaat coupling --repo .`). PR #74 implemented it. It reused `generate-log`'s git invocation, promoted the log-filter flags to global flags, and produced output identical to the two-step flow.

## Decision

Log-based analyses keep reading a log file. `--repo` was not merged. PR #74 and issue #47 were closed with "Decided not to do this one."

From the README, the reasons for the two-step design: repeated analyses over one log are fast, and the log can be version-controlled so results are reproducible.

The maintainer's reason for declining `--repo`: gomaat is meant to work on repos in the current directory.

PR #74 would also have promoted `generate-log`'s filter flags (`--after`, `--before`, `--exclude`, `--exclude-author`, `--ignore-revs-file`, `--use-mailmap`) to global flags, adding them to every subcommand's surface.

`rework` is the one exception: it reads the repository directly. See [ADR 0010](0010-rework-reads-repository-directly.md).

## Consequences

- The log file is the artifact you share and pin, and it's what the smoke test diffs.
- Every analysis needs one extra command first. That friction is deliberate.
- Log-generation filters (mailmap, author and rev excludes) stay on `generate-log` alone, not on every subcommand.

## Sources

- README "Generating a Git Log" section
- Issue #47, PR #74 (closed, not merged)
