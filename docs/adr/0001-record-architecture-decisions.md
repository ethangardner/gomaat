# 1. Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-09-27

## Context

gomaat's design decisions are scattered across GitHub issue comments, closed PRs, commit bodies, AGENTS.md, and the README. Several of the most consequential ones are recorded only as a one-line comment on a closed issue ("Decided not to do this"), which a new contributor or agent will not find before re-proposing the same idea.

## Decision

Record architecturally significant decisions as Markdown files in `docs/adr/`, using Michael Nygard's format: Status, Context, Decision, Consequences. Each ADR ends with a Sources section that links the issues, PRs, and commits it was drawn from.

- Files are named `NNNN-short-title.md`, numbered sequentially, never renumbered.
- A decision to *not* build something is an ADR too, when the idea is likely to come back.
- An ADR is not edited to reverse its decision. Write a new ADR and mark the old one `Superseded by NNNN`.
- `docs/adr/README.md` is the index. Add a line when adding an ADR.
- Copy `template.md` to start a new one.

## Consequences

- Rejected ideas (a SQLite index, `--repo`, `--half-life`, HTML reports) have a discoverable record, so re-proposals start from the prior reasoning instead of from zero.
- ADRs 0002–0016 were written retroactively on 2026-09-27 from issue, PR, commit, and session history. Their dates are when the decision was made, not when the ADR was written. Where the original reasoning was never written down, the ADR says so rather than inventing one.
- One more place to keep current. The bar is "significant and likely to be questioned", not every choice.
