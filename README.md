# gomaat

A Go port of [code-maat](https://github.com/adamtornhill/code-maat) — a command-line tool that mines git history to surface design insights. Identify logical coupling, code churn, authorship patterns, fragmentation, and more.

Inspired by the books [*Your Code as a Crime Scene*](https://pragprog.com/titles/atcrime2/your-code-as-a-crime-scene-second-edition/) and [*Software Design X-Rays*](https://pragprog.com/titles/atevol/software-design-x-rays/) by Adam Tornhill.

---

## Table of Contents

- [Installation](#installation)
  - [From a GitHub Release](#from-a-github-release)
  - [From Source](#from-source)
- [Workflow](#workflow)
- [Generating a Git Log](#generating-a-git-log)
- [Global Flags](#global-flags)
- [Analyses](#analyses)
  - [authors](#authors)
  - [revisions](#revisions)
  - [defects](#defects)
  - [coupling](#coupling)
  - [soc](#soc-sum-of-coupling)
  - [summary](#summary)
  - [statistics](#statistics)
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
- [Code Metrics](#code-metrics)
  - [cloc](#cloc)
- [Advanced Usage](#advanced-usage)
  - [Architectural Grouping](#architectural-grouping)
  - [Team Mapping](#team-mapping)
  - [Tracking Metrics Over Time](#tracking-metrics-over-time)
  - [Limiting Output Rows](#limiting-output-rows)
  - [Writing to a File](#writing-to-a-file)

---

## Installation

### From a GitHub Release

Prebuilt binaries for Linux, macOS, and Windows are published on the [Releases page](https://github.com/ethangardner/gomaat/releases/latest). Archives are named `gomaat_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) with a `checksums.txt` alongside them.

**Linux (amd64):**

```bash
curl -LO https://github.com/ethangardner/gomaat/releases/latest/download/gomaat_1.0.0_linux_amd64.tar.gz
tar -xzf gomaat_1.0.0_linux_amd64.tar.gz
sudo mv gomaat /usr/local/bin/
```

Use `gomaat_1.0.0_linux_arm64.tar.gz` on arm64 hosts. Replace `1.0.0` with the [latest version number](https://github.com/ethangardner/gomaat/releases/latest).

**macOS (Apple Silicon / Intel):**

```bash
curl -LO https://github.com/ethangardner/gomaat/releases/latest/download/gomaat_1.0.0_darwin_arm64.tar.gz
tar -xzf gomaat_1.0.0_darwin_arm64.tar.gz
sudo mv gomaat /usr/local/bin/
```

Use `gomaat_1.0.0_darwin_amd64.tar.gz` on Intel Macs. Replace `1.0.0` with the [latest version number](https://github.com/ethangardner/gomaat/releases/latest). The binary is unsigned, so the first run may require approving it via **System Settings → Privacy & Security** (or run `xattr -d com.apple.quarantine /usr/local/bin/gomaat`).

**Windows (amd64):**

1. Download `gomaat_1.0.0_windows_amd64.zip` from the [latest release](https://github.com/ethangardner/gomaat/releases/latest) (replace `1.0.0` with the current version).
2. Extract the archive and move `gomaat.exe` into a folder on your `PATH` (e.g. `C:\Program Files\gomaat\`).
3. Add that folder to `PATH` if it isn't already: **System Properties → Environment Variables → Path → New**.

Or via PowerShell:

```powershell
Invoke-WebRequest -Uri https://github.com/ethangardner/gomaat/releases/latest/download/gomaat_1.0.0_windows_amd64.zip -OutFile gomaat.zip
Expand-Archive gomaat.zip -DestinationPath .
Move-Item gomaat.exe "C:\Program Files\gomaat\gomaat.exe"
```

Verify any download against `checksums.txt` from the same release.

### From Source

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

Check the installed version with `gomaat --version`.

---

## Workflow

1. **Generate a log** from your git repository.
2. **Run an analysis** against that log file.

```bash
# Step 1 — generate the log
gomaat generate-log --after 2023-01-01 --outfile logfile.log

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

| Flag                 | Default         | Description                                                                                          |
|----------------------|-----------------|-------------------------------------------------------------------------------------------------------|
| `--after`            | _(all history)_ | Only include commits after this date (`YYYY-MM-DD`)                                                 |
| `--before`           | _(all history)_ | Only include commits before this date (`YYYY-MM-DD`)                                                |
| `--path`             | `.`             | Path to the git repository                                                                          |
| `--outfile`          | stdout          | Write the log to this file                                                                          |
| `--exclude`          | _(none)_        | Exclude paths matching this pattern (repeatable, supports globs)                                    |
| `--exclude-author`   | _(none)_        | Exclude commits by this author name (repeatable, supports `*` globs, case-sensitive, matches `%aN`) |
| `--ignore-revs-file` | _(none)_        | Drop commits listed in this file (one SHA per line, same format as `git blame --ignore-revs-file`)  |
| `--use-mailmap`      | `false`         | Resolve author identities via a `.mailmap` file at the repo root (native `git log --use-mailmap`)   |

**Examples:**

```bash
# Current directory, all history, print to stdout
gomaat generate-log

# Last two years, save to file
gomaat generate-log --after 2023-01-01 --outfile logfile.log

# Fixed historical window, e.g. Q1 2025
gomaat generate-log --after 2025-01-01 --before 2025-04-01 --outfile logfile.log

# Different repo
gomaat generate-log --path /path/to/project --after 2022-06-01 --outfile logfile.log

# Exclude generated files and vendored dependencies
gomaat generate-log --exclude vendor/ --exclude '*.pb.go' --outfile logfile.log

# Collapse the same human's commits from different machines/emails into one
# canonical author (requires a .mailmap file at the repo root)
gomaat generate-log --use-mailmap --outfile logfile.log

# Drop bot accounts so they don't skew churn/coupling/ownership metrics
gomaat generate-log --exclude-author "dependabot[bot]" --exclude-author "renovate*" --outfile logfile.log

# Drop a mass-reformat commit (and any other commits listed in the file) by SHA
gomaat generate-log --ignore-revs-file .git-blame-ignore-revs --outfile logfile.log
```

The log is generated using:
```
git log --all --numstat --date=short --pretty=format:'%x00%H%x00%ad%x00%aN%x00%B%x00' --no-renames --no-merges [--use-mailmap] [--after=DATE] [--before=DATE] [-- . :(exclude)PATTERN ...]
```

> **Note:** `--no-renames` means renamed files are tracked as a delete + add rather than a rename. This avoids inflated coupling between old and new paths.

> **Note:** `--no-merges` excludes merge commits, so a combined merge diff never gets double-counted against the commits it merges.

> **Note:** the log's rev field is the full commit hash (`%H`), not the abbreviated `%h` used in earlier versions, so it can be matched unambiguously against a `.git-blame-ignore-revs`-style file. `Rev` is treated as an opaque string everywhere it's consumed, so this only changes the value shown in output, not its meaning.

> **Note:** `--ignore-revs-file` is not a native `git log` concept (`--ignore-revs-file` is a `git blame` flag) — gomaat reads the file itself and drops matching commits from its own output after `git log` runs.

> **Note:** `generate-log`'s output is raw git log text (the format `internal/parser` reads), not CSV, so `--format` is a no-op here — `--format json` is rejected since there's no tabular data to convert.

> **Note:** each commit's full message (subject, body, and trailers such as `Co-Authored-By:`) is captured via `%B` and delimited with NUL bytes (`%x00`), since a commit message can legitimately contain blank lines or even a line that looks like a header from an older log format — NUL bytes are the one byte sequence git guarantees never appears inside a commit message. **This is a breaking change to the log format**: log files generated by gomaat versions prior to this one (single-line `--rev--date--author` headers) will no longer parse — regenerate them with the current `gomaat generate-log`. Running `identity` against an old-format file fails with an explicit error rather than silently producing an empty report.

---

## Global Flags

These flags are available on every analysis subcommand.

| Flag              | Short | Default      | Description                                                 |
|-------------------|-------|--------------|-------------------------------------------------------------|
| `--log`           | `-l`  | _(required)_ | Path to the git log file                                    |
| `--outfile`       | `-o`  | stdout       | Write output to this file                                   |
| `--rows`          | `-r`  | 0 (no limit) | Maximum number of result rows                               |
| `--group`         | `-g`  | _(none)_     | [Architectural grouping](#architectural-grouping) spec file |
| `--team-map-file` | `-p`  | _(none)_     | [Team mapping](#team-mapping) CSV file                      |
| `--format`        | `-f`  | `csv`        | Output format: `csv` or `json`                              |

---

## Analyses

All analyses write CSV to stdout by default. Use `-o <file>` to write to a file instead, or `--format json` (`-f json`) to write a JSON array of objects instead of CSV.

---

### authors

Count the number of distinct authors and total revisions per entity. Entities with many authors have a higher communication overhead and tend to accumulate more defects.

```
gomaat authors -l logfile.log
```

**Output:**

| Column      | Description                |
|-------------|----------------------------|
| `entity`    | File path                  |
| `n-authors` | Number of distinct authors |
| `n-revs`    | Total revisions            |

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

| Column   | Description     |
|----------|-----------------|
| `entity` | File path       |
| `n-revs` | Total revisions |

Sorted by `n-revs` descending.

---

### defects

Classify commits as bug fixes by matching their message against a pattern, then report defect density (bugfix revisions vs. total revisions) per entity. This distinguishes files that change constantly because they keep breaking from files that change constantly due to ongoing feature work — something churn metrics alone (`revisions`, `entity-churn`) can't tell apart. Requires commit messages, so the log must be generated by a version of `gomaat generate-log` that captures them.

```
gomaat defects -l logfile.log
gomaat defects -l logfile.log --bugfix-pattern '(?i)^fix'
gomaat defects -l logfile.log --conventional-commit-type fix
```

**Flags:**

| Flag                          | Default                        | Description                                                                          |
|-------------------------------|---------------------------------|---------------------------------------------------------------------------------------|
| `--bugfix-pattern`            | `` (?i)fix\|bug\|defect\|hotfix `` | Regex matched against each commit's full message. Invalid regex is a hard error.     |
| `--conventional-commit-type`  | (none)                          | Match a [Conventional Commits](https://www.conventionalcommits.org/) prefix instead, e.g. `fix` matches `fix:` and `fix(scope):`. Mutually exclusive with `--bugfix-pattern`. |

**Output:**

| Column              | Description                                  |
|---------------------|-----------------------------------------------|
| `entity`            | File path                                    |
| `bugfix-revisions`  | Revisions whose commit matched the pattern   |
| `total-revisions`   | Total revisions for the entity               |
| `defect-ratio`      | `bugfix-revisions / total-revisions`         |

Sorted by `defect-ratio` descending.

**Cross-referencing with `hotspots`:** a hotspot that's also defect-prone is a stronger signal than either metric alone — big, frequently-changed, *and* frequently broken. Join the two on `entity` with `jq`, following the same composability pattern as [Tracking Metrics Over Time](#tracking-metrics-over-time):

```bash
gomaat hotspots -l logfile.log --path . --format json > hotspots.json
gomaat defects -l logfile.log --format json > defects.json
jq -s '
  (.[0] | INDEX(.entity)) as $hot |
  .[1][] | . + {"hotspot-score": ($hot[.entity]["hotspot-score"] // null)}
' hotspots.json defects.json
```

---

### coupling

Detect temporal (logical) coupling — modules that change together more often than chance. Coupling that isn't visible in the code is often a sign of hidden dependencies or misplaced responsibilities.

```
gomaat coupling -l logfile.log [flags]
```

| Flag                   | Short | Default | Description                                                                        |
|------------------------|-------|---------|------------------------------------------------------------------------------------|
| `--min-revs`           | `-n`  | `5`     | Minimum revisions for an entity to be included                                     |
| `--min-shared-revs`    | `-m`  | `5`     | Minimum number of shared revisions between a pair                                  |
| `--min-coupling`       | `-i`  | `30`    | Minimum coupling percentage to report                                              |
| `--max-coupling`       | `-x`  | `100`   | Maximum coupling percentage to report                                              |
| `--max-changeset-size` | `-s`  | `30`    | Ignore commits that touch more than this many files (large refactors skew results) |
| `--verbose-results`    |       | `false` | Add extra columns: per-entity revision counts and shared revision count            |

**Coupling formula:**
```
degree = (shared_revisions / average_revisions(A, B)) × 100
```

**Output:**

| Column         | Description                              |
|----------------|------------------------------------------|
| `entity`       | First file                               |
| `coupled`      | Second file                              |
| `degree`       | Coupling percentage                      |
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

| Column   | Description                             |
|----------|-----------------------------------------|
| `entity` | File path                               |
| `soc`    | Sum of coupling (total co-change count) |

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

### statistics

Descriptive statistics (count, min, q1, median, q3, max, mean, sample-stddev) for five core metrics: files and lines changed per commit, and revisions/authors/sum-of-coupling per entity. Useful for spotting outliers and understanding the overall shape of a codebase's history at a glance.

```
gomaat statistics -l logfile.log
```

**Output:**

| Column   | Description                      |
|----------|-----------------------------------|
| `metric` | Which metric this row describes  |
| `count`  | Number of data points            |
| `min`    | Minimum value                    |
| `q1`     | First quartile                   |
| `median` | Median value                     |
| `q3`     | Third quartile                   |
| `max`    | Maximum value                    |
| `mean`   | Arithmetic mean                  |
| `stddev` | Sample standard deviation        |

`metric` is always the following five rows, in this order: `files-changed-per-commit`, `lines-changed-per-commit`, `revisions-per-entity`, `authors-per-entity`, `soc-per-entity`.

Unlike other analyses, `statistics` always reports on the whole dataset — it ignores coupling-style threshold flags (`--min-revs`, `--max-changeset-size`, etc.) since it's a codebase health report, not a filtered query.

```
metric,count,min,q1,median,q3,max,mean,stddev
files-changed-per-commit,48,1.00,1.00,2.00,5.00,28.00,4.73,6.06
lines-changed-per-commit,48,1.00,17.50,54.00,177.25,2244.00,194.75,399.44
revisions-per-entity,56,1.00,2.00,4.00,5.25,11.00,4.05,2.75
authors-per-entity,56,1.00,1.00,1.00,1.00,1.00,1.00,0.00
soc-per-entity,56,0.00,13.00,45.00,70.50,110.00,45.96,33.76
```

---

### abs-churn

Absolute code churn aggregated by date — total lines added and deleted per day. Useful for identifying turbulent periods in development history.

```
gomaat abs-churn -l logfile.log
```

**Output:**

| Column    | Description                    |
|-----------|--------------------------------|
| `date`    | Commit date (`YYYY-MM-DD`)     |
| `added`   | Lines added                    |
| `deleted` | Lines deleted                  |
| `commits` | Number of commits on that date |

Sorted by `date` ascending.

---

### author-churn

Lines added and deleted aggregated by author. Shows individual contribution volume.

```
gomaat author-churn -l logfile.log
```

**Output:**

| Column    | Description         |
|-----------|---------------------|
| `author`  | Author name         |
| `added`   | Total lines added   |
| `deleted` | Total lines deleted |
| `commits` | Total commits       |

Sorted by `author` ascending.

---

### entity-churn

Lines added and deleted aggregated by entity. Pre-release churn is one of the strongest predictors of post-release defects.

```
gomaat entity-churn -l logfile.log
```

**Output:**

| Column    | Description         |
|-----------|---------------------|
| `entity`  | File path           |
| `added`   | Total lines added   |
| `deleted` | Total lines deleted |
| `commits` | Total commits       |

Sorted by `added` descending.

---

### entity-ownership

Churn broken down by (entity, author) pair. Shows exactly how much each author contributed to each file in terms of lines written.

```
gomaat entity-ownership -l logfile.log
```

**Output:**

| Column    | Description                  |
|-----------|------------------------------|
| `entity`  | File path                    |
| `author`  | Author name                  |
| `added`   | Lines added by this author   |
| `deleted` | Lines deleted by this author |

Sorted by `entity` ascending.

---

### main-dev

Identify the main developer per entity — the author responsible for the most lines added. Combined with `entity-churn`, this tells you who to talk to about a problematic file.

```
gomaat main-dev -l logfile.log
```

**Output:**

| Column        | Description                      |
|---------------|----------------------------------|
| `entity`      | File path                        |
| `main-dev`    | Author with the most lines added |
| `added`       | Lines added by main developer    |
| `total-added` | Total lines added to this entity |
| `ownership`   | Main developer's share (%)       |

Sorted by `entity` ascending.

---

### refactoring-main-dev

Like `main-dev`, but ranks by lines deleted rather than added. Line deletions are a proxy for design decisions — the author who deletes the most often has the deepest understanding of the code.

```
gomaat refactoring-main-dev -l logfile.log
```

**Output:**

| Column          | Description                          |
|-----------------|--------------------------------------|
| `entity`        | File path                            |
| `main-dev`      | Author with the most lines deleted   |
| `removed`       | Lines deleted by main developer      |
| `total-removed` | Total lines deleted from this entity |
| `ownership`     | Main developer's share (%)           |

Sorted by `entity` ascending.

---

### entity-effort

Revision count per (entity, author) pair. Useful for understanding knowledge distribution without relying on line counts (which can be misleading for reformatted files).

```
gomaat entity-effort -l logfile.log
```

**Output:**

| Column        | Description                    |
|---------------|--------------------------------|
| `entity`      | File path                      |
| `author`      | Author name                    |
| `author-revs` | Revisions by this author       |
| `total-revs`  | Total revisions to this entity |

Sorted by `entity` ascending, then `author-revs` descending within each entity.

---

### main-dev-by-revs

Main developer per entity ranked by revision count rather than lines. Revision count is a more stable signal than line count for files that are frequently reformatted or auto-generated.

```
gomaat main-dev-by-revs -l logfile.log
```

**Output:**

| Column        | Description                    |
|---------------|--------------------------------|
| `entity`      | File path                      |
| `main-dev`    | Author with the most revisions |
| `added`       | Revisions by main developer    |
| `total-added` | Total revisions to this entity |
| `ownership`   | Main developer's share (%)     |

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

| Column          | Description                     |
|-----------------|---------------------------------|
| `entity`        | File path                       |
| `fractal-value` | Fragmentation score (0.00–1.00) |
| `total-revs`    | Total revisions to this entity  |

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

| Column     | Description                                       |
|------------|---------------------------------------------------|
| `author`   | First author                                      |
| `peer`     | Second author                                     |
| `shared`   | Number of entities both have touched              |
| `average`  | `ceil((total_entities_A + total_entities_B) / 2)` |
| `strength` | Communication need as a percentage                |

Sorted by `strength` descending. Each pair appears twice (once per direction).

---

### age

Months since each entity was last modified, relative to a reference date. Old, untouched code is either stable or forgotten — either way it's worth knowing about.

```
gomaat age -l logfile.log [--age-time-now YYYY-MM-DD]
```

| Flag             | Short | Default | Description                        |
|------------------|-------|---------|------------------------------------|
| `--age-time-now` | `-d`  | today   | Reference date for age calculation |

**Output:**

| Column       | Description                    |
|--------------|--------------------------------|
| `entity`     | File path                      |
| `age-months` | Months since last modification |

Sorted by `age-months` ascending (youngest first).

---

### identity

Dump the raw parsed commit records as CSV. Useful for debugging the log format or inspecting what gomaat sees before analysis.

```
gomaat identity -l logfile.log
```

**Output:**

| Column        | Description                                     |
|---------------|--------------------------------------------------|
| `entity`      | File path                                       |
| `rev`         | Commit hash                                     |
| `date`        | Commit date                                     |
| `author`      | Author name                                     |
| `message`     | Full commit message (subject, body, trailers)   |
| `loc-added`   | Lines added (0 for binary files)                |
| `loc-deleted` | Lines deleted (0 for binary files)              |

---

## Code Metrics

### cloc

Count lines of code, comments, and blank lines across a directory. Unlike the analysis commands, `cloc` works directly against source files and does not require a git log.

```
gomaat cloc [flags]
```

| Flag        | Default      | Description                                                      |
|-------------|--------------|------------------------------------------------------------------|
| `--path`    | `.`          | Directory to analyze                                             |
| `--by-file` | `false`      | Show results per file instead of per language                    |
| `--exclude` | _(none)_     | Exclude paths matching this pattern (repeatable, supports globs) |
| `--outfile` | stdout       | Write output to this file                                        |
| `--rows`    | 0 (no limit) | Maximum number of result rows                                    |
| `--format`  | `csv`        | Output format: `csv` or `json`                                   |

**Output (by language, default):**

| Column     | Description          |
|------------|----------------------|
| `Language` | Programming language |
| `Files`    | Number of files      |
| `Blank`    | Blank lines          |
| `Comment`  | Comment lines        |
| `Code`     | Lines of code        |

Sorted by `Code` descending. A `TOTAL` row is always appended.

```
Language,Files,Blank,Comment,Code
Go,42,410,180,3100
YAML,5,12,0,88
Makefile,1,8,4,22
TOTAL,48,430,184,3210
```

With `--by-file`, each row represents a single file sorted by path, with a `File` column replacing `Language` and a `Language` column added.

**Examples:**

```bash
# Count lines in the current directory
gomaat cloc

# Analyze a different repo
gomaat cloc --path /path/to/project

# Exclude vendored and generated files
gomaat cloc --exclude vendor/ --exclude '*.pb.go'

# Per-file breakdown saved to CSV
gomaat cloc --by-file -o cloc.csv

# Top 10 largest files by code
gomaat cloc --by-file -r 10

# Per-language breakdown as JSON
gomaat cloc --format json
```

The `--exclude` patterns follow the same rules as `generate-log --exclude`: patterns ending in `/` match directory prefixes; all others are matched as globs against both the full path and the base filename.

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

### Tracking Metrics Over Time

gomaat has no built-in trend command — every analysis operates on a single, static log file. To see how a metric changes over time (e.g. is `statistics`'s `soc-per-entity` getting worse release over release?), generate one log per period with [`generate-log --after/--before`](#generating-a-git-log), run the same analysis against each, tag the results with their period, and combine them yourself. This works for any analysis, but is most useful for whole-dataset reports like `statistics` and `summary`.

There are two ways to window each period, and they answer different questions:

- **Cumulative** — `--before <date>` only, no `--after`. Each log is "everything up to this point," so `statistics`/`summary` show how the codebase's overall shape has evolved to date. Best for metrics like `revisions-per-entity` or `soc-per-entity`, where you care about accumulated history.
- **Windowed** — `--after <start> --before <end>` per period. Each log contains only that slice's activity, so results reflect what happened *during* the period, not the whole history. Best for churn-style, period-over-period comparisons.

**Example — quarterly cumulative snapshots of `statistics`, combined into one JSON time series:**

```bash
#!/usr/bin/env bash
for cutoff in 2024-04-01 2024-07-01 2024-10-01 2025-01-01; do
  gomaat generate-log --before "$cutoff" --outfile "log-$cutoff.log"
  gomaat statistics -l "log-$cutoff.log" --format json \
    | jq --arg period "$cutoff" '[.[] | . + {period: $period}]'
done | jq -s 'add' > statistics-over-time.json
```

The result is a single JSON array with one object per metric per period (each carrying a `period` field), ready to load into a spreadsheet, filter with `jq` (e.g. `jq '[.[] | select(.metric=="soc-per-entity")]' statistics-over-time.json` to isolate one metric's trend), or feed into a plotting tool.

Prefer CSV? Prepend a period column with `awk` instead of using `--format json`:

```bash
echo "period,metric,count,min,q1,median,q3,max,mean,stddev" > statistics-over-time.csv
for cutoff in 2024-04-01 2024-07-01 2024-10-01 2025-01-01; do
  gomaat generate-log --before "$cutoff" --outfile "log-$cutoff.log"
  gomaat statistics -l "log-$cutoff.log" | tail -n +2 \
    | awk -v period="$cutoff" '{print period "," $0}' >> statistics-over-time.csv
done
```

This pattern combines well with [Architectural Grouping](#architectural-grouping) and [Team Mapping](#team-mapping) — add `-g`/`-p` to each per-period run to track one component's or team's metrics over time instead of the whole codebase.

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
# 1. Generate a log 
gomaat generate-log \
  --path /path/to/your/project \
  --after 2024-01-01 \
  --outfile logfile.log

# 2. Overview: how big is the dataset?
gomaat summary -l logfile.log

# 3. Which files change the most?
gomaat revisions -l logfile.log -r 20

# 4. Which files have the most authors?
gomaat authors -l logfile.log -r 20

# 5. Are there hidden dependencies?
gomaat coupling -l logfile.log --min-revs 10 --min-shared-revs 5

# 6. Where is knowledge fragmented?
gomaat fragmentation -l logfile.log -r 20

# 7. Who should talk to whom?
gomaat communication -l logfile.log

# 8. What code has gone untouched for years?
gomaat age -l logfile.log -r 20

# 9. Find the largest files
gomaat cloc --by-file | sort -t, -k5 -nr
```
