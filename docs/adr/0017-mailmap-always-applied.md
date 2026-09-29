# 17. `.mailmap` is always applied; no `--use-mailmap` flag

- **Status:** Accepted
- **Date:** 2026-09-29

## Context

[ADR 0009](0009-identity-hygiene-at-log-generation.md) added `generate-log --use-mailmap` to collapse one person's several git identities into one author, on the premise that `%aN` doesn't resolve `.mailmap` without it. That premise was wrong. Git's `%aN` placeholder always applies the repository's `.mailmap`, whatever `--use-mailmap` is set to. The flag only changes the default output headers, which `generate-log` never uses.

Pre-release testing for v1.1.0 against a clone with a `.mailmap` that remapped the dependabot author confirmed it. The commit was remapped with and without the flag, and left alone in both cases without the file. The original test only ran with the flag on, so it couldn't catch this.

Options considered:

- **Remove the flag.** It never shipped in a release, so removing it breaks no one. Behaviour stays the same as v1.0.0.
- **Make it a real opt-in** by using `%an` by default and `%aN` with the flag. Rejected: it changes the default for v1.0.0 users who already have a `.mailmap`, so their author counts would shift in a minor release.
- **Keep it and fix the docs.** Rejected: a flag that does nothing is API surface that has to be kept for the whole major version.

## Decision

`generate-log` has no `--use-mailmap` flag. Author names come from `%aN`, so a `.mailmap` at the repository root is always applied. The docs tell users to add a `.mailmap` to collapse aliases.

## Consequences

- One fewer flag, and the behaviour matches what v1.0.0 already did.
- There is no way to turn mailmap resolution off. If someone needs raw identities, that would be a new opt-out flag switching to `%an`, not a revival of `--use-mailmap`.
- ADRs 0002 and 0012 mention `--use-mailmap` as it stood when they were written. They were left unchanged because they are point-in-time records.

## Sources

- Supersedes the `--use-mailmap` part of [ADR 0009](0009-identity-hygiene-at-log-generation.md) (#44, PR #73)
- [git pretty-formats: `%aN`](https://git-scm.com/docs/pretty-formats), "author name (respecting .mailmap)"
