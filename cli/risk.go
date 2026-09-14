package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ethangardner/gomaat/internal/analysis"
	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/parser"
)

func newRiskCmd() *cobra.Command {
	var staged bool
	var diffRef string
	var path string
	var minCoupling float64

	cmd := &cobra.Command{
		Use:   "risk",
		Short: "Flag historically-coupled files missing from a staged change or diff",
		Long: `Cross-references a staged change (or a diff against another ref) with
gomaat's temporal coupling analysis, flagging files that historically
co-change with what you touched but aren't part of this change.

Like cloc, risk reads live repository state directly via git in addition to
-l/--log for historical coupling data: its changed-file set reflects the
working tree at the moment it runs, not the log file.

Exits non-zero when a finding exists, so it can be used as a pre-commit hook
or CI check.

Examples:
  gomaat risk -l logfile.log --staged
  gomaat risk -l logfile.log --diff main
  gomaat risk -l logfile.log --staged --min-coupling 60`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if staged == (diffRef != "") {
				return fmt.Errorf("specify exactly one of --staged or --diff <ref>")
			}
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

			opts := model.Options{
				MinRevs:          5,
				MinSharedRevs:    5,
				MinCoupling:      minCoupling,
				MaxCoupling:      100,
				MaxChangesetSize: 30,
			}
			coupling := analysis.Coupling(commits, opts)

			diffArg := "--staged"
			if diffRef != "" {
				diffArg = diffRef
			}
			changed, err := gitChangedFiles(path, diffArg)
			if err != nil {
				return err
			}

			results := analysis.Risk(coupling, changed)
			if err := writeRows(analysis.FormatRisk(results, model.Options{})); err != nil {
				return err
			}

			if len(results) == 0 {
				fmt.Fprintln(os.Stderr, "risk: no concerns")
				return nil
			}
			fmt.Fprintf(os.Stderr, "risk: %d finding(s) at >= %.0f%% coupling — see output\n", len(results), minCoupling)
			return fmt.Errorf("risk: staged/diff change is missing %d historically-coupled file(s)", len(results))
		},
	}
	cmd.SilenceUsage = true

	cmd.Flags().BoolVar(&staged, "staged", false, "check files staged for commit (git diff --staged)")
	cmd.Flags().StringVar(&diffRef, "diff", "", "check files changed relative to this ref (git diff <ref>)")
	cmd.Flags().StringVar(&path, "path", ".", "path to the git repository")
	cmd.Flags().Float64Var(&minCoupling, "min-coupling", 50, "minimum coupling percentage to report as a risk")

	return cmd
}

// gitChangedFiles resolves path's repo root, then runs
// `git -C <repoRoot> diff <diffArg> --name-only`, returning paths relative
// to repoRoot (same convention as model.Commit.Entity and cloc's tracked
// files).
func gitChangedFiles(path, diffArg string) ([]string, error) {
	repoRoot, err := resolveRepoRoot(path)
	if err != nil {
		return nil, err
	}
	out, err := exec.Command("git", "-C", repoRoot, "diff", diffArg, "--name-only").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
