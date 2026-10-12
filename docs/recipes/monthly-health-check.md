# Recipe: a monthly code-health check in CI

This recipe runs gomaat on a schedule, compares the results with a baseline you've agreed on, and opens one issue when something new crosses a threshold. The team triages that issue into the backlog as part of regular maintenance.

It's written for GitHub Actions, but the script only needs `bash`, `git`, `jq`, GNU `date`, and gomaat. It runs on any CI system that can do a full clone on a schedule.

- [Why monthly, not on every commit](#why-monthly-not-on-every-commit)
- [What it reports](#what-it-reports)
- [Set it up](#set-it-up)
- [Set the thresholds](#set-the-thresholds)
- [Set the baseline](#set-the-baseline)
- [Act on the findings](#act-on-the-findings)
- [Gotchas](#gotchas)

## Why monthly, not on every commit

gomaat measures history. A single commit hardly moves a hotspot, coupling, or ownership number, so a per-commit run reports the same findings over and over. People learn to ignore a check like that.

- **Trends need time.** Coupling needs shared commits to build up before it means anything. Rework needs its window (14 days by default) to elapse before a line counts. A month is long enough for real movement and short enough to catch a problem before it sets in.
- **It isn't a gate.** These are prompts for a conversation, not pass/fail answers. "Is this file a hotspot?" has no right answer that should block a merge. gomaat's own CI follows the same rule: [noisy signals inform rather than gate](../adr/0015-ci-signals-inform-rather-than-gate.md).
- **It costs more than a lint.** `rework` streams the full patch history. It's seconds on most repos, but it's a poor fit for every push (see the [rework performance table](../../README.md#rework)).
- **It fits maintenance time.** A monthly issue lines up with a recurring maintenance or cleanup slot, so findings land in a backlog someone is already working through.

## What it reports

The script turns four analyses into *findings*. Each finding has a `kind`, a `key` (a file or a pair of files), and a `value` that the threshold applies to.

| Kind | Source | A finding when | Why it matters |
|---|---|---|---|
| `hotspot` | [`revisions`](../../README.md#revisions) × [`cloc --by-file`](../../README.md#cloc) | Among the top N files by revisions × lines of code | Large files that change often are where defects and slow changes concentrate. `value` is revisions × lines of code. |
| `coupling` | [`coupling`](../../README.md#coupling) | Two files change together at least X% of the time over at least M shared commits | A dependency the code doesn't show. Changing one without the other is a likely bug. `value` is the coupling degree. |
| `bus-factor` | [`bus-factor`](../../README.md#bus-factor) | One author wrote most of a file that changes often | Knowledge risk on code that's actively worked on. `value` is that author's share. |
| `rework` | [`rework`](../../README.md#rework) | At least X% of a file's added lines came back out within 14 days, over enough lines to matter | Code that keeps not being right the first time. `value` is the rework ratio. |

It also saves [`statistics`](../../README.md#statistics) and every raw analysis output as a build artifact, for digging in during triage.

Each run looks at the last 12 months only, so a hotspot that was fixed a year ago stops being reported on its own.

## Set it up

Add two files to your repository and, optionally, a label.

### 1. The report script

Save this as `.github/gomaat/report.sh`. The `excludes`, `authors`, and threshold variables near the top are the parts to edit for your repo.

```bash
#!/usr/bin/env bash
# Monthly gomaat code-health report.
#
# usage: report.sh <out dir> [baseline.json]
#
# Writes to <out dir>:
#   findings.json  every finding over a threshold below
#   new.json       findings not yet in the baseline
#   report.md      a summary for the job summary and the backlog issue
# plus the raw analysis output each finding came from.
set -euo pipefail

out=$1
baseline=${2:-.github/gomaat/baseline.json}

# Look-back window. Twelve months keeps a year-old, since-fixed hotspot from
# being reported forever, and is long enough for coupling to show up.
since=$(date -u -d '12 months ago' +%F)

# Paths that aren't hand-written. They dominate churn and rework otherwise.
excludes=(--exclude vendor/ --exclude node_modules/ --exclude dist/
  --exclude '*.lock' --exclude package-lock.json --exclude '*.min.js')
# Bots and mass reformats skew churn, coupling and ownership.
authors=(--exclude-author 'dependabot[bot]' --exclude-author 'renovate*')
[ -f .git-blame-ignore-revs ] && authors+=(--ignore-revs-file .git-blame-ignore-revs)

# Thresholds. Tune them on the first run so the report lists a handful of
# findings, not hundreds; see the recipe's "Set the thresholds" section.
hotspot_top=10        # report the N files with the highest revisions x lines of code
min_revs=10           # ignore files changed fewer times than this in the window
coupling_min=50       # coupling degree (%) ...
coupling_shared=10    # ... over at least this many shared commits
rework_min_ratio=20   # rework ratio (%) ...
rework_min_added=200  # ... over at least this many added lines

mkdir -p "$out"
log="$out/gomaat.log"
gomaat generate-log --after "$since" "${excludes[@]}" "${authors[@]}" --outfile "$log"

gomaat statistics -l "$log" -o "$out/statistics.csv"
gomaat revisions -l "$log" -f json -o "$out/revisions.json"
gomaat cloc --by-file "${excludes[@]}" -f json -o "$out/cloc.json"
gomaat coupling -l "$log" -n "$min_revs" -m "$coupling_shared" -i "$coupling_min" \
  -f json -o "$out/coupling.json"
gomaat bus-factor -l "$log" -f json -o "$out/bus-factor.json"
gomaat rework --after "$since" "${excludes[@]}" -f json -o "$out/rework.json"

# Every finding is {kind, key, value, detail}. kind+key identifies it across
# runs; value is the number the threshold applies to, so a baselined finding
# that keeps getting worse is still visible.
jq -n \
  --slurpfile revs "$out/revisions.json" \
  --slurpfile cloc "$out/cloc.json" \
  --slurpfile coupling "$out/coupling.json" \
  --slurpfile bus "$out/bus-factor.json" \
  --slurpfile rework "$out/rework.json" \
  --argjson top "$hotspot_top" --argjson min_revs "$min_revs" \
  --argjson min_ratio "$rework_min_ratio" --argjson min_added "$rework_min_added" '
  # JSON values are strings, as in the CSV output.
  ($cloc[0] | map({(.File): (.Code | tonumber)}) | add // {}) as $loc
  | ($revs[0] | map({(.entity): (.["n-revs"] | tonumber)}) | add // {}) as $nrevs
  # Files no longer in the tree have no line count, so they drop out here.
  | ( [ $nrevs | to_entries[]
      | select(.value >= $min_revs and $loc[.key] != null)
      | {kind: "hotspot", key: .key, value: (.value * $loc[.key]),
         detail: "\(.value) revisions, \($loc[.key]) lines of code"} ]
    | sort_by(-.value) | .[:$top]
  ) + [ $coupling[0][]
      | {kind: "coupling", key: "\(.entity) <-> \(.coupled)",
         value: (.degree | tonumber), detail: "\(.degree)% coupled"} ]
  + [ $bus[0][]
      | select(.["bus-factor"] == "1" and ($nrevs[.entity] // 0) >= $min_revs)
      | {kind: "bus-factor", key: .entity, value: (.ownership | tonumber),
         detail: "\(.["top-owners"]) wrote \(.ownership)%, \($nrevs[.entity]) revisions"} ]
  + [ $rework[0][]
      | select((.["rework-ratio"] | tonumber) >= $min_ratio
               and (.["added-lines"] | tonumber) >= $min_added)
      | {kind: "rework", key: .entity, value: (.["rework-ratio"] | tonumber),
         detail: "\(.["reworked-lines"]) of \(.["added-lines"]) added lines reworked"} ]
  ' > "$out/findings.json"

if [ ! -f "$baseline" ]; then
  baseline="$out/empty-baseline.json"
  echo '[]' > "$baseline"
fi

jq --slurpfile base "$baseline" '
  ($base[0] | map({("\(.kind)|\(.key)"): true}) | add // {}) as $known
  | map(select($known["\(.kind)|\(.key)"] | not))
  ' "$out/findings.json" > "$out/new.json"

jq -r -n --slurpfile found "$out/findings.json" --slurpfile new "$out/new.json" \
  --slurpfile base "$baseline" --arg since "$since" '
  def table: "| Kind | Where | Detail |", "|---|---|---|",
    (.[] | "| \(.kind) | `\(.key)` | \(.detail) |");
  ($found[0] | map({("\(.kind)|\(.key)"): .}) | add // {}) as $now
  | ($base[0] | map(. + {now: $now["\(.kind)|\(.key)"]})) as $known
  | "Commits since \($since).",
    "",
    "## New findings (\($new[0] | length))",
    "",
    if ($new[0] | length) == 0 then "None." else ($new[0] | table) end,
    "",
    "## Baselined findings",
    "",
    if ($known | length) == 0 then "None." else
      ("| Kind | Where | Baseline | Now | Reason |", "|---|---|---|---|---|",
       ($known[] | "| \(.kind) | `\(.key)` | \(.value) | \(.now.value // "resolved") | \(.reason) |"))
    end
  ' > "$out/report.md"
```

Run it locally first, from the root of your repo, to see what it finds:

```bash
bash .github/gomaat/report.sh /tmp/gomaat-report
cat /tmp/gomaat-report/report.md
```

### 2. The workflow

Save this as `.github/workflows/gomaat.yml`. It runs at 06:17 UTC on the 1st of each month and can also be started by hand from the Actions tab.

```yaml
name: gomaat health check

on:
  schedule:
    - cron: "17 6 1 * *"
  workflow_dispatch:

permissions: {}

jobs:
  report:
    runs-on: ubuntu-latest
    timeout-minutes: 30
    permissions:
      contents: read
      issues: write # open the backlog issue
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0 # gomaat reads history; a shallow clone has almost none
          persist-credentials: false

      - name: Install gomaat
        env:
          GOMAAT_VERSION: 1.1.0
        run: |
          base="https://github.com/ethangardner/gomaat/releases/download/v${GOMAAT_VERSION}"
          archive="gomaat_${GOMAAT_VERSION}_linux_amd64.tar.gz"
          cd "$RUNNER_TEMP"
          curl -fsSL --remote-name-all "$base/$archive" "$base/checksums.txt"
          sha256sum --check --ignore-missing checksums.txt
          tar -xzf "$archive" gomaat
          echo "$RUNNER_TEMP" >> "$GITHUB_PATH"

      - name: Run gomaat
        run: |
          bash .github/gomaat/report.sh "$RUNNER_TEMP/gomaat-report"
          cat "$RUNNER_TEMP/gomaat-report/report.md" >> "$GITHUB_STEP_SUMMARY"

      - uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: gomaat-report
          path: ${{ runner.temp }}/gomaat-report
          retention-days: 90

      - name: Open a backlog issue for new findings
        env:
          GH_TOKEN: ${{ github.token }}
          GH_REPO: ${{ github.repository }}
          REPORT: ${{ runner.temp }}/gomaat-report
        run: |
          [ "$(jq length "$REPORT/new.json")" -gt 0 ] || exit 0
          gh issue create --label code-health --body-file "$REPORT/report.md" \
            --title "Code health: new gomaat findings for $(date -u +%Y-%m)"
```

Pin `GOMAAT_VERSION` and bump it on purpose. A new release can change what an analysis reports, and you want that to arrive as a reviewed change, not a surprise on the 1st. The action SHAs are the ones gomaat itself pins; let Dependabot keep them current.

### 3. The label

`gh issue create --label` fails if the label doesn't exist. Create it once:

```bash
gh label create code-health --description "Monthly gomaat findings"
```

Or drop `--label code-health` from the workflow.

## Set the thresholds

The defaults in the script are a starting point, not a standard. There's no universal "good" coupling degree or rework ratio; it depends on the language, the team, and the size of the repo. Set them from your own data:

1. **Run the workflow by hand** (Actions → gomaat health check → Run workflow) or run the script locally.
2. **Look at the distribution.** `statistics.csv` in the artifact has the quartiles for revisions, authors, and coupling per file. A reasonable `min_revs` is around the `q3` of `revisions-per-entity`: it limits findings to the most-changed quarter of files.
3. **Aim for a short list.** Tighten thresholds until the first report has roughly ten findings or fewer. A list of 200 doesn't get triaged; a list of eight does. You can loosen them later once the backlog is under control.
4. **Commit the thresholds** with a short note on why you picked them, so the next person can tell a deliberate choice from a default.

Change one threshold at a time and rerun, so you can see which change moved which findings.

## Set the baseline

The first run reports everything that's over a threshold, including problems everyone already knows about. The baseline is the list of those known findings, so later runs only raise what's *new*.

Create it from the first run's `findings.json` (in the artifact, or your local run), then edit it down:

```bash
jq --arg reason "accepted at baseline, $(date -u +%Y-%m)" \
  '[.[] | {kind, key, value, reason: $reason}]' \
  findings.json > .github/gomaat/baseline.json
```

Each entry looks like this:

```json
{
  "kind": "coupling",
  "key": "src/billing/invoice.go <-> src/billing/tax.go",
  "value": 72,
  "reason": "Tax rules live in invoice.go for now; split tracked in #412"
}
```

Before committing it:

- **Remove anything that's actually noise** and fix the cause instead. A generated file belongs in `excludes`, a bot in `authors`, and one person under two names in a [`.mailmap`](../../README.md#generating-a-git-log). Those shouldn't be baselined, because they skew every other analysis too.
- **Replace the generic reason** with a real one where you can: a ticket number, "test and implementation, expected to change together", "owned by the payments team on purpose".
- **Review it in a PR** like any other change. The baseline is a record of what the team agreed to live with.

Every report lists baselined findings with their baseline value next to the current value, or `resolved` once a finding drops below its threshold. That's how you notice a known problem getting worse, or get to delete one that's gone.

## Act on the findings

When the issue opens, triage it in your next maintenance slot. Every new finding gets one of three outcomes:

| Outcome | When | What to do |
|---|---|---|
| **Fix** | It's a real problem worth time | Open a backlog item, link it from the issue, and add the finding to the baseline with the item as the reason. When the fix lands, the finding resolves and you remove it from the baseline. |
| **Accept** | It's real but expected, or not worth fixing now | Add it to the baseline with the reason. |
| **Tune** | It's noise | Add an exclude, a bot, or a `.mailmap` entry, or adjust a threshold. |

Close the issue once every finding has an outcome. The baseline PR is the record of the decisions. A finding that's left untriaged stays out of the baseline, so it's raised again next month.

Pick only one or two fix items per month to work on. The point is steady improvement during maintenance, not a cleanup project.

### What each kind tends to mean

A finding is a question to investigate, not a verdict. Read a file's findings together, then look at its commits.

- **hotspot**: Start here; it's where effort pays off most. Look at *why* it changes so often. One file handling many concerns is a candidate to split. Frequent bug fixes point to missing tests. A long, central file that everyone touches is the classic target for a careful refactor, one piece at a time. Documentation and config files show up too; usually that's fine, so accept them.
- **coupling**: A file and its test changing together is expected; accept it. Two production files in different modules changing together usually means a responsibility in the wrong place or a shared concept with no home. Consider moving the code closer together or extracting the shared piece. Run `coupling` with `-g` ([architectural grouping](../../README.md#architectural-grouping)) to see whether the coupling crosses module boundaries.
- **bus-factor**: A file one person wrote that changes often is a knowledge risk. Pair on the next change to it, rotate reviews, or write down the parts only that person knows. In a single-maintainer repo every file is bus factor 1; delete the `bus-factor` block from the script there.
- **rework**: Code that comes back out within two weeks. Read the commits: review churn, a bug found right after merge, unclear requirements, or an approach that was tried and replaced. Clearer requirements and tests up front often help more than changing the code. A ratio near 100% usually means a generated or committed-by-mistake file, which belongs in `excludes`.

Overlap is the strongest signal. A hotspot that also has high rework and a bus factor of 1 is worth more of your time than three unrelated findings.

## Gotchas

- **Scheduled workflows stop in quiet public repos.** GitHub disables `schedule` triggers in a public repo after 60 days without activity. Re-enable it from the Actions tab.
- **Schedules run on the default branch only.** Changes to the script or workflow on another branch don't take effect until they're merged.
- **Shallow clones hide history.** Without `fetch-depth: 0` there are almost no commits to analyze, and `rework` can't tell when lines landed.
- **Trends don't need saved artifacts.** Artifacts expire, but your git history doesn't. To chart a metric over the past year, regenerate it per period from history with the [tracking metrics over time](../../README.md#tracking-metrics-over-time) recipe instead of keeping old reports.
- **Bots and duplicate identities.** Look at the first report's author names. Anything like `github-actions[bot]`, or one person as both `jdoe` and `Jane Doe`, needs an `authors` exclude or a `.mailmap` entry before you baseline.
