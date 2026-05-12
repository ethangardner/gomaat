package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"gomaat/internal/analysis"
	"gomaat/internal/grouper"
	"gomaat/internal/model"
	"gomaat/internal/output"
	"gomaat/internal/parser"
	"gomaat/internal/teammapper"
)

// persistent flag values (shared across all analysis subcommands)
var (
	logFile     string
	outFile     string
	maxRows     int
	groupFile   string
	teamMapFile string
)

var rootCmd = &cobra.Command{
	Use:   "gomaat",
	Short: "Mine and analyze git history",
	Long: `gomaat mines git version-control history to surface design insights:
logical coupling, code churn, authorship patterns, code age, and more.

Generate a git log first:
  gomaat generate-log --after 2023-01-01 -o logfile.log

Then run an analysis:
  gomaat authors -l logfile.log
  gomaat coupling -l logfile.log --min-coupling 30`,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&logFile, "log", "l", "", "git log file to analyze")
	rootCmd.PersistentFlags().StringVarP(&outFile, "outfile", "o", "", "write output to file (default: stdout)")
	rootCmd.PersistentFlags().IntVarP(&maxRows, "rows", "r", 0, "max result rows (0 = no limit)")
	rootCmd.PersistentFlags().StringVarP(&groupFile, "group", "g", "", "architectural grouping spec file")
	rootCmd.PersistentFlags().StringVarP(&teamMapFile, "team-map-file", "p", "", "CSV file mapping author to team")
}

// analysisFunc is the signature all analysis functions satisfy.
type analysisFunc func([]model.Commit, model.Options) [][]string

// runAnalysis is the shared execution path for all analysis subcommands.
func runAnalysis(fn analysisFunc, opts model.Options) error {
	if logFile == "" {
		return fmt.Errorf("--log (-l) is required")
	}

	commits, err := parser.ParseFile(logFile)
	if err != nil {
		return err
	}

	if groupFile != "" {
		groups, err := grouper.LoadFile(groupFile)
		if err != nil {
			return err
		}
		commits = grouper.Apply(commits, groups)
	}

	if teamMapFile != "" {
		lookup, err := teammapper.LoadFile(teamMapFile)
		if err != nil {
			return err
		}
		commits = teammapper.Apply(commits, lookup)
	}

	rows := fn(commits, opts)

	if outFile != "" {
		return output.WriteFile(outFile, rows, maxRows)
	}
	return output.Write(os.Stdout, rows, maxRows)
}

// couplingOpts holds coupling-specific flag values.
type couplingFlags struct {
	minRevs          int
	minSharedRevs    int
	minCoupling      float64
	maxCoupling      float64
	maxChangesetSize int
	verboseResults   bool
}

func newCouplingCmd(use, short string, fn analysisFunc, addFlags func(*cobra.Command, *couplingFlags)) *cobra.Command {
	cf := &couplingFlags{
		minRevs:          5,
		minSharedRevs:    5,
		minCoupling:      30,
		maxCoupling:      100,
		maxChangesetSize: 30,
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := model.Options{
				MinRevs:          cf.minRevs,
				MinSharedRevs:    cf.minSharedRevs,
				MinCoupling:      cf.minCoupling,
				MaxCoupling:      cf.maxCoupling,
				MaxChangesetSize: cf.maxChangesetSize,
				VerboseResults:   cf.verboseResults,
			}
			return runAnalysis(fn, opts)
		},
	}
	cmd.Flags().IntVarP(&cf.minRevs, "min-revs", "n", cf.minRevs, "minimum revisions to include entity")
	cmd.Flags().IntVarP(&cf.minSharedRevs, "min-shared-revs", "m", cf.minSharedRevs, "minimum shared revisions for coupling")
	cmd.Flags().Float64VarP(&cf.minCoupling, "min-coupling", "i", cf.minCoupling, "minimum coupling percentage")
	cmd.Flags().Float64VarP(&cf.maxCoupling, "max-coupling", "x", cf.maxCoupling, "maximum coupling percentage")
	cmd.Flags().IntVarP(&cf.maxChangesetSize, "max-changeset-size", "s", cf.maxChangesetSize, "max modules in changeset for coupling")
	if addFlags != nil {
		addFlags(cmd, cf)
	}
	return cmd
}

