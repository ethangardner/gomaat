package cli

import (
	"fmt"
	"io"
	"iter"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/gitdiff"
	"github.com/ethangardner/gomaat/internal/model"
)

func newReworkCmd() *cobra.Command {
	var path, window, after, before string
	var excludes []string

	cmd := &cobra.Command{
		Use:   "rework [pathspec...]",
		Short: "Share of added lines removed or rewritten within a time window, per entity",
		Long: `Walks the repository's first-parent history with full patches and reports, per
file, how many of the lines added were removed or substantially rewritten
within --rework-window of landing. Moved lines and small edits keep their
original provenance and are not counted as rework.

Unlike most analyses this reads the repository directly (--path) rather than a
--log file, and it is much more expensive: scope it to known hotspots with
pathspecs and/or --after where possible.

Examples:
  gomaat rework
  gomaat rework --rework-window 2w --after 2024-01-01
  gomaat rework --path /path/to/repo src/billing/ src/api/handlers.go
  gomaat rework --exclude vendor/ --exclude '*.pb.go' -r 20`,
		RunE: func(cmd *cobra.Command, pathspecs []string) error {
			if err := validateOutputFormat(); err != nil {
				return err
			}
			if err := rejectLogOnlyFlags(); err != nil {
				return err
			}
			opts, err := reworkOptions(window, after, before)
			if err != nil {
				return err
			}
			results, err := runRework(path, after, before, pathspecs, excludes, opts)
			if err != nil {
				return err
			}
			return writeRows(analysis.FormatRework(results, opts))
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "path to the git repository")
	cmd.Flags().StringVarP(&window, "rework-window", "w", "14d", "how soon after landing a removal counts as rework (e.g. 14d, 2w, 36h)")
	cmd.Flags().StringVar(&after, "after", "", "only walk commits after this date (YYYY-MM-DD); earlier lines are untracked")
	cmd.Flags().StringVar(&before, "before", "", "only walk commits before this date (YYYY-MM-DD); also the date lines are judged at (default: now)")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "exclude paths matching this pattern (repeatable, supports globs)")

	return cmd
}

// reworkOptions validates rework's --rework-window, --after and --before
// flags into the options analysis.Rework reads.
func reworkOptions(window, after, before string) (model.Options, error) {
	w, err := parseWindow(window)
	if err != nil {
		return model.Options{}, err
	}
	if _, err := parseDateFlag("--after", after, time.Time{}); err != nil {
		return model.Options{}, err
	}
	// Deletions after --before are invisible, so judge lines as of then.
	now, err := parseDateFlag("--before", before, time.Now())
	if err != nil {
		return model.Options{}, err
	}
	return model.Options{ReworkWindow: w, ReworkTimeNow: now}, nil
}

// runRework streams the first-parent patch history of the repository at path
// through analysis.Rework.
func runRework(path, after, before string, pathspecs, excludes []string, opts model.Options) ([]analysis.ReworkResult, error) {
	// The explicit prefixes override diff.noprefix/diff.mnemonicPrefix,
	// since gitdiff strips a/ and b/ to recover paths.
	gitArgs := slices.Concat(
		[]string{
			"log", "--reverse", "--first-parent", "--diff-merges=first-parent",
			"-p", "-U0", "--no-renames", "--no-color", "--no-ext-diff", "--no-textconv",
			"--src-prefix=a/", "--dst-prefix=b/",
			"--format=" + gitdiff.Format,
		},
		dateRangeArgs(after, before),
		buildPathspecArgs(pathspecs, excludes),
	)

	var results []analysis.ReworkResult
	err := streamGit(path, gitArgs, func(r io.Reader) (err error) {
		results, err = analysis.Rework(excludeFiles(gitdiff.Parse(r), excludes), opts)
		return err
	})
	return results, err
}

// rejectLogOnlyFlags errors if a persistent flag that only applies to
// numstat log analyses is set, rather than silently ignoring it.
func rejectLogOnlyFlags() error {
	for _, f := range []struct{ name, value string }{
		{"--log", logFile},
		{"--group", groupFile},
		{"--team-map-file", teamMapFile},
	} {
		if f.value != "" {
			return fmt.Errorf("%s is not supported by rework, which reads the repository directly (see --path)", f.name)
		}
	}
	return nil
}

// parseWindow parses a positive duration given in days ("14d"), weeks ("2w"),
// or any unit time.ParseDuration accepts ("36h").
func parseWindow(s string) (time.Duration, error) {
	d, err := parseDuration(s)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("--rework-window: expected a positive duration such as 14d, 2w or 36h, got %q", s)
	}
	return d, nil
}

func parseDuration(s string) (time.Duration, error) {
	for suffix, unit := range map[string]time.Duration{"d": 24 * time.Hour, "w": 7 * 24 * time.Hour} {
		if n, ok := strings.CutSuffix(s, suffix); ok {
			count, err := strconv.Atoi(n)
			return time.Duration(count) * unit, err
		}
	}
	return time.ParseDuration(s)
}

// excludeFiles drops file diffs whose path matches an --exclude pattern.
// Directory patterns are also passed to git as pathspecs; this catches the
// glob patterns git's pathspec matching can't express the same way.
func excludeFiles(commits iter.Seq2[gitdiff.Commit, error], excludes []string) iter.Seq2[gitdiff.Commit, error] {
	if len(excludes) == 0 {
		return commits
	}
	excluded := func(f gitdiff.FileDiff) bool { return pathMatchesAnyExclude(f.Path, excludes) }
	return func(yield func(gitdiff.Commit, error) bool) {
		for c, err := range commits {
			if err == nil {
				c.Files = slices.DeleteFunc(c.Files, excluded)
			}
			if !yield(c, err) {
				return
			}
		}
	}
}
