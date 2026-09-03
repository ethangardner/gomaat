#!/usr/bin/env bash
# Lists PRs merged to main since a given point, oldest first, as JSON.
#
# Usage: fetch-merged-prs.sh [since-ref]
#   since-ref  A git ref to diff from (tag, SHA, ...). Defaults to the most
#              recent tag (`git describe --tags --abbrev=0`) — i.e. everything
#              merged since the last release.
#
# Output: a JSON array of {number, title, url, mergedAt, labels}, sorted by
# mergedAt ascending (matches this repo's .goreleaser.yml changelog sort).
set -euo pipefail

since_ref="${1:-$(git describe --tags --abbrev=0)}"
since_date="$(git log -1 --format=%aI "$since_ref")"

gh pr list --state merged --base main \
  --search "merged:>$since_date" \
  --json number,title,url,mergedAt,labels \
  --jq 'sort_by(.mergedAt)'
