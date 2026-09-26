#!/usr/bin/env bash
# Runs every analysis with one gomaat build and writes each result to its own
# file, so smoke-test.yml can diff a main build against a PR build.
#
# usage: run.sh <gomaat binary> <shared log> <repo> <out dir>
#
# A command that fails (e.g. one the main build doesn't have yet) records its
# stderr and exit code instead of aborting, so the diff shows the difference.
set -uo pipefail

bin=$1 log=$2 repo=$3 out=$4
fixtures=$(dirname "$0")
mkdir -p "$out"

run() {
  local name=$1
  shift
  if ! "$bin" "$@" > "$out/$name" 2> "$out/$name.err"; then
    echo "exit status $?" >> "$out/$name.err"
  fi
  # Keep the error file only when there's something in it.
  [ -s "$out/$name.err" ] || rm -f "$out/$name.err"
}

# Log-based analyses, in both output formats.
thresholds=(-n 2 -m 2 -i 10)
for fmt in csv json; do
  for cmd in summary statistics revisions authors abs-churn author-churn \
    entity-churn entity-ownership entity-effort main-dev main-dev-by-revs \
    refactoring-main-dev fragmentation bus-factor communication identity; do
    run "$cmd.$fmt" "$cmd" -l "$log" -f "$fmt"
  done
  run "coupling.$fmt" coupling -l "$log" "${thresholds[@]}" -f "$fmt"
  run "soc.$fmt" soc -l "$log" "${thresholds[@]}" -f "$fmt"
  # A fixed reference date keeps age independent of when the job runs.
  run "age.$fmt" age -l "$log" -d 2026-01-01 -f "$fmt"
  run "knowledge-loss.$fmt" knowledge-loss -l "$log" \
    --former-authors "$fixtures/former-authors.csv" -f "$fmt"
done

# Grouping and team mapping.
run grouped-coupling.csv coupling -l "$log" "${thresholds[@]}" -g "$fixtures/groups.txt"
run grouped-revisions.csv revisions -l "$log" -g "$fixtures/groups.txt"
run team-communication.csv communication -l "$log" -p "$fixtures/teams.csv"
run team-fragmentation.csv fragmentation -l "$log" -p "$fixtures/teams.csv"

# Commands that read the repository directly. rework judges lines as of
# --before, so pin it to the day of the repo's last commit instead of now.
run generate-log.log generate-log --path "$repo"
run cloc.csv cloc --path "$repo"
run cloc-by-file.csv cloc --path "$repo" --by-file
run rework.csv rework --path "$repo" --before "$(git -C "$repo" log -1 --format=%cs)"
