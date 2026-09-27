# 5. Stay a stateless text pipeline; no on-disk index or cache

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

A backlog proposal described `gomaat index`, a SQLite or columnar store under `.gomaat/` that is updated incrementally, as the prerequisite for hotspots, reports, trends, and any analysis that combines two metrics.

On review, that premise didn't hold. The README's "Tracking Metrics Over Time" recipe already combines outputs across logs with `jq`, and a `hotspots` command needs only a log file and `cloc` output. An index would add a stateful component, invalidation logic, and a migration story. The only thing it would save is parse time, and nobody had measured parse time as a problem.

Issue #48 was written as "benchmark first, then decide". The benchmark was never run.

## Decision

No index or cache. The maintainer closed #48 before the benchmark: "Holding off on this for now … Can reopen if/when parsing performance actually becomes a measured problem."

gomaat stays log in, CSV/JSON out. Combining metrics is done by composing outputs with standard tools or with a command that reads its inputs directly.

## Consequences

- No hidden state to invalidate, and every result can be reproduced from the log.
- Every run reparses the whole log. This hasn't been measured as a bottleneck.
- To reopen, bring a measurement. The #48 criteria still apply: benchmark `parser.ParseFile` on a 10k+ commit log. If a cache is justified, it must be opt-in and must never replace the log-file path.

## Sources

- Issue #48 and its closing comment
- Backlog-planning session, 2026-09-10 ("I want to hold off on the sqlite piece")
