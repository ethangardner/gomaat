# Recipe: check whether the architecture still matches the code

Most codebases have an intended structure: layers, modules, or services that are meant to change independently. The code can respect that structure on paper, with every import going the right way, while history shows otherwise. If changing the UI keeps requiring a change to the database code, the boundary between them isn't doing its job.

This recipe groups files into the components you intend, then asks which components change together. It needs gomaat and, for the last step, `jq`.

- [1. Describe the intended structure](#1-describe-the-intended-structure)
- [2. See which components change together](#2-see-which-components-change-together)
- [3. Flag the coupling you didn't expect](#3-flag-the-coupling-you-didnt-expect)
- [Act on the findings](#act-on-the-findings)
- [Gotchas](#gotchas)

## 1. Describe the intended structure

Write a [grouping spec](../../README.md#architectural-grouping) that maps paths to the components as you'd draw them on a whiteboard. Save it as `components.txt`:

```
# Layers
src/web          => Web
src/api          => API
src/domain       => Domain
src/persistence  => Persistence

# Shared code, by regex: anything under a top-level lib/ or shared/
^(lib|shared)/   => Shared
```

Rules are checked in order and the first match wins, so put more specific paths above more general ones. Files that match no rule are left out of the analysis.

Check that the spec covers what you expect before reading anything into the results:

```bash
since=$(date -u -d '12 months ago' +%F)
gomaat generate-log --after "$since" -o year.log

# Commits per component; a component missing here matched no files
gomaat revisions -l year.log -g components.txt
# Compare with the whole log to see how much the spec leaves out
gomaat summary -l year.log
```

If a component is missing from the `revisions` output, its rule matched nothing; see the [gotchas](#gotchas).

## 2. See which components change together

```bash
# Pairs of components that change together, strongest first
gomaat coupling -l year.log -g components.txt -n 5 -m 5 -i 20 --verbose-results
# How many other components each one changes with
gomaat soc -l year.log -g components.txt -n 5 -m 5 -i 20
```

Read `coupling` as "when one of these changes, how often does the other change in the same commit?". The `degree` is the share of their commits they have in common; `shared-revisions` is how many commits that is.

Then compare with the structure you intended:

- **Neighbors coupled.** `Web` and `API`, or `API` and `Domain`, changing together often is expected: a feature usually touches adjacent layers.
- **Layers skipped.** `Web` coupled to `Persistence` means changes reach past the layers in between. Often a query or a data shape leaking upward.
- **Everything coupled to one component.** A component with a much higher `soc` than the rest is a hub. If it's `Shared` or `Domain`, it may be fine. If it's something meant to be peripheral, it's carrying more than it should.
- **Two modules that should be independent coupled.** For example `Orders` and `Payments` with 60% coupling. They aren't independent in practice, whatever the code says.

## 3. Flag the coupling you didn't expect

Write down the pairs you expect to change together in `allowed-coupling.txt`, one per line:

```
# Pairs expected to change together, in either order
Web <-> API
API <-> Domain
Domain <-> Persistence
```

Then list everything else:

```bash
gomaat coupling -l year.log -g components.txt -n 5 -m 5 -i 20 -f json \
  | jq -r --rawfile allowed allowed-coupling.txt '
    ($allowed | split("\n") | map(select(. != "" and (startswith("#") | not)) | split(" <-> ") | sort)) as $ok
    | .[] | select([.entity, .coupled] | sort | IN($ok[]) | not)
    | "\(.entity) <-> \(.coupled): \(.degree)% over \(.["average-revs"]) average revisions"'
```

Example output:

```
Web <-> Persistence: 34% over 41 average revisions
```

Commit both files. The spec and the allowed list are a short, reviewable record of the intended architecture, and the next person can rerun the check and see if anything has drifted. Rerunning it quarterly fits well alongside the [monthly health check](monthly-health-check.md).

## Act on the findings

For each unexpected pair, find the commits behind it:

```bash
# Files in the two components that change together most
gomaat coupling -l year.log -n 3 -m 3 -i 30 | grep -E 'src/web/.*src/persistence/|src/persistence/.*src/web/'
```

Running `coupling` without `-g` shows which files create the component-level coupling. Usually a few file pairs account for most of it. Read some of their shared commits and decide:

| Outcome | When | What to do |
|---|---|---|
| **Fix the code** | A responsibility is in the wrong component | Move it, or add an interface so one side stops depending on the other's details. Then rerun after a few months and check the degree has dropped. |
| **Fix the spec** | The intended structure was wrong or out of date | Update `components.txt` or `allowed-coupling.txt` in a reviewed PR, so the change to the architecture is deliberate and recorded. |
| **Accept** | It's a one-off, such as a cross-cutting rename | Leave it. If it doesn't come back in the next run, it was noise. |

The [refactoring recipe](refactoring.md) shows how to measure whether a fix worked.

## Gotchas

- **In gomaat 1.1.0 and earlier, plain paths only match directories, without a trailing slash.** `src/web/ => Web` silently matches no files, and neither does a single file such as `src/web/index.ts => Entry`. On those versions, write `src/web => Web`, and use a regex for a file: `^src/web/index\.ts$ => Entry`.
- **Large commits count as small ones once grouped.** `--max-changeset-size` (`-s`, default 30) counts components, not files, after grouping. A mass reformat touching 2,000 files across five components counts as a five-entity change and couples all five. Add such commits to a `.git-blame-ignore-revs` file and pass it to `generate-log --ignore-revs-file`.
- **Tests can hide in components.** If tests live beside the code, they group with it, which is usually what you want. If they live in a separate `test/` tree, give them their own component, or every component will look coupled to `Tests`.
- **Coupling needs history.** With fewer than about five shared commits, a high degree is chance. Keep `-m` at 5 or more, and use a year of history rather than a month.
