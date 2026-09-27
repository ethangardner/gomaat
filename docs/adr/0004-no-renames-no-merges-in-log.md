# 4. Bake `--no-renames` and `--no-merges` into `generate-log`

- **Status:** Accepted
- **Date:** 2026-05-08 (original design); rename-following deferred 2026-09-10

## Context

With rename detection on, git reports a moved file under both its old and new paths in one commit. For temporal coupling, that looks like two files changing together, which inflates coupling between a file and its own former name. Merge commits cause a similar problem: a combined merge diff repeats changes the merged commits already reported.

The cost of `--no-renames` is that a renamed file's history splits across two entities, which understates its `revisions`, `age`, and `soc`. Issue #46 proposed an opt-in `--follow-renames` that builds a rename graph and resolves paths to one identity before aggregating.

## Decision

`generate-log` always passes `--no-renames` and `--no-merges`. Neither is exposed as a flag.

`--follow-renames` (#46) is deferred. When #44–#47 were planned, it was skipped because it changes what "entity identity" means for nearly every analysis and risks subtly wrong results. Its own issue recommends sequencing it after the commit-message log-format work (#60), since both touch the parser and the git invocation.

## Consequences

- Coupling numbers aren't polluted by renames or double-counted merges.
- Renamed files have fragmented history. The README notes this.
- `rework` uses `--no-renames` too, but its move detection keeps a line's origin when the same text is re-added in the same commit, so renames don't show up as rework.
- If #46 is picked up, its acceptance criteria require a regression test proving a renamed file doesn't appear coupled to its old name.

## Sources

- AGENTS.md "CLI structure"; README note under "Generating a Git Log"
- Issue #46 (open); planning session for #44–#47 (answer: "Skip 46 for now")
