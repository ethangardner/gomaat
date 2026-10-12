# Recipe: a handover plan when someone leaves

When someone gives notice, there are a few weeks to get what they know into other people's heads. This recipe uses history to find where that matters most: the files they wrote most of that are still changing, and who, if anyone, has worked on each one recently.

It needs `bash`, `git`, `jq`, GNU `date`, and gomaat. On macOS, install GNU `date` with `brew install coreutils` and change `date` to `gdate` in the script.

- [What it reports](#what-it-reports)
- [Run it](#run-it)
- [Turn the report into a plan](#turn-the-report-into-a-plan)
- [Variations](#variations)
- [Gotchas](#gotchas)

## What it reports

The script writes a `handover.md` with two tables.

**Files they wrote most of.** Every file still in the tree where the people leaving added at least half the lines, according to [`knowledge-loss`](../../README.md#knowledge-loss). For each one:

| Column | Source | Why it matters |
|---|---|---|
| Their share | [`knowledge-loss`](../../README.md#knowledge-loss) over all history | How much of the file they wrote. |
| Lines of code | [`cloc --by-file`](../../README.md#cloc) | How much there is to learn. |
| Changes | [`revisions`](../../README.md#revisions) over the last 12 months | How likely someone is to need that knowledge soon. The table is sorted by this, busiest first. |
| Next person | [`whoknows`](../../README.md#whoknows) over the last 12 months, leaving out the people who are leaving | Who else has added lines to it lately. **nobody** means nobody else has touched it in a year. |

A file that's busy and has **nobody** as the next person is the top of the list.

**Files they added in one commit that nobody has changed since**, from [`orphans`](../../README.md#orphans). These are big drops of code with no history to learn from, so the only record of why they exist is in the head of the person leaving.

## Run it

Save this as `handover.sh`. The `excludes` and `authors` lists and the two thresholds near the top are the parts to edit.

```bash
#!/usr/bin/env bash
# Handover report for one or more people leaving.
#
# usage: handover.sh <out dir> <author name> [<author name>...]
#
# Writes <out dir>/handover.md: the files the leavers wrote most of, ranked by
# how actively they're still changing, with who else has worked on each
# recently. The raw analysis output is kept alongside it.
set -euo pipefail

out=$1
shift

# Paths that aren't hand-written; nobody needs a handover for them.
excludes=(--exclude vendor/ --exclude node_modules/ --exclude dist/
  --exclude '*.lock' --exclude package-lock.json --exclude '*.min.js')
authors=(--exclude-author 'dependabot[bot]' --exclude-author 'renovate*')
[ -f .git-blame-ignore-revs ] && authors+=(--ignore-revs-file .git-blame-ignore-revs)

min_share=50  # report files the leavers wrote at least this % of
top=25        # ... and at most this many of them
since=$(date -u -d '12 months ago' +%F)

mkdir -p "$out"
jq -n '$ARGS.positional' --args "$@" > "$out/leavers.json"
# Quote every name so one with a comma or quote survives the CSV reader.
jq -r '.[] | "\"\(gsub("\""; "\"\""))\""' "$out/leavers.json" > "$out/leavers.csv"

# Full history: what someone knows was built up over all of it.
log="$out/gomaat.log"
gomaat generate-log "${excludes[@]}" "${authors[@]}" --outfile "$log"
# The last 12 months: how actively each file is changing now.
recent="$out/recent.log"
gomaat generate-log --after "$since" "${excludes[@]}" "${authors[@]}" --outfile "$recent"

gomaat knowledge-loss -l "$log" --former-authors "$out/leavers.csv" -f json -o "$out/knowledge-loss.json"
gomaat revisions -l "$recent" -f json -o "$out/recent-revisions.json"
gomaat cloc --by-file "${excludes[@]}" -f json -o "$out/cloc.json"
gomaat orphans -l "$log" -f json -o "$out/orphans.json"

# Files still in the tree that the leavers wrote most of, busiest first.
jq --slurpfile revs "$out/recent-revisions.json" --slurpfile cloc "$out/cloc.json" \
  --argjson min "$min_share" --argjson top "$top" '
  ($cloc[0] | map({(.File): (.Code | tonumber)}) | add // {}) as $loc
  | ($revs[0] | map({(.entity): (.["n-revs"] | tonumber)}) | add // {}) as $nrevs
  | map(select((.["knowledge-loss"] | tonumber) >= $min and $loc[.entity] != null)
        | {entity, share: .["knowledge-loss"], loc: $loc[.entity],
           revs: ($nrevs[.entity] // 0)})
  | sort_by(-.revs, -(.share | tonumber)) | .[:$top]
  ' "$out/knowledge-loss.json" > "$out/at-risk.json"

# For each file, the person other than the leavers who added the most to it
# in the last 12 months. Nobody is the finding that matters most.
jq -r '.[].entity' "$out/at-risk.json" | while IFS= read -r file; do
  gomaat whoknows "$file" -l "$log" --after "$since" -f json \
    | jq --arg file "$file" --slurpfile who "$out/leavers.json" '
      map(select(.added != "0" and (.author as $a | $who[0] | index($a) | not)))
      | {($file): .[0]}'
done | jq -s 'add // {}' > "$out/successors.json"

jq -r -n --slurpfile risk "$out/at-risk.json" --slurpfile next "$out/successors.json" \
  --slurpfile orphans "$out/orphans.json" --slurpfile cloc "$out/cloc.json" \
  --slurpfile who "$out/leavers.json" --arg since "$since" '
  $who[0] as $who
  | ($cloc[0] | map({(.File): true}) | add // {}) as $intree
  | "# Handover: \($who | join(", "))",
    "",
    "## Files they wrote most of",
    "",
    "Busiest first. *Changes* and *Next person* cover commits since \($since).",
    "",
    if ($risk[0] | length) == 0 then "None." else
      ("| File | Their share | Lines of code | Changes | Next person |",
       "|---|---|---|---|---|",
       ($risk[0][] | $next[0][.entity] as $n
        | "| `\(.entity)` | \(.share)% | \(.loc) | \(.revs) | \(if $n then "\($n.author) (\($n.ownership)%)" else "**nobody**" end) |"))
    end,
    "",
    "## Files they added in one commit that nobody has changed since",
    "",
    ([$orphans[0][] | select(.author as $a | $who | index($a)) | select($intree[.entity])]
     | if length == 0 then "None." else
         ("| File | Lines added | Date |", "|---|---|---|",
          (.[] | "| `\(.entity)` | \(.["added-lines"]) | \(.date) |"))
       end)
  ' > "$out/handover.md"
```

Run it from the root of the repository, with the person's name as it appears in `git log`:

```bash
bash handover.sh /tmp/handover "Alice Smith"
cat /tmp/handover/handover.md
```

Pass more than one name for a group of people leaving together, such as a team being reorganized. The shares and next people then treat them as one group.

Check the name first with `git log --format='%aN' | sort -u`. If someone has committed under more than one name, add a [`.mailmap`](../../README.md#generating-a-git-log) before you run it. Otherwise each name only gets credit for part of their work, and files they wrote can fall below the threshold.

## Turn the report into a plan

Work down the first table from the top, not the whole thing at once. With a few weeks' notice, the top five to ten files are realistic.

| Situation | What to do |
|---|---|
| Busy, next person is **nobody** | Most urgent. Pick a successor and have them make the next change to the file, with the leaver pairing or reviewing. Doing a real change teaches more than a walkthrough. |
| Busy, next person has a small share | The next person knows a little. Give them the next few changes, with the leaver reviewing. |
| Busy, next person has a large share | Lower risk. Confirm the next person agrees they own it now. |
| Rarely changes | Write down what isn't obvious from the code: why it exists, what depends on it, and what to be careful with. A short note in the file or its directory beats a document nobody finds. |

For the second table, ask the leaver about each file: what it's for, whether it's still used, and whether it was generated. Some will turn out to be dead; see the [deletion candidates recipe](deletion-candidates.md) for how to check.

Run the script again a week before their last day. A file that dropped off the list or gained a next person shows the handover is working.

## Variations

- **By component instead of by file.** Add `-g groups.txt` ([architectural grouping](../../README.md#architectural-grouping)) to the `knowledge-loss` command to see which modules they own, which is easier to discuss with a manager than a list of files.
- **Everyone who's ever left.** Keep an [`alumni.csv`](../../README.md#former-authors) of former authors and run `knowledge-loss` with it. Files with a high share are code nobody on the team wrote. Add the leaver to it on their last day.
- **Before anyone leaves.** The [`bus-factor`](../../README.md#bus-factor) analysis answers the same question for everyone at once, without naming anyone. The [monthly health check](monthly-health-check.md) reports it on a schedule.

## Gotchas

- **History isn't the whole story.** gomaat sees commits, not reviews, design discussions, or on-call experience. Someone who reviewed every change to a file may know it well with no lines to their name. Ask the team as well.
- **"Lines added" isn't "lines still there".** Their share counts every line they added over the file's history, including lines since rewritten. A file they wrote three years ago and someone else has rewritten since can still show a high share; the *Next person* column helps catch that.
- **Names must match exactly.** `knowledge-loss` and `whoknows` compare author names as plain strings. A typo matches nothing and the report comes back empty.
- **Deleted files are left out.** Files that no longer exist drop out, because there's nothing left to hand over.
