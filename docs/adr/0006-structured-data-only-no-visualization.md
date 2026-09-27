# 6. gomaat outputs structured data only; visualization and integrations live downstream

- **Status:** Accepted
- **Date:** 2026-09-10 (MCP scoping); 2026-09-25 (#69 closed)

## Context

Several backlog items would have turned the binary into a presentation layer or a long-running service:

- #69: `report --html`, a self-contained page with a treemap and coupling graph.
- #68 (original scope): Mermaid and DOT graph output alongside `md` and `table`.
- #70: an MCP server exposing hotspots, coupling, and the like as agent tools.

## Decision

gomaat produces rows: CSV or JSON, first row is the header. It doesn't render charts or graphs, and it doesn't run as a server.

- #69 closed as won't do: "gomaat's job is to produce structured data (CSV/JSON) that other tools consume. It shouldn't also be a visualization layer. Treemaps and coupling graphs are better built downstream from `--format json` output, in whatever tool the reader already uses, than maintained inside the binary."
- Mermaid and DOT were dropped from #68 for the same reason. #68 covers only the `table` and `md` text formats.
- #70 is scoped as a thin external wrapper that shells out to `gomaat … --format json`, not as a new run mode in the binary.

## Consequences

- One output contract: the `internal/output` writers over `[][]string`.
- No JS, template, or graph-layout dependencies in the binary.
- Visual reports are built outside gomaat. For example, D3 charts have been built from `--format json` output for one-off analyses of other repositories.
- New output formats need to be tabular text to fit here.

## Sources

- Issue #69 and its closing comment; issue #68 (rescoped); issue #70
- Backlog-planning session, 2026-09-10
