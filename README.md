# gomaat

A Go port of [code-maat](https://github.com/adamtornhill/code-maat) — a command-line tool that mines git history to surface design insights. Identify logical coupling, code churn, authorship patterns, fragmentation, and more.

Inspired by the books [*Your Code as a Crime Scene*](https://pragprog.com/titles/atcrime2/your-code-as-a-crime-scene-second-edition/) and [*Software Design X-Rays*](https://pragprog.com/titles/atevol/software-design-x-rays/) by Adam Tornhill.

---

## Table of Contents

- [Installation](#installation)
- [Workflow](#workflow)
- [Generating a Git Log](#generating-a-git-log)
- [Global Flags](#global-flags)
- [Analyses](#analyses)
  - [authors](#authors)
  - [revisions](#revisions)
  - [coupling](#coupling)
  - [soc](#soc-sum-of-coupling)
  - [summary](#summary)
  - [abs-churn](#abs-churn)
  - [author-churn](#author-churn)
  - [entity-churn](#entity-churn)
  - [entity-ownership](#entity-ownership)
  - [main-dev](#main-dev)
  - [refactoring-main-dev](#refactoring-main-dev)
  - [entity-effort](#entity-effort)
  - [main-dev-by-revs](#main-dev-by-revs)
  - [fragmentation](#fragmentation)
  - [communication](#communication)
  - [age](#age)
  - [identity](#identity)
- [Advanced Usage](#advanced-usage)
  - [Architectural Grouping](#architectural-grouping)
  - [Team Mapping](#team-mapping)
  - [Limiting Output Rows](#limiting-output-rows)
  - [Writing to a File](#writing-to-a-file)

---

## Installation

**Requirements:** Go 1.21 or later, `git` on your `PATH`.

```bash
git clone <repo-url>
cd gomaat
go install ./cmd/gomaat/
```

Or build a binary directly:

```bash
go build -o gomaat ./cmd/gomaat/
```

---

## Workflow

1. **Generate a log** from your git repository.
2. **Run an analysis** against that log file.

```bash
# Step 1 — generate the log
gomaat generate-log --after 2023-01-01 --output logfile.log

# Step 2 — run an analysis
gomaat coupling -l logfile.log
```

All analyses read from a pre-generated log file rather than calling git directly. This makes repeated analysis fast and allows you to version-control the log for reproducible results.

---

## Generating a Git Log

The `generate-log` subcommand runs the correct `git log` invocation so you don't have to remember the flags.

```
gomaat generate-log [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--after` | _(all history)_ | Only include commits after this date (`YYYY-MM-DD`) |
| `--repo` | `.` | Path to the git repository |
| `--output` | stdout | Write the log to this file |

**Examples:**

```bash
# Current directory, all history, print to stdout
gomaat generate-log

# Last two years, save to file
gomaat generate-log --after 2023-01-01 --output logfile.log

# Different repo
gomaat generate-log --repo /path/to/project --after 2022-06-01 --output logfile.log
```

The log is generated using:
```
git log --all --numstat --date=short --pretty=format:'--%h--%ad--%aN' --no-renames [--after=DATE]
```

> **Note:** `--no-renames` means renamed files are tracked as a delete + add rather than a rename. This avoids inflated coupling between old and new paths.

---

## Global Flags

These flags are available on every analysis subcommand.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--log` | `-l` | _(required)_ | Path to the git log file |
| `--outfile` | `-o` | stdout | Write CSV output to this file |
| `--rows` | `-r` | 0 (no limit) | Maximum number of result rows |
| `--group` | `-g` | _(none)_ | [Architectural grouping](#architectural-grouping) spec file |
| `--team-map-file` | `-p` | _(none)_ | [Team mapping](#team-mapping) CSV file |

---

## Analyses

All analyses write CSV to stdout by default. Use `-o <file>` to write to a file instead.

---

### authors

Count the number of distinct authors and total revisions per entity. Entities with many authors have a higher communication overhead and tend to accumulate more defects.

```
gomaat authors -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `n-authors` | Number of distinct authors |
| `n-revs` | Total revisions |

Sorted by `n-authors` descending.

```
entity,n-authors,n-revs
src/core/Engine.java,8,42
src/api/Router.java,5,18
src/util/Parser.java,1,3
```

---

### revisions

Count the total number of revisions per entity. Frequently changed files are higher-risk and worth prioritizing for quality improvements.

```
gomaat revisions -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `n-revs` | Total revisions |

Sorted by `n-revs` descending.

---

### coupling

Detect temporal (logical) coupling — modules that change together more often than chance. Coupling that isn't visible in the code is often a sign of hidden dependencies or misplaced responsibilities.

```
gomaat coupling -l logfile.log [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--min-revs` | `-n` | `5` | Minimum revisions for an entity to be included |
| `--min-shared-revs` | `-m` | `5` | Minimum number of shared revisions between a pair |
| `--min-coupling` | `-i` | `30` | Minimum coupling percentage to report |
| `--max-coupling` | `-x` | `100` | Maximum coupling percentage to report |
| `--max-changeset-size` | `-s` | `30` | Ignore commits that touch more than this many files (large refactors skew results) |
| `--verbose-results` | | `false` | Add extra columns: per-entity revision counts and shared revision count |

**Coupling formula:**
```
degree = (shared_revisions / average_revisions(A, B)) × 100
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | First file |
| `coupled` | Second file |
| `degree` | Coupling percentage |
| `average-revs` | Average revision count across both files |

With `--verbose-results`, three extra columns are appended: `first-entity-revisions`, `second-entity-revisions`, `shared-revisions`.

Sorted by `degree` descending.

```
entity,coupled,degree,average-revs
src/Order.java,src/Invoice.java,82,44
src/User.java,src/Auth.java,61,38
```

**Tip:** Start with looser thresholds (`-n 2 -m 2 -i 10`) to see the full picture, then tighten them to focus on the strongest signals.

---

### soc (Sum of Coupling)

Aggregate the total coupling for each entity — how many co-change relationships it participates in across all revisions. High SOC entities are "hubs" that everything depends on.

```
gomaat soc -l logfile.log [flags]
```

Accepts the same threshold flags as [coupling](#coupling).

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `soc` | Sum of coupling (total co-change count) |

Sorted by `soc` descending.

---

### summary

Print a quick overview of the dataset: commit count, entity count, and author count.

```
gomaat summary -l logfile.log
```

**Output:**

```
statistic,value
number-of-commits,1432
number-of-entities,318
number-of-entities-changed,8741
number-of-authors,24
```

---

### abs-churn

Absolute code churn aggregated by date — total lines added and deleted per day. Useful for identifying turbulent periods in development history.

```
gomaat abs-churn -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `date` | Commit date (`YYYY-MM-DD`) |
| `added` | Lines added |
| `deleted` | Lines deleted |
| `commits` | Number of commits on that date |

Sorted by `date` ascending.

---

### author-churn

Lines added and deleted aggregated by author. Shows individual contribution volume.

```
gomaat author-churn -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `author` | Author name |
| `added` | Total lines added |
| `deleted` | Total lines deleted |
| `commits` | Total commits |

Sorted by `author` ascending.

---

### entity-churn

Lines added and deleted aggregated by entity. Pre-release churn is one of the strongest predictors of post-release defects.

```
gomaat entity-churn -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `added` | Total lines added |
| `deleted` | Total lines deleted |
| `commits` | Total commits |

Sorted by `added` descending.

---

### entity-ownership

Churn broken down by (entity, author) pair. Shows exactly how much each author contributed to each file in terms of lines written.

```
gomaat entity-ownership -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `author` | Author name |
| `added` | Lines added by this author |
| `deleted` | Lines deleted by this author |

Sorted by `entity` ascending.

---

### main-dev

Identify the main developer per entity — the author responsible for the most lines added. Combined with `entity-churn`, this tells you who to talk to about a problematic file.

```
gomaat main-dev -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `main-dev` | Author with the most lines added |
| `added` | Lines added by main developer |
| `total-added` | Total lines added to this entity |
| `ownership` | Main developer's share (%) |

Sorted by `entity` ascending.

---

### refactoring-main-dev

Like `main-dev`, but ranks by lines deleted rather than added. Line deletions are a proxy for design decisions — the author who deletes the most often has the deepest understanding of the code.

```
gomaat refactoring-main-dev -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `main-dev` | Author with the most lines deleted |
| `removed` | Lines deleted by main developer |
| `total-removed` | Total lines deleted from this entity |
| `ownership` | Main developer's share (%) |

Sorted by `entity` ascending.

---

### entity-effort

Revision count per (entity, author) pair. Useful for understanding knowledge distribution without relying on line counts (which can be misleading for reformatted files).

```
gomaat entity-effort -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `author` | Author name |
| `author-revs` | Revisions by this author |
| `total-revs` | Total revisions to this entity |

Sorted by `entity` ascending, then `author-revs` descending within each entity.

---

### main-dev-by-revs

Main developer per entity ranked by revision count rather than lines. Revision count is a more stable signal than line count for files that are frequently reformatted or auto-generated.

```
gomaat main-dev-by-revs -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `main-dev` | Author with the most revisions |
| `added` | Revisions by main developer |
| `total-added` | Total revisions to this entity |
| `ownership` | Main developer's share (%) |

Sorted by `entity` ascending.

---

### fragmentation

The fractal value (0–1) measuring how evenly development effort is distributed across authors for each entity.

- `0.00` — single author owns this entity entirely
- Approaching `1.00` — many authors contribute equally

Highly fragmented entities have diffuse ownership and are harder to reason about.

```
gomaat fragmentation -l logfile.log
```

**Fragmentation formula:**
```
fractal = 1 - Σ(author_revisions / total_revisions)²
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `fractal-value` | Fragmentation score (0.00–1.00) |
| `total-revs` | Total revisions to this entity |

Sorted by `fractal-value` descending.

---

### communication

Map communication needs across the team. Author pairs who frequently modify the same entities need to coordinate — this analysis makes that implicit need explicit. Based on Conway's Law.

```
gomaat communication -l logfile.log
```

**Strength formula:**
```
strength = (shared_entities / ceil(avg(entities_A, entities_B))) × 100
```
where `shared_entities` is the count of entities both authors have touched.

**Output:**

| Column | Description |
|--------|-------------|
| `author` | First author |
| `peer` | Second author |
| `shared` | Number of entities both have touched |
| `average` | `ceil((total_entities_A + total_entities_B) / 2)` |
| `strength` | Communication need as a percentage |

Sorted by `strength` descending. Each pair appears twice (once per direction).

---

### age

Months since each entity was last modified, relative to a reference date. Old, untouched code is either stable or forgotten — either way it's worth knowing about.

```
gomaat age -l logfile.log [--age-time-now YYYY-MM-DD]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--age-time-now` | `-d` | today | Reference date for age calculation |

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `age-months` | Months since last modification |

Sorted by `age-months` ascending (youngest first).

---

### identity

Dump the raw parsed commit records as CSV. Useful for debugging the log format or inspecting what gomaat sees before analysis.

```
gomaat identity -l logfile.log
```

**Output:**

| Column | Description |
|--------|-------------|
| `entity` | File path |
| `rev` | Commit hash |
| `date` | Commit date |
| `author` | Author name |
| `loc-added` | Lines added (0 for binary files) |
| `loc-deleted` | Lines deleted (0 for binary files) |

---

## Advanced Usage

### Architectural Grouping

Map file paths to named architectural boundaries before running analysis. Results are aggregated at the component level rather than the file level — useful for large codebases where file-level coupling is too noisy.

Create a grouping spec file (one rule per line):

```
# Lines starting with # are ignored
src/orders    => Orders
src/payments  => Payments
src/users     => Users
^src/shared/  => Shared
```

Plain paths are matched as prefixes (`src/orders/` matches `src/orders/Model.java`).
Lines starting with `^` are treated as regular expressions.

```bash
gomaat coupling -l logfile.log -g groups.txt
```

Files that don't match any group are excluded from the analysis.

---

### Team Mapping

Replace individual author names with team names before running analysis. Social metrics like `communication` and `fragmentation` then operate at the team level.

Create a CSV file mapping authors to teams:

```csv
author,team
Alice Smith,Backend
Bob Jones,Backend
Carol White,Frontend
Dave Brown,Platform
```

The header row is optional and automatically skipped.

```bash
gomaat communication -l logfile.log -p teams.csv
gomaat fragmentation -l logfile.log -p teams.csv
```

Authors not present in the map are excluded from analysis.

---

### Limiting Output Rows

Use `-r` to cap the number of result rows. Useful when piping to other tools or when you only care about the top N results.

```bash
# Top 10 most coupled module pairs
gomaat coupling -l logfile.log -n 2 -m 2 -r 10

# Top 5 most revised files
gomaat revisions -l logfile.log -r 5
```

---

### Writing to a File

Use `-o` to write CSV output to a file instead of stdout.

```bash
gomaat authors -l logfile.log -o authors.csv
gomaat coupling -l logfile.log -o coupling.csv
```

---

## Example End-to-End Session

```bash
# 1. Generate a log for the last year
gomaat generate-log \
  --repo /path/to/your/project \
  --after 2024-01-01 \
  --output project.log

# 2. Overview: how big is the dataset?
gomaat summary -l project.log

# 3. Which files change the most?
gomaat revisions -l project.log -r 20

# 4. Which files have the most authors?
gomaat authors -l project.log -r 20

# 5. Are there hidden dependencies?
gomaat coupling -l project.log --min-revs 10 --min-shared-revs 5

# 6. Where is knowledge fragmented?
gomaat fragmentation -l project.log -r 20

# 7. Who should talk to whom?
gomaat communication -l project.log

# 8. What code has gone untouched for years?
gomaat age -l project.log -r 20
```
