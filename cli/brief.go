package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/brief"
)

// newBriefCmd builds the brief subcommand. It writes one Markdown document
// rather than rows, so of the global flags it honors --log, --outfile,
// --group and --team-map-file, and rejects --format and --rows (use --top)
// instead of silently ignoring them.
func newBriefCmd() *cobra.Command {
	var top, dormantMonths int
	var ageTimeNow string

	cmd := &cobra.Command{
		Use:   "brief",
		Short: "One Markdown architecture overview built from several analyses",
		Long: `Runs summary, revisions, coupling, main-dev, age and soc in-process and writes
one Markdown document: dataset summary, hotspots, coupling, main developers per
component, active vs. dormant files, and a suggested reading order. Commit it
as a starting ARCHITECTURE.md or paste it into an agent's context.

Writes Markdown, so --format and --rows don't apply; use --top to size each
section. Without -g, main developers are grouped by directory.

Examples:
  gomaat brief -l logfile.log > ARCHITECTURE.md
  gomaat brief -l logfile.log -g groups.txt --top 5 -o ARCHITECTURE.md`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFormat != "csv" {
				return fmt.Errorf("--format (-f): brief writes Markdown, so --format doesn't apply")
			}
			if maxRows != 0 {
				return fmt.Errorf("--rows (-r) doesn't apply to brief; use --top to limit rows per section")
			}
			now, err := parseDateFlag("--age-time-now", ageTimeNow, time.Now())
			if err != nil {
				return err
			}
			commits, err := loadCommits()
			if err != nil {
				return err
			}

			doc := brief.Render(commits, brief.Options{Top: top, Now: now, DormantMonths: dormantMonths, Grouped: groupFile != ""})
			if outFile != "" {
				return os.WriteFile(outFile, []byte(doc), 0o644)
			}
			_, err = io.WriteString(os.Stdout, doc)
			return err
		},
	}
	cmd.Flags().IntVar(&top, "top", 10, "max rows per section (0 = no limit)")
	cmd.Flags().IntVar(&dormantMonths, "dormant-months", 12, "months without changes after which a file counts as dormant")
	cmd.Flags().StringVarP(&ageTimeNow, "age-time-now", "d", "", "reference date for activity (YYYY-MM-DD, default: today)")
	return cmd
}
