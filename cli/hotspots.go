package cli

import (
	"fmt"

	"github.com/hhatto/gocloc"
	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/grouper"
	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/parser"
)

func newHotspotsCmd() *cobra.Command {
	var path string
	var excludes []string

	cmd := &cobra.Command{
		Use:   "hotspots",
		Short: "Rank entities by churn x size (revisions x current lines of code)",
		Long: `Joins revision counts (churn) with current lines-of-code (size) to rank
entities by hotspot risk. Files present in the log but not on disk, or on
disk but not in the log, are excluded from the report.

Examples:
  gomaat hotspots -l logfile.log --path .
  gomaat hotspots -l logfile.log --path . --exclude vendor/ -r 20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if logFile == "" {
				return fmt.Errorf("--log (-l) is required")
			}
			if err := validateOutputFormat(); err != nil {
				return err
			}

			commits, err := parser.ParseFile(logFile)
			if err != nil {
				return err
			}

			var matchGroup func(string) string
			if groupFile != "" {
				groups, err := grouper.LoadFile(groupFile)
				if err != nil {
					return err
				}
				commits = grouper.Apply(commits, groups)
				matchGroup = func(p string) string { return grouper.MatchPath(p, groups) }
			}

			trackedFiles, repoRoot, err := gitTrackedFiles(path, excludes)
			if err != nil {
				return err
			}
			if len(trackedFiles) == 0 {
				return fmt.Errorf("no git-tracked files found in %s", path)
			}

			result, err := newClocProcessor().Analyze(trackedFiles)
			if err != nil {
				return fmt.Errorf("cloc failed: %w", err)
			}
			relativizeResult(result, repoRoot)
			if len(excludes) > 0 {
				applyClocExcludes(result, excludes)
			}

			locByFile := clocLinesByFile(result)
			if matchGroup != nil {
				locByFile = groupLOC(locByFile, matchGroup)
			}

			results := analysis.Hotspots(commits, locByFile, model.Options{})
			return writeRows(analysis.FormatHotspots(results, model.Options{}))
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "path to analyze for current lines of code")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "exclude paths matching this pattern (repeatable, supports globs)")

	return cmd
}

// clocLinesByFile extracts lines-of-code per repo-relative path from a
// (relativized) gocloc result.
func clocLinesByFile(result *gocloc.Result) map[string]int {
	m := make(map[string]int, len(result.Files))
	for path, f := range result.Files {
		m[path] = int(f.Code)
	}
	return m
}

// groupLOC re-keys locByFile by matched group name, summing lines-of-code
// for paths that share a group. Paths matching no group are dropped,
// mirroring grouper.Apply's drop-unmatched convention.
func groupLOC(locByFile map[string]int, matchGroup func(string) string) map[string]int {
	out := make(map[string]int, len(locByFile))
	for path, lines := range locByFile {
		name := matchGroup(path)
		if name == "" {
			continue
		}
		out[name] += lines
	}
	return out
}
