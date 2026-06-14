---
name: run-gomaat
description: Build, run, and test gomaat (a CLI that mines git history for design insights). Use when asked to run gomaat, build it, generate a git log, run an analysis (coupling, churn, authors, etc.), run cloc, or run its test suite / linter.
---

gomaat is a Go CLI (cobra-based). There's no server or GUI — it's driven
entirely via stdin/argv/stdout. The harness is
`.claude/skills/run-gomaat/smoke.sh`, which builds the binary, generates a
git log from this repo's own history, and runs the main subcommands against
it end to end.

## Prerequisites

- Go (this repo uses `go.mod`'s `go 1.26.3`; `go` must be on `PATH` —
  `/usr/local/go/bin/go` in this container).
- `git` on `PATH` (used both by `generate-log` and by `cloc`, which shells
  out to `git ls-files`).
- Optional, for linting: `golangci-lint` (already on `PATH` in this
  container at `/home/ethan/go/bin/golangci-lint`).

No `go mod download` / network step is needed — `go build` resolves
dependencies from the module cache automatically.

## Build

```bash
make build
# -> /usr/local/go/bin/go build -o ./bin/gomaat ./cmd/gomaat/
```

Produces `./bin/gomaat`.

## Run (agent path)

```bash
.claude/skills/run-gomaat/smoke.sh
```

This:
1. `make build`s the binary.
2. Runs `generate-log` against **this repo's own git history** (no fixture
   needed — the repo is always a valid input).
3. Pipes that log through `summary`, `revisions`, `authors`, and `coupling`.
4. Runs `cloc` against the repo.
5. Confirms a subcommand run without `--log` exits non-zero.

Expected tail of output: `==> SMOKE TEST PASSED`.

### Driving it manually

```bash
./bin/gomaat generate-log --outfile /tmp/gomaat.log   # log of the current repo
./bin/gomaat summary -l /tmp/gomaat.log
./bin/gomaat coupling -l /tmp/gomaat.log -n 2 -m 2 -i 10 -r 10
./bin/gomaat cloc --by-file -r 10
```

All analysis subcommands read CSV-producing `[][]string` from
`internal/analysis/*.go` and print CSV to stdout (or `-o file`). `-r N`
caps output rows; the header row is never counted toward the cap.

## Test

```bash
make test            # go test ./...
go test -v ./...     # verbose
```

Run a single test:

```bash
go test ./internal/analysis/ -run TestCoupling -v
```

## Lint / format (CI parity)

```bash
make vet              # go vet ./...
gofmt -l .            # must print nothing
golangci-lint run      # 0 issues expected
make check            # fmt + vet + test
```

All four passed clean as of this writing.

## Gotchas

- **`--no-renames` surfaces dead file paths in coupling/soc output.**
  `generate-log` runs `git log --no-renames`, so a rename shows up as a
  delete of the old path + add of the new one. Running `coupling`/`soc`
  against this repo's own history will list pairs like
  `cmd/godemaat/main.go,internal/analysis/coupling.go` — `cmd/godemaat/`
  doesn't exist on disk anymore (it was renamed to `cmd/gomaat/`). This is
  correct, expected behavior, not a bug in the harness.
- **`cloc` requires being run inside a git repo** — it shells out to
  `git -C <path> rev-parse --show-toplevel` and `git ls-files`, so it only
  sees tracked files (respecting `.gitignore`). It errors with "no
  git-tracked files found" if pointed outside a repo.
- **Every analysis subcommand requires `--log/-l`** and exits 1 with
  `Error: --log (-l) is required` if omitted — this is the error-path check
  in `smoke.sh`.
- **`generate-log --path .` defaults to cwd**, not the repo containing the
  binary — pass `--path <repo>` explicitly when analyzing a different repo.
