# Recipe: choose a refactor and measure whether it helped

Refactoring time is limited, and "this file is a mess" doesn't say which mess is worth it. This recipe uses history to pick a target where the effort pays off, writes down what should improve, and then checks with the same numbers whether it did.

It needs `bash`, `git`, `jq`, GNU `date`, and gomaat. On macOS, install GNU `date` with `brew install coreutils` and change `date` to `gdate` in the script.

- [1. Find candidates](#1-find-candidates)
- [2. Look closer at one](#2-look-closer-at-one)
- [3. Write down the expected result](#3-write-down-the-expected-result)
- [4. Measure before and after](#4-measure-before-and-after)
- [Gotchas](#gotchas)

## 1. Find candidates

A refactor pays off where people change the code often *and* where changing it is hard. Code nobody touches can stay messy at no cost. Generate a log of the last year and look at four views:

```bash
since=$(date -u -d '12 months ago' +%F)
gomaat generate-log --after "$since" -o year.log

# What changes most often
gomaat revisions -l year.log -r 20
# What's biggest (--by-file sorts by path, so sort by the Code column)
gomaat cloc --by-file --exclude vendor/ | sort -t, -k5 -nr | head -20
# What changes along with the most other files
gomaat soc -l year.log -r 20
# What keeps being rewritten
gomaat rework --after "$since" -r 20
```

The strongest candidates show up in more than one list. A large file near the top of `revisions` is a *hotspot*; the [monthly health check](monthly-health-check.md) computes the same revisions × lines of code score and keeps a list over time. High `soc` adds that changing it tends to drag other files along. High `rework` adds that changes to it often aren't right the first time.

Pick one. Starting with one gets a result you can learn from faster than three refactors going at once.

## 2. Look closer at one

Before changing anything, find out *why* it changes so often. Two things help:

```bash
# Which files change along with it, and how often
gomaat coupling -l year.log -n 3 -m 3 -i 20 --verbose-results | grep 'src/billing/invoice.go'

# Who's been working on it, to ask
gomaat whoknows -l year.log src/billing/invoice.go
```

Then read the last 20 or so commits to it with `git log --oneline -- <file>` and sort them roughly into features, bug fixes, and changes made only because something else changed. The mix suggests the refactor:

| What you see | Likely problem | Refactor to try |
|---|---|---|
| Many unrelated features touch it | It does too many things | Split it by responsibility |
| It always changes with one other file | A concept is split across two places | Move the code together, or extract what they share |
| Lots of bug fixes and high rework | It's hard to change correctly | Add tests around it first, then simplify |
| Changes with many files across the repo (high `soc`) | Everything depends on its details | Put a narrower interface in front of it |

## 3. Write down the expected result

Before starting, write one sentence in the ticket or PR:

> We believe splitting `invoice.go` by responsibility will mean fewer changes to billing also touch other code, measured by the share of billing commits that change files outside it, over the four months after the merge compared with the four months before.

Writing it down first keeps the measurement honest: if you pick the metric after seeing the results, you'll find one that improved. Name the metric that matches the problem from step 2. Not every refactor should change every number.

## 4. Measure before and after

Wait until the refactor has been merged for as long as the window you chose; three to six months is usually enough for a busy area. Then run this script to compare the same length of time before and after the merge.

Save it as `compare.sh`:

```bash
#!/usr/bin/env bash
# Compare an area of the code before and after a refactor.
#
# usage: compare.sh <merge date> <months> <path> [<path>...]
#
# Measures the area (every given path, old and new) over <months> before the
# refactor merged and the same length of time after, and prints a Markdown
# table. Run it once the "after" window has fully passed.
set -euo pipefail

merged=$1
months=$2
shift 2
paths=("$@")

excludes=(--exclude vendor/ --exclude node_modules/ --exclude dist/
  --exclude '*.lock' --exclude package-lock.json --exclude '*.min.js')
authors=(--exclude-author 'dependabot[bot]' --exclude-author 'renovate*')
[ -f .git-blame-ignore-revs ] && authors+=(--ignore-revs-file .git-blame-ignore-revs)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# One group for the area, one for everything else, so files the refactor
# moved, split or renamed still count as the same area. Each path becomes a
# regex because in gomaat 1.1.0 and earlier a plain path in a grouping spec
# only matches a directory, not a file.
{
  printf '%s\n' "${paths[@]%/}" | sed 's/[][\.*^$+?(){}|]/\\&/g; s/.*/^&(\/|$) => Area/'
  echo '^. => Rest'
} > "$tmp/groups.txt"

# measure <start> <end> prints one JSON object of metrics for that window.
measure() {
  local log="$tmp/$1.log"
  gomaat generate-log --after "$1" --before "$2" "${excludes[@]}" "${authors[@]}" \
    --outfile "$log" 2>/dev/null
  jq -n \
    --slurpfile summary <(gomaat summary -l "$log" -f json) \
    --slurpfile churn <(gomaat entity-churn -l "$log" -g "$tmp/groups.txt" -f json) \
    --slurpfile coupling <(gomaat coupling -l "$log" -g "$tmp/groups.txt" \
      -n 1 -m 1 -i 0 --verbose-results -f json) \
    --slurpfile rework <(gomaat rework --after "$1" --before "$2" "${excludes[@]}" \
      -f json -- "${paths[@]}") '
    ($churn[0][] | select(.entity == "Area")) as $area
    | ($coupling[0][0] // {}) as $pair
    | ($area.commits | tonumber) as $commits
    | ($rework[0] | map(.["added-lines"] | tonumber) | add // 0) as $added
    | {
        "Commits to the area": $commits,
        "Lines added + deleted": (($area.added | tonumber) + ($area.deleted | tonumber)),
        "Area commits that also changed other code (%)":
          (if $commits == 0 then 0
           else ($pair["shared-revisions"] // "0" | tonumber) * 100 / $commits | round end),
        "Rework ratio (%)":
          (if $added == 0 then 0
           else ($rework[0] | map(.["reworked-lines"] | tonumber) | add) * 100 / $added | round end),
        "Commits to the whole repo": ($summary[0][] | select(.statistic == "number-of-commits") | .value | tonumber)
      }'
}

start=$(date -u -d "$merged - $months months" +%F)
end=$(date -u -d "$merged + $months months" +%F)

jq -r -n --argjson before "$(measure "$start" "$merged")" \
  --argjson after "$(measure "$merged" "$end")" --arg merged "$merged" --arg months "$months" '
  "| \($months)-month windows around \($merged) | Before | After |",
  "|---|---|---|",
  ($before | keys_unsorted[] as $k | "| \($k) | \($before[$k]) | \($after[$k]) |")'
```

List every path in the area, before and after the refactor: if `invoice.go` was split into `invoice.go`, `pricing.go`, and `discounts.go`, pass all of them, or just the directory. The script groups them together, so the numbers stay comparable even though the files changed.

```bash
bash compare.sh 2026-03-01 4 src/billing/
```

Example output:

```
| 4-month windows around 2026-03-01 | Before | After |
|---|---|---|
| Commits to the area | 41 | 33 |
| Lines added + deleted | 2810 | 1904 |
| Area commits that also changed other code (%) | 63 | 38 |
| Rework ratio (%) | 18 | 9 |
| Commits to the whole repo | 410 | 395 |
```

How to read it:

- **Commits to the whole repo** is the control. If the whole repo slowed down by a third, a third fewer commits to the area isn't the refactor's doing. Compare the area's change to the repo's.
- **Area commits that also changed other code** is the main coupling measure: how often working in the area meant touching something else as well. A refactor that contained a responsibility should bring it down.
- **Rework ratio** is the share of lines added to the area that were deleted or rewritten within 14 days. It should drop if the code got easier to change correctly.
- **Fewer commits isn't always better.** If the area got easier to change, people may change it *more*. Read the commits and the other rows together.

Put the table in the ticket under the sentence from step 3 and say whether the prediction held. If it didn't, that's worth knowing too: it tells you something about the code the next refactor can use.

## Gotchas

- **Small numbers are noise.** With fewer than about 20 commits in each window, a difference of a few commits means nothing. Use a longer window, or pick a busier target.
- **Other things change too.** A product launch, a new team member, or a holiday period can move every number. The whole-repo row catches some of that; knowing what else happened catches the rest.
- **The refactor commit itself is in the "after" window.** A big refactor adds and deletes a lot of lines on its first day. If it dominates the *Lines added + deleted* row, start the "after" window a few days later.
- **Renames count as delete plus add.** gomaat deliberately [doesn't follow renames](../adr/0004-no-renames-no-merges-in-log.md). That's why the script groups old and new paths together instead of comparing files.
- **Mass reformats skew everything.** Add reformat commits to `.git-blame-ignore-revs`; the script skips the commits listed there.
