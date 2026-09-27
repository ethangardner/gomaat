# 7. JSON output is an array of objects keyed by the CSV header, and ragged rows are an error

- **Status:** Accepted
- **Date:** 2026-08-27 (JSON added); 2026-08-29 (width check)

## Context

CSV-only output meant anyone embedding gomaat in a dashboard or CI job needed a CSV parser (#23). Two ways to add JSON: give every analysis a second, typed JSON path, or render JSON from the rows every formatter already produces.

A later review (#34) found that `WriteJSON` zipped header and row by position. A short row silently produced an object with missing keys, and a long row silently dropped columns. CSV output doesn't have that failure mode.

## Decision

- `--format/-f json` renders the same `[][]string` as CSV: an array of objects whose keys are the header row. No analysis has a separate JSON path.
- `WriteJSON` returns an error when any row's width differs from the header's, checked in the same pass that builds the records, before anything is written.
- `generate-log` rejects `--format json`, because it streams raw git-log text, not tabular rows. `cloc` supports both formats through the same `writeRows` dispatch. Accepting `--format` and silently ignoring it was the bug being fixed.

## Consequences

- Adding an analysis gets JSON for free.
- All JSON values are strings, as in CSV. Consumers convert numeric columns themselves.
- A formatter that breaks the header/row width invariant fails loudly.

## Sources

- Issue #23, PR #33; issue #34, PR #35 (commit `f177248`)
- Review session 2026-08-27/28 (`--format` ignored by `cloc` and `generate-log`)
