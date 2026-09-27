# 12. `knowledge-loss` takes a `--former-authors` list; team maps stay strict

- **Status:** Accepted
- **Date:** 2026-09-25

## Context

`knowledge-loss` (#58) reports, per file, the share of lines written by people who no longer work on the code. It needs a list of those people. The first draft called the flag `--departed` and the column `departed-added`. It also had the team-map CSV and the new author list share one lenient CSV reader.

Review found two problems:

- A lenient reader made the team map silently drop a row with a missing team (`Bob` with no `,Team`). That author's commits then vanished from every `-p` run with no error.
- Names containing quotes (`Bob "The Builder" Smith`) failed to parse.

## Decision

- **Naming:** `--former-authors` and `former-added`. "Departed" reads as though the person died.
- **Parsing:** the team map keeps its strict field-count check. Only the author list accepts varying column counts and bare quotes. A name with a comma must be quoted. Both still share one CSV-reading path (`teammapper.LoadAuthorsFile`).
- **Matching:** names match exactly, after team mapping when `-p` is set. The README tells users to pair this with `generate-log --use-mailmap`.
- No warning when no names match. This was offered and not chosen.

## Consequences

- A typo or a case mismatch in the list (`alice` vs `Alice`) produces a report where every file shows 0%, which looks the same as "no risk". The README says names must match exactly and recommends `--use-mailmap`, but it doesn't call out the all-zero symptom.
- Adding a leniency to one loader doesn't weaken the other.

## Sources

- Issue #58; PRs #93, #95, #96 (commits `23233ab`, `26c10b3`)
- Implementation session, 2026-09-25 (review triage answers; "Is there a different name for departed?")