func simpleCmd(use, short string, fn analysisFunc) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(fn, model.Options{})
		},
	}
}

func init() {
	// Simple analysis subcommands
	rootCmd.AddCommand(simpleCmd("authors", "Count authors and revisions per entity", analysis.Authors))
	rootCmd.AddCommand(simpleCmd("revisions", "Count revisions per entity", analysis.Revisions))
	rootCmd.AddCommand(simpleCmd("summary", "Overview statistics for the log", analysis.Summary))
	rootCmd.AddCommand(simpleCmd("identity", "Dump raw parsed commit data (debug)", analysis.Identity))
	rootCmd.AddCommand(simpleCmd("abs-churn", "Lines added/deleted aggregated by date", analysis.AbsChurn))
	rootCmd.AddCommand(simpleCmd("author-churn", "Lines added/deleted aggregated by author", analysis.AuthorChurn))
	rootCmd.AddCommand(simpleCmd("entity-churn", "Lines added/deleted aggregated by entity", analysis.EntityChurn))
	rootCmd.AddCommand(simpleCmd("entity-ownership", "Churn per author per entity", analysis.EntityOwnership))
	rootCmd.AddCommand(simpleCmd("main-dev", "Main developer per entity by lines added", analysis.MainDev))
	rootCmd.AddCommand(simpleCmd("refactoring-main-dev", "Main developer per entity by lines deleted", analysis.RefactoringMainDev))
	rootCmd.AddCommand(simpleCmd("entity-effort", "Revision count per author per entity", analysis.EntityEffort))
	rootCmd.AddCommand(simpleCmd("main-dev-by-revs", "Main developer per entity by revision count", analysis.MainDevByRevs))
	rootCmd.AddCommand(simpleCmd("fragmentation", "Author fragmentation (fractal value) per entity", analysis.Fragmentation))
	rootCmd.AddCommand(simpleCmd("communication", "Team communication needs based on shared code", analysis.Communication))

	// Age subcommand (needs --age-time-now flag)
	var ageTimeNow string
	ageCmd := &cobra.Command{
		Use:   "age",
		Short: "Months since last modification per entity",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := model.Options{}
			if ageTimeNow != "" {
				t, err := time.Parse("2006-01-02", ageTimeNow)
				if err != nil {
					return fmt.Errorf("--age-time-now: expected YYYY-MM-DD, got %q", ageTimeNow)
				}
				opts.AgeTimeNow = t
			} else {
				opts.AgeTimeNow = time.Now()
			}
			return runAnalysis(analysis.Age, opts)
		},
	}
	ageCmd.Flags().StringVarP(&ageTimeNow, "age-time-now", "d", "", "reference date for age calculation (YYYY-MM-DD, default: today)")
	rootCmd.AddCommand(ageCmd)

	// Coupling subcommand (with verbose flag)
	couplingCmd := newCouplingCmd("coupling", "Detect temporal coupling between modules", analysis.Coupling,
		func(cmd *cobra.Command, cf *couplingFlags) {
			cmd.Flags().BoolVar(&cf.verboseResults, "verbose-results", false, "include extra columns (entity revs, shared revs)")
		},
	)
	rootCmd.AddCommand(couplingCmd)

	// SOC subcommand
	rootCmd.AddCommand(newCouplingCmd("soc", "Sum of coupling per entity", analysis.SumOfCoupling, nil))

	// generate-log subcommand
	rootCmd.AddCommand(newGenerateLogCmd())

	// cloc subcommand
	rootCmd.AddCommand(newClocCmd())
}
