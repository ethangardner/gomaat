# Recipe: find code that might be safe to delete

Code nobody uses still costs something: it shows up in searches, slows builds, needs dependency upgrades, and makes the codebase harder to learn. History can't prove code is unused, but it's a cheap way to make a short list of files worth checking.

This recipe lists large files that haven't changed in a long time, then walks through how to check each one before deleting it. It needs `bash`, `git`, `jq`, and gomaat.

- [What it reports](#what-it-reports)
- [Run it](#run-it)
- [Check before you delete](#check-before-you-delete)
- [Delete safely](#delete-safely)
- [Gotchas](#gotchas)

## What it reports

`candidates.md` lists every file still in the tree, in a language [`cloc`](../../README.md#cloc) recognises, that:

- hasn't changed in at least 24 months, according to [`age`](../../README.md#age), and
- has at least 50 lines of code, so it's worth the time to check.

Biggest first, with two more columns to help decide where to start:

| Column | Source | Why it matters |
|---|---|---|
| Single commit | [`orphans`](../../README.md#orphans) | The file was added in one commit and never changed. Code that's never been fixed or extended has often never been used much either. |
| Main author | [`main-dev`](../../README.md#main-dev) | Who to ask what it's for. |

**Old isn't the same as dead.** A file can go years without changes because it works: a date library, or a parser for a format that hasn't changed. The list is where to start looking, not what to delete.

## Run it

Save this as `deletion-candidates.sh`. The `excludes` and `authors` lists and the two thresholds near the top are the parts to edit.

```bash
#!/usr/bin/env bash
# Files worth checking for deletion: code nobody has changed in a long time.
#
# usage: deletion-candidates.sh <out dir>
#
# Writes <out dir>/candidates.md, biggest first, plus the raw analysis output.
set -euo pipefail

out=$1

excludes=(--exclude vendor/ --exclude node_modules/ --exclude dist/
  --exclude '*.lock' --exclude package-lock.json --exclude '*.min.js')
authors=(--exclude-author 'dependabot[bot]' --exclude-author 'renovate*')
[ -f .git-blame-ignore-revs ] && authors+=(--ignore-revs-file .git-blame-ignore-revs)

stale_months=24  # untouched for at least this long
min_code=50      # and at least this many lines of code, so it's worth a look

mkdir -p "$out"
# Full history: age only knows about files that appear in the log, so a
# windowed log would leave out exactly the old files this is looking for.
log="$out/gomaat.log"
gomaat generate-log "${excludes[@]}" "${authors[@]}" --outfile "$log"

gomaat age -l "$log" -f json -o "$out/age.json"
gomaat orphans -l "$log" -f json -o "$out/orphans.json"
gomaat main-dev -l "$log" -f json -o "$out/main-dev.json"
gomaat cloc --by-file "${excludes[@]}" -f json -o "$out/cloc.json"

# cloc lists only files in the tree in a language it recognises, so deleted
# files and binaries drop out of the join.
jq -r -n --slurpfile age "$out/age.json" --slurpfile orphans "$out/orphans.json" \
  --slurpfile dev "$out/main-dev.json" --slurpfile cloc "$out/cloc.json" \
  --argjson months "$stale_months" --argjson min "$min_code" '
  ($age[0] | map({(.entity): (.["age-months"] | tonumber)}) | add // {}) as $age_of
  | ($orphans[0] | map({(.entity): true}) | add // {}) as $orphan
  | ($dev[0] | map({(.entity): .["main-dev"]}) | add // {}) as $author
  | [ $cloc[0][]
      | {file: .File, code: (.Code | tonumber), age: $age_of[.File]}
      | select(.age != null and .age >= $months and .code >= $min) ]
  | sort_by(-.code)
  | "# Deletion candidates",
    "",
    "Code untouched for \($months)+ months, biggest first. *Single commit* means it was added once and never changed.",
    "",
    if length == 0 then "None." else
      ("| File | Lines of code | Months untouched | Single commit | Main author |",
       "|---|---|---|---|---|",
       (.[] | "| `\(.file)` | \(.code) | \(.age) | \(if $orphan[.file] then "yes" else "" end) | \($author[.file] // "") |"))
    end
  ' > "$out/candidates.md"
```

Run it from the root of the repository:

```bash
bash deletion-candidates.sh /tmp/candidates
cat /tmp/candidates/candidates.md
```

Example output:

```
| File | Lines of code | Months untouched | Single commit | Main author |
|---|---|---|---|---|
| `src/legacy/importer.go` | 300 | 43 | yes | Dave Brown |
| `src/util/strings.go` | 120 | 42 |  | Dave Brown |
```

If the list is long, raise `stale_months` or `min_code` until it's short enough to work through. If it's empty, lower them.

## Check before you delete

Take the files one at a time, starting from the top. For each, look for evidence that it's used:

1. **Search for references.** Search for the file name, and for its exported names, across the whole repository, including config, build files, scripts, and docs. Code can be loaded by name from a string, a config file, or a plugin list, where a plain import search won't find it.
2. **Ask a tool that knows the language.** Static analysis finds unreachable code more reliably than history. For example: [`deadcode`](https://pkg.go.dev/golang.org/x/tools/cmd/deadcode) for Go, [Knip](https://knip.dev/) for JavaScript and TypeScript, [Vulture](https://github.com/jendrikseipp/vulture) for Python, or your IDE's "find usages".
3. **Check production.** If you have coverage data from tests, or logs and metrics from production, see whether the code runs. An endpoint with no requests in six months is a stronger signal than any history.
4. **Ask the main author**, or [whoever knows the area now](../../README.md#whoknows). They often know at once: "that was for the 2023 migration, it's done."
5. **Check outside this repository.** A library's public API, a file served at a fixed URL, a script someone runs by hand, or a schema another service reads can all be used with no reference in the code.

Delete only when none of these turns up a use. If you're unsure, leave it and move on; there are other files on the list.

## Delete safely

- **One file or feature per PR.** Small deletions are easy to review and easy to revert. A PR deleting 40 files gets a skim, and a mistake in it is hard to find.
- **Say why in the PR.** List what you checked: "no references, deadcode reports it unreachable, no requests in production logs since March, confirmed with Dave." That's the record if it turns out to be needed.
- **Delete the tests with it.** Tests for deleted code only keep the deleted code's dependencies alive.
- **Revert if needed.** The code is still in git history. If something breaks, `git revert` brings it back in minutes, which is why small PRs matter.

Track what you delete. Lines of code removed is a satisfying number to report, and a before/after [`cloc`](../../README.md#cloc) total shows the progress.

## Gotchas

- **The log must cover all history.** `age` only knows about files that appear in the log. If you pass a log generated with `--after`, every file last changed before that date is missing from `age`, which is exactly the files this recipe is looking for. The script generates a full log for this reason.
- **Mass changes reset the clock.** A reformat, a license header update, or a bulk rename touches every file and makes them all look recently changed. List those commits in a `.git-blame-ignore-revs` file at the root of the repository; the script skips them.
- **Moved files look new.** gomaat [doesn't follow renames](../adr/0004-no-renames-no-merges-in-log.md), so a file moved last month has an age of one month, however old its contents are.
- **Generated and vendored code.** Code generated once and committed, or vendored in, often looks old and single-commit. Add it to `excludes` rather than deleting it.
- **Non-code files are skipped.** Files `cloc` doesn't recognise as a language, such as images, data files, and binaries, aren't listed, even if they're unused.
