package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/grouper"
	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/output"
	"github.com/ethangardner/gomaat/internal/parser"
	"github.com/ethangardner/gomaat/internal/teammapper"
)

// persistent flag values (shared across all analysis subcommands)
var (
	logFile      string
	outFile      string
	maxRows      int
	groupFile    string
	teamMapFile  string
	outputFormat string
)

// version is overridden at release build time via -ldflags (see .goreleaser.yml).
var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "gomaat",
	Short:   "Mine and analyze git history",
	Version: version,
	Long: `gomaat mines git version-control history to surface design insights:
logical coupling, code churn, authorship patterns, code age, and more.

Generate a git log first:
  gomaat generate-log --after 2023-01-01 -o logfile.log

Then run an analysis:
  gomaat authors -l logfile.log
  gomaat coupling -l logfile.log --min-coupling 30`,
}

// Execute is the entry point called from main. Cobra already prints the
// error and usage to stderr, so this only sets the exit code.
func Execute() {
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&logFile, "log", "l", "", "git log file to analyze")
	rootCmd.PersistentFlags().StringVarP(&outFile, "outfile", "o", "", "write output to file (default: stdout)")
	rootCmd.PersistentFlags().IntVarP(&maxRows, "rows", "r", 0, "max result rows (0 = no limit)")
	rootCmd.PersistentFlags().StringVarP(&groupFile, "group", "g", "", "architectural grouping spec file")
	rootCmd.PersistentFlags().StringVarP(&teamMapFile, "team-map-file", "p", "", "CSV file mapping author to team")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "format", "f", "csv", "output format: csv or json")
}

// runAnalysis is the shared execution path for all analysis subcommands.
// T is the analysis's intermediate result type (e.g. []CouplingResult or
// Summary). The type parameter ties fn's return type to format's input type,
// so the compiler rejects mismatched compute/format pairs instead of relying
// on `any` plus a runtime type assertion inside format.
func runAnalysis[T any](fn func([]model.Commit, model.Options) T, format func(T, model.Options) [][]string, opts model.Options) error {
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

	results := fn(commits, opts)
	rows := format(results, opts)

	return writeRows(rows)
}

// validateOutputFormat checks outputFormat against the formats writeRows supports.
func validateOutputFormat() error {
	if outputFormat != "csv" && outputFormat != "json" {
		return fmt.Errorf("--format (-f): expected \"csv\" or \"json\", got %q", outputFormat)
	}
	return nil
}

// parseDateFlag parses a YYYY-MM-DD flag value, returning fallback when the
// flag is unset.
func parseDateFlag(flag, value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	t, err := time.Parse(time.DateOnly, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s: expected YYYY-MM-DD, got %q", flag, value)
	}
	return t, nil
}

// writeRows writes rows to outFile (or stdout if unset) in outputFormat.
func writeRows(rows [][]string) error {
	writeFile, writeStdout := output.WriteFile, output.Write
	if outputFormat == "json" {
		writeFile, writeStdout = output.WriteJSONFile, output.WriteJSON
	}

	if outFile != "" {
		return writeFile(outFile, rows, maxRows)
	}
	return writeStdout(os.Stdout, rows, maxRows)
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

// T is inferred from fn/format and forwarded to runAnalysis; see its comment
// for why the type parameter is needed.
func newCouplingCmd[T any](use, short string, fn func([]model.Commit, model.Options) T, format func(T, model.Options) [][]string, addFlags func(*cobra.Command, *couplingFlags)) *cobra.Command {
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
			return runAnalysis(fn, format, opts)
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

// T is inferred from fn/format and forwarded to runAnalysis; see its comment
// for why the type parameter is needed.
func simpleCmd[T any](use, short string, fn func([]model.Commit, model.Options) T, format func(T, model.Options) [][]string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalysis(fn, format, model.Options{})
		},
	}
}

