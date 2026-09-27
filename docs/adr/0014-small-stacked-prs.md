# 14. Keep PRs under about 400 lines of runtime code, splitting with `gh stack`

- **Status:** Accepted
- **Date:** 2026-09-14 (budget stated); 2026-09-24 (applied to `rework`)

## Context

The first `rework` implementation was about 750 lines of new runtime code: a git streaming refactor, a patch parser, the analysis, and the CLI. That's too large to review with focus. The same happened with `knowledge-loss`/`bus-factor`.

## Decision

Each PR's runtime diff (non-test, non-doc Go) stays under about 400 lines. Larger work is split into a stack of dependent PRs with `gh stack`, each passing `make check` on its own. Tests, docs, README, and CHANGELOG don't count toward the budget.

To make stacks work:

- CI `pull_request` triggers have no base-branch filter, so every PR in a stack gets CI, not just the bottom one.
- The smoke test builds and diffs against the PR's base commit, not `main`, so each layer's comment shows only its own output changes.
- `bench.yml` compares each layer against the PR below it.
- A preparatory refactor lands first as its own PR (e.g. #83 before the `rework` stack), so feature PRs contain only new behavior.

## Consequences

- Reviews stay focused, and each layer can be verified on its own.
- Stacks cost coordination: rebasing layers, force-pushing with lease, and merging in order. PRs #85/#86 were once merged into a stale intermediate branch instead of `main` and had to be redone as a new stack (#88–#90).

## Sources

- Hotspots/risk/defects session, 2026-09-14 ("keep the runtime changes around 400 lines")
- Rework split session, 2026-09-24 ("I like things that have runtime code that's under 400 lines"); PRs #83–#90
- `knowledge-loss` session, 2026-09-25 ("this PR is too damn big. Split it up with a stack.")
- Commits `fe5cbb2`, `3ead56c` (CI triggers and smoke-test base for stacks)
