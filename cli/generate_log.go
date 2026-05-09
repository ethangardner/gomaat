package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func newGenerateLogCmd() *cobra.Command {
	var after string
	var repo string
	var outputFile string

	cmd := &cobra.Command{
		Use:   "generate-log",
		Short: "Generate a git2-format log from a git repository",
		Long: `Runs the git log command in git2 format so you don't have to remember the syntax.

Examples:
  gomaat generate-log -o logfile.log
  gomaat generate-log --after 2023-01-01 -o logfile.log
  gomaat generate-log --repo /path/to/repo --after 2022-06-01 -o logfile.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gitArgs := []string{
				"-C", repo,
				"log", "--all", "--numstat",
				"--date=short",
				"--pretty=format:--%h--%ad--%aN",
				"--no-renames",
			}
			if after != "" {
				gitArgs = append(gitArgs, "--after="+after)
			}

			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stderr = os.Stderr

			out, err := gitCmd.Output()
			if err != nil {
				return fmt.Errorf("git log failed: %w\nCommand: git %s", err, strings.Join(gitArgs, " "))
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, out, 0644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Log written to %s\n", outputFile)
			} else {
				_, err = os.Stdout.Write(out)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "only include commits after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&repo, "repo", ".", "path to the git repository")
	cmd.Flags().StringVar(&outputFile, "output", "", "write log to this file (default: stdout)")

	return cmd
}
