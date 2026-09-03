---
name: update-changelog
description: Curates CHANGELOG.md's [Unreleased] section (or cuts a new version section) from PRs merged since the last tag, using gh CLI. Use when asked to update, regenerate, or fill in the changelog, or as part of preparing a release.
---

`CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Per the convention documented in `AGENTS.md`, entries are supposed to land under `[Unreleased]` in the same PR as the change — this skill is the catch-up path for PRs that skipped that, or for drafting a release's final section.

Never invent entries from commit subjects alone (that's what `goreleaser`'s raw per-commit release notes already do, and is exactly what starting this file was meant to replace) — every bullet must come from actually reading the PR.

## 1. List merged PRs to cover

```bash
.claude/skills/update-changelog/fetch-merged-prs.sh          # since the last tag
.claude/skills/update-changelog/fetch-merged-prs.sh <ref>     # since an arbitrary ref
```

Returns merged PRs (oldest first) as JSON: `number`, `title`, `url`, `mergedAt`, `labels`.

Skip any PR number that already appears in `CHANGELOG.md` (grep for `pull/<n>)`) — this skill is meant to be re-run incrementally as new PRs land.

## 2. Categorize each PR

For each remaining PR, don't trust the title alone — read what actually changed:

```bash
gh pr view <n> --json body -q .body   # PR description, if substantive
gh pr diff <n>                        # the actual diff, when the title/body is ambiguous
```

Decide:
- **Is it user-facing?** A new/changed/removed CLI flag, subcommand, or output column, a behavior/bug fix, or a documentation addition (README, etc.) — include it.
- **Is it internal-only?** A refactor, test-only change, dedup, formatting, or CI tweak with no effect on the CLI's behavior or its public Go API (`internal/analysis` signatures, etc.) — omit it, unless it *does* touch that public API, in which case note it under Changed as an internal-API note (see the `v0.1.0-alpha.4` entry in `CHANGELOG.md` for the pattern).

Map to a Keep a Changelog heading: `Added`, `Changed`, `Fixed`, `Removed`, `Deprecated`, `Security`, or this repo's extra `Documentation` heading (used here for README/doc-only additions). Only include headings that have at least one bullet.

## 3. Write the entry

Match the existing style in `CHANGELOG.md` exactly:

```markdown
- <One concise, user-facing sentence, present tense, no trailing PR title copy-paste>. ([#<n>](https://github.com/ethangardner/gomaat/pull/<n>))
```

If several merged PRs are really one logical change (e.g. a feature landed across a few follow-up PRs), fold them into one bullet with multiple citations, as `v0.1.0-alpha.4`'s dedup entry does.

Insert bullets under the right heading inside `[Unreleased]` at the top of the file. Create a heading if none of that category exist yet; keep the heading order Added, Changed, Deprecated, Removed, Fixed, Security, Documentation.

## 4. If cutting a release (a version, not just "catch up unreleased")

1. Rename `## [Unreleased]` to `## [vX.Y.Z] - YYYY-MM-DD` (the date the tag is/will be cut).
2. Add a fresh empty `## [Unreleased]` section above it.
3. Update the link footer at the bottom of the file:
   - Change `[Unreleased]: .../compare/<old-latest-tag>...HEAD` to `.../compare/vX.Y.Z...HEAD`.
   - Add `[vX.Y.Z]: .../compare/<old-latest-tag>...vX.Y.Z` below it.

## 5. Verify

- `git diff CHANGELOG.md` reads cleanly as prose, not a commit dump.
- Every bullet has a working `pull/<n>` link (compare against the PR numbers from step 1).
- No internal-only PR made it in unless it changes the public Go API.
