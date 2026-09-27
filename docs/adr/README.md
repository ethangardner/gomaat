# Architecture Decision Records

Significant decisions about gomaat's design, scope, and process, including ideas that were considered and declined. See [ADR 0001](0001-record-architecture-decisions.md) for the format and conventions. To add one, copy [template.md](template.md) and take the next number.

| # | Decision | Status |
|---|---|---|
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted |
| [0002](0002-log-file-as-pipeline-input.md) | A pre-generated log file is the input to every log-based analysis (no `--repo`) | Accepted |
| [0003](0003-compute-format-split.md) | Split every analysis into a typed compute step and a format step | Accepted |
| [0004](0004-no-renames-no-merges-in-log.md) | Bake `--no-renames` and `--no-merges` into `generate-log` | Accepted |
| [0005](0005-stateless-pipeline-no-index.md) | Stay a stateless text pipeline; no on-disk index or cache | Accepted |
| [0006](0006-structured-data-only-no-visualization.md) | Output structured data only; visualization and integrations live downstream | Accepted |
| [0007](0007-json-output-shape.md) | JSON is an array of objects keyed by the header; ragged rows are an error | Accepted |
| [0008](0008-no-time-decay-weighting.md) | No time-decay (`--half-life`) weighting | Accepted |
| [0009](0009-identity-hygiene-at-log-generation.md) | Author identity and noise filtering happen at log generation, keyed by full hash | Accepted |
| [0010](0010-rework-reads-repository-directly.md) | `rework` streams the repository's patch history instead of reading a log file | Accepted |
| [0011](0011-rework-edit-similarity-heuristic.md) | Rework's edit-vs-rewrite test: token Dice at 0.6 with a punctuation fallback | Accepted |
| [0012](0012-knowledge-loss-input-and-naming.md) | `knowledge-loss` takes a `--former-authors` list; team maps stay strict | Accepted |
| [0013](0013-semver-and-changelog.md) | SemVer from v1.0.0, and a curated, CI-enforced CHANGELOG | Accepted |
| [0014](0014-small-stacked-prs.md) | Keep PRs under about 400 lines of runtime code, splitting with `gh stack` | Accepted |
| [0015](0015-ci-signals-inform-rather-than-gate.md) | Noisy CI signals inform rather than gate | Accepted |
| [0016](0016-ci-supply-chain-hardening.md) | Pin, delay, and least-privilege the CI supply chain | Accepted |
