#!/usr/bin/env bash
# Smoke test / driver for gomaat: build it, run it against this repo's
# own git history, and exercise the main subcommands end to end.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"

echo "==> Building"
make build

BIN="$ROOT/bin/gomaat"
WORKDIR="$(mktemp -d)"
LOG="$WORKDIR/gomaat.log"

echo "==> generate-log (this repo's own history -> $LOG)"
"$BIN" generate-log --path "$ROOT" --outfile "$LOG"
test -s "$LOG"

echo "==> summary"
"$BIN" summary -l "$LOG"

echo "==> revisions (top 5)"
"$BIN" revisions -l "$LOG" -r 5

echo "==> authors (top 5)"
"$BIN" authors -l "$LOG" -r 5

echo "==> coupling (loose thresholds, top 5)"
"$BIN" coupling -l "$LOG" -n 2 -m 2 -i 10 -r 5

echo "==> cloc (by language, top 8)"
"$BIN" cloc --path "$ROOT" -r 8

echo "==> error path: missing --log must exit non-zero"
if "$BIN" authors >/dev/null 2>&1; then
  echo "FAIL: expected non-zero exit for missing --log" >&2
  exit 1
fi
echo "OK: exits non-zero without --log"

rm -rf "$WORKDIR"
echo "==> SMOKE TEST PASSED"
