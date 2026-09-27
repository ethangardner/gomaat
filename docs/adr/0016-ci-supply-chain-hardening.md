# 16. Pin, delay, and least-privilege the CI supply chain

- **Status:** Accepted
- **Date:** 2026-09-26

## Context

gomaat distributes binaries for Linux, macOS, and Windows through goreleaser, so a compromised action, dependency, or cache can reach users. Before #99 and #102:

- Actions were SHA-pinned, but nothing bumped the pins, and they had drifted: checkout was v6.0.2 in two workflows and v6.0.3 in another.
- `verify.yml` had no `permissions` block.
- `smoke-test.yml` expanded `${{ }}` directly inside a `run:` script.
- Tests ran only on Linux, even though release binaries ship for all three OSes.

## Decision

- **Pinning:** actions stay SHA-pinned. Dependabot bumps actions and Go modules weekly, grouped by ecosystem. Tool versions that an action pulls in are pinned too; for example, zizmor is pinned to 1.30.1 by commit, since the action's `version` input defaults to latest. Dependabot doesn't update those, so bump them by hand.
- **Cooldown:** Dependabot waits 7 days after a release before proposing it, which leaves time for a compromised or yanked version to be caught.
- **Least privilege:** every workflow sets `contents: read` at the top level. Only the smoke-test job (`pull-requests: write`) and the release job (`contents: write`) widen it. Checkouts use `persist-credentials: false`.
- **Cache isolation:** the release build doesn't use setup-go's cache, so a tag build can't restore a cache written by a PR run.
- **Linting and scanning:** actionlint (with shellcheck) and zizmor run on workflow changes. `govulncheck` runs on PRs, pushes to main, and weekly.
- **Cross-OS tests:** the full test suite runs on Linux, macOS, and Windows with `-race`. Code builds paths with `filepath` and doesn't assume `/` separators or LF line endings.

## Consequences

- There's a weekly stream of Dependabot PRs to merge.
- Tool pins inside action inputs are a manual chore.
- Windows and macOS runners surfaced real path bugs: symlinked temp dirs, 8.3 short names like `RUNNER~1`, and forward-slash repo roots. Those bugs would otherwise have shipped.

## Sources

- Issues #99, #102; PRs #105, #106
- Commits `e533a61`, `79a1c38`, `fe5cbb2`, `8afe5dd`, `0b065fb`
