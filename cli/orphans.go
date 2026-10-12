package cli

import (
	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/model"
)

func newOrphansCmd() *cobra.Command {
	var minLines int
	cmd := &cobra.Command{
		Use:   "orphans",
		Short: "Files added in one large commit and never revised",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(analysis.Orphans, analysis.FormatOrphans, model.Options{MinLines: minLines})
		},
	}
	cmd.Flags().IntVar(&minLines, "min-lines", 200, "minimum lines added in the single commit to report an entity")
	return cmd
}
