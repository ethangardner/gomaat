package cli

import (
	"path"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/model"
)

// newWhoKnowsCmd builds the whoknows subcommand. Its one argument is a file,
// directory, or (with -g) group name as it appears in the log; it's cleaned
// to forward slashes so "./src/core/" and Windows-style "src\core" both match
// the log's "src/core/..." entities, and "." covers the whole log.
func newWhoKnowsCmd() *cobra.Command {
	var after string

	cmd := &cobra.Command{
		Use:   "whoknows <path>",
		Short: "Authors ranked by recent lines added to a file or directory",
		Long: `Ranks who currently knows a file or directory: authors by lines added to it
on or after --after (default: 12 months ago). A directory aggregates every
file beneath it.

Unlike main-dev, which counts every line ever added, contributions before
--after are dropped, so a large early author who has since moved on doesn't
outrank the people maintaining the code now.

Examples:
  gomaat whoknows -l logfile.log src/core/Engine.java
  gomaat whoknows -l logfile.log src/core --after 2025-01-01`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			since, err := parseDateFlag("--after", after, time.Now().AddDate(-1, 0, 0))
			if err != nil {
				return err
			}
			p := path.Clean(filepath.ToSlash(args[0]))
			if p == "." {
				p = ""
			}
			return runAnalysis(analysis.WhoKnows, analysis.FormatWhoKnows, model.Options{Path: p, After: since})
		},
	}
	cmd.Flags().StringVar(&after, "after", "", "count only commits on or after this date (YYYY-MM-DD, default: 12 months ago)")
	return cmd
}
