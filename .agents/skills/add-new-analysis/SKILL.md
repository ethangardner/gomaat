---
name: add-new-analysis
description: Adds new analysis functionality to the CLI application
---

# Adding a new analysis

1. Add `internal/analysis/<name>.go` with a function matching `analysisFunc`, returning details that match the existing analyses.
2. Add a test in `internal/analysis/<name>_test.go` (table-driven against small `[]model.Commit` fixtures — see `simple_test.go` for the pattern).
3. Register it in `cli/root.go` via `simpleCmd`, `newCouplingCmd`, or a bespoke `cobra.Command` if it needs unique flags.
4. Document it in the README's Analyses section/TOC.
5. Add it to `.github/smoke-test/run.sh` so the PR smoke test diffs its output between `main` and the PR.
6. Add an entry under `[Unreleased]` in `CHANGELOG.md` (CI fails the PR without one).
