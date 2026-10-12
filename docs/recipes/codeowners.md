# Recipe: draft a CODEOWNERS file from history

A [CODEOWNERS](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners) file asks the right people to review each change. Writing one from scratch means guessing who knows what. This recipe drafts one from who has actually worked on each directory in the last year, for the team to correct.

It's a starting point, not an answer. The people who commit to a directory most aren't always the people who should approve changes to it: a lead may review everything and commit little, and a contractor may have written most of a module and left. Every line needs a human to check it.

It needs `bash` 4 or later, `git`, `jq`, GNU `date`, and gomaat. macOS ships bash 3.2; install a newer one and GNU `date` with `brew install bash coreutils`, and change `date` to `gdate` in the script.

- [How it works](#how-it-works)
- [1. Map authors to GitHub handles](#1-map-authors-to-github-handles)
- [2. Run the script](#2-run-the-script)
- [3. Review the draft](#3-review-the-draft)
- [Keep it current](#keep-it-current)
- [Gotchas](#gotchas)

## How it works

For the repository root and every directory down to a set depth, the script runs [`whoknows`](../../README.md#whoknows), which ranks authors by lines added in the last 12 months. Everyone with at least 25% of a directory's lines becomes an owner, up to three. If nobody reaches 25%, the biggest contributor is the owner.

It runs `whoknows` with [`-p`](../../README.md#team-mapping) and a file mapping author names to GitHub handles. That does three things at once:

- The output is in CODEOWNERS form: `@alice`, not `Alice Smith`.
- If you map several people to a team, such as `@acme/payments`, their lines are added together, so the team can own a directory none of them would own alone.
- Authors you leave out of the map, such as bots and people who've left, drop out of the results.

A directory is only written to the draft when its owners differ from those of the nearest directory above it, because in CODEOWNERS the last matching line wins. That keeps the draft short.

## 1. Map authors to GitHub handles

List everyone who has committed in the last year:

```bash
git log --since='12 months ago' --format='%aN' | sort -u
```

Save the mapping as `owners.csv`, with an `author,team` header. Map a person to their own handle, or to a team if the team should own the code they work on:

```csv
author,team
Alice Smith,@alice
Bob Jones,@acme/payments
Carol White,@acme/payments
Dave Brown,@dave
```

Leave out bots and anyone who shouldn't be asked to review. The script lists every author it found who isn't in the file, so you can check nobody was left out by mistake.

If someone has committed under more than one name, map each name to the same handle, or add a [`.mailmap`](../../README.md#generating-a-git-log) so gomaat sees one name.

## 2. Run the script

Save this as `codeowners-draft.sh`. The `excludes` list and the two thresholds near the top are the parts to edit.

```bash
#!/usr/bin/env bash
# Draft a CODEOWNERS file from who has worked on each directory recently.
#
# usage: codeowners-draft.sh <owners.csv> [depth] > CODEOWNERS.draft
#
# owners.csv maps commit author names to GitHub users or teams, with an
# "author,team" header row:
#   author,team
#   Alice Smith,@alice
#   Bob Jones,@acme/payments
# Authors missing from it are left out of the draft and listed on stderr.
set -euo pipefail

owners_csv=$1
depth=${2:-2}  # how many directory levels deep to suggest owners for

excludes=(--exclude vendor/ --exclude node_modules/ --exclude dist/
  --exclude '*.lock' --exclude package-lock.json --exclude '*.min.js')
ignore=()
[ -f .git-blame-ignore-revs ] && ignore=(--ignore-revs-file .git-blame-ignore-revs)

min_share=25  # list everyone who added at least this % of a directory's lines
max_owners=3  # ... up to this many
since=$(date -u -d '12 months ago' +%F)

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
log="$tmp/gomaat.log"
gomaat generate-log --after "$since" "${excludes[@]}" "${ignore[@]}" --outfile "$log" 2>/dev/null

# -p drops authors it can't map without a word, so say who they are; a bot
# or someone who's left can stay unmapped on purpose.
gomaat author-churn -l "$log" -f json | jq -r '.[].author' | sort > "$tmp/authors"
cut -d, -f1 "$owners_csv" | tr -d '"' | sed 's/^ *//; s/ *$//' | sort > "$tmp/mapped"
comm -23 "$tmp/authors" "$tmp/mapped" | sed 's/^/not in owners file: /' >&2

# owners_of <path> prints the owners of path, or nothing if no mapped author
# has added a line to it since $since.
owners_of() {
  gomaat whoknows "$1" -l "$log" -p "$owners_csv" --after "$since" -f json \
    | jq -r --argjson min "$min_share" --argjson max "$max_owners" '
      map(select(.added != "0")) as $all
      | ($all | map(select((.ownership | tonumber) >= $min))) as $top
      # Nobody over the threshold: the biggest contributor still beats no owner.
      | (if $top == [] then $all[:1] else $top[:$max] end)
      | map(.author) | join(" ")'
}

echo "# Draft generated by gomaat on $(date -u +%F) from commits since $since."
echo "# Review every line before committing it."
root=$(owners_of .)
echo "* $root"

# Only write a directory when its owners differ from the nearest one above
# it, since CODEOWNERS applies the last matching line.
declare -A owners=(["/"]="$root")
git ls-files | awk -F/ -v depth="$depth" '
  { p = ""; for (i = 1; i < NF && i <= depth; i++) { p = p $i "/"; print p } }' \
  | LC_ALL=C sort -u | while IFS= read -r dir; do
    o=$(owners_of "${dir%/}")
    [ -n "$o" ] || continue
    # Sorted order puts every directory after its parents.
    parent=$dir
    while :; do
      parent=${parent%/}
      [[ $parent == */* ]] && parent=${parent%/*}/ || parent=/
      [ -n "${owners[$parent]+x}" ] && break
    done
    owners[$dir]=$o
    [ "$o" = "${owners[$parent]}" ] || echo "/$dir $o"
  done
```

Run it from the root of the repository. The optional second argument is how many directory levels deep to go (default 2):

```bash
bash codeowners-draft.sh owners.csv > CODEOWNERS.draft
```

Example output:

```
not in owners file: dependabot[bot]
# Draft generated by gomaat on 2026-10-12 from commits since 2025-10-12.
# Review every line before committing it.
* @acme/payments @alice
/src/api/ @acme/payments
/src/billing/ @alice
/src/web/ @acme/payments
```

The `not in owners file` lines go to stderr, so they don't end up in the draft.

## 3. Review the draft

Open a PR that adds the draft as `.github/CODEOWNERS` and ask each owner it names to review their lines. For each line, ask:

- **Is this who should approve changes here?** Swap in the person or team who should, even if they don't show up in the history.
- **Is one person enough?** A directory with a single owner means one person reviews every change to it. If that person is away, reviews stop. Consider a team instead.
- **Is anything missing?** Directories with no commits in the last year inherit their parent's owners. Check that's right, especially for rarely changed but sensitive code such as CI workflows, auth, and billing.
- **Is the root owner right?** The `*` line is the fallback for everything else. It's often better as a team than as the few most active people.

Leave **Require review from Code Owners** in branch protection off at first. Run with the file for a few weeks, see whether the review requests land in the right place, and adjust. Turn the requirement on once the team trusts the file.

## Keep it current

Rerun the script every six months or so, or when someone joins or leaves, and diff the result against the committed file:

```bash
bash codeowners-draft.sh owners.csv > CODEOWNERS.draft
diff .github/CODEOWNERS CODEOWNERS.draft
```

The diff shows where the people doing the work have drifted from the people reviewing it. Don't copy the draft over the file; you'll lose the corrections from step 3. Apply the differences that make sense by hand.

When someone leaves, the [handover recipe](offboarding.md) finds the code they own that needs a new owner.

## Gotchas

- **Owners need write access.** GitHub ignores a user or team in CODEOWNERS that doesn't have write access to the repository, and a team must be visible. The file looks fine but nobody is asked to review.
- **Paths are case-sensitive.** CODEOWNERS paths are matched case-sensitively, like `git` paths. The script uses the paths as git stores them.
- **The window is 12 months.** A directory with no commits in that window has no owners in the draft. Change `since` in the script to look further back, at the cost of naming people who may have moved on.
- **Names in the CSV can't contain commas.** The check for missing authors splits on the first comma. Quote names with commas as the [team mapping](../../README.md#team-mapping) docs describe, and expect them to be reported as missing anyway.