func init() {
	// Simple analysis subcommands
	rootCmd.AddCommand(simpleCmd("authors", "Count authors and revisions per entity", analysis.Authors, analysis.FormatAuthors))
	rootCmd.AddCommand(simpleCmd("revisions", "Count revisions per entity", analysis.Revisions, analysis.FormatRevisions))
	rootCmd.AddCommand(simpleCmd("summary", "Overview statistics for the log", analysis.Summary, analysis.FormatSummary))
	rootCmd.AddCommand(simpleCmd("identity", "Dump raw parsed commit data (debug)", analysis.Identity, analysis.FormatIdentity))
	rootCmd.AddCommand(simpleCmd("abs-churn", "Lines added/deleted aggregated by date", analysis.AbsChurn, analysis.FormatAbsChurn))
	rootCmd.AddCommand(simpleCmd("author-churn", "Lines added/deleted aggregated by author", analysis.AuthorChurn, analysis.FormatAuthorChurn))
	rootCmd.AddCommand(simpleCmd("entity-churn", "Lines added/deleted aggregated by entity", analysis.EntityChurn, analysis.FormatEntityChurn))
	rootCmd.AddCommand(simpleCmd("entity-ownership", "Churn per author per entity", analysis.EntityOwnership, analysis.FormatEntityOwnership))
	rootCmd.AddCommand(simpleCmd("main-dev", "Main developer per entity by lines added", analysis.MainDev, analysis.FormatMainDev))
	rootCmd.AddCommand(simpleCmd("refactoring-main-dev", "Main developer per entity by lines deleted", analysis.RefactoringMainDev, analysis.FormatRefactoringMainDev))
	rootCmd.AddCommand(simpleCmd("entity-effort", "Revision count per author per entity", analysis.EntityEffort, analysis.FormatEntityEffort))
	rootCmd.AddCommand(simpleCmd("main-dev-by-revs", "Main developer per entity by revision count", analysis.MainDevByRevs, analysis.FormatMainDevByRevs))
	rootCmd.AddCommand(simpleCmd("fragmentation", "Author fragmentation (fractal value) per entity", analysis.Fragmentation, analysis.FormatFragmentation))
	rootCmd.AddCommand(simpleCmd("communication", "Team communication needs based on shared code", analysis.Communication, analysis.FormatCommunication))
	rootCmd.AddCommand(simpleCmd("statistics", "Descriptive statistics for core metrics (files/lines per commit, revisions/authors/soc per entity)", analysis.Statistics, analysis.FormatStatistics))

	// Age subcommand (needs --age-time-now flag)
	var ageTimeNow string
	ageCmd := &cobra.Command{
		Use:   "age",
		Short: "Months since last modification per entity",
		RunE: func(cmd *cobra.Command, args []string) error {
			now, err := parseDateFlag("--age-time-now", ageTimeNow, time.Now())
			if err != nil {
				return err
			}
			return runAnalysis(analysis.Age, analysis.FormatAge, model.Options{AgeTimeNow: now})
		},
	}
	ageCmd.Flags().StringVarP(&ageTimeNow, "age-time-now", "d", "", "reference date for age calculation (YYYY-MM-DD, default: today)")
	rootCmd.AddCommand(ageCmd)

	// Coupling subcommand (with verbose flag)
	couplingCmd := newCouplingCmd("coupling", "Detect temporal coupling between modules", analysis.Coupling, analysis.FormatCoupling,
		func(cmd *cobra.Command, cf *couplingFlags) {
			cmd.Flags().BoolVar(&cf.verboseResults, "verbose-results", false, "include extra columns (entity revs, shared revs)")
		},
	)
	rootCmd.AddCommand(couplingCmd)

	// SOC subcommand
	rootCmd.AddCommand(newCouplingCmd("soc", "Sum of coupling per entity", analysis.SumOfCoupling, analysis.FormatSumOfCoupling, nil))

	// generate-log subcommand
	rootCmd.AddCommand(newGenerateLogCmd())

	// cloc subcommand
	rootCmd.AddCommand(newClocCmd())
}
