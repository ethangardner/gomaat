package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newGenerateLogCmd() *cobra.Command {
	var after string
	var path string
	var excludes []string

	cmd := &cobra.Command{
		Use:   "generate-log",
		Short: "Generate a formatted log file from a git repository",
		Long: `Runs the git log command with the predefined options to generate a formatted log file.

Examples:
  gomaat generate-log -o logfile.log
  gomaat generate-log --after 2023-01-01 -o logfile.log
  gomaat generate-log --path /path/to/repo --after 2022-06-01 -o logfile.log
  gomaat generate-log --exclude vendor/ --exclude '*.pb.go' -o logfile.log`,
		RunE: func(cmd *cobra.Command, args []string) error {
			gitArgs := []string{
				"-C", path,
				"log", "--all", "--numstat",
				"--date=short",
				"--pretty=format:--%h--%ad--%aN",
				"--no-renames",
			}
			if after != "" {
				gitArgs = append(gitArgs, "--after="+after)
			}

			var stderr strings.Builder
			gitCmd := exec.Command("git", gitArgs...)
			gitCmd.Stderr = &stderr

			out, err := gitCmd.Output()
			if err != nil {
				return fmt.Errorf("git log failed: %w\n%s\nCommand: git %s", err, stderr.String(), strings.Join(gitArgs, " "))
			}

			if len(excludes) > 0 {
				out = filterExcludes(out, excludes)
			}

			if outFile != "" {
				if err := os.WriteFile(outFile, out, 0644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Log written to %s\n", outFile)
			} else {
				_, err = os.Stdout.Write(out)
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&after, "after", "", "only include commits after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&path, "path", ".", "path to the git repository")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "exclude paths matching this pattern (repeatable, supports globs)")

	return cmd
}

// filterExcludes removes numstat lines whose path matches any exclude pattern.
// Patterns ending in "/" match directory prefixes; others are matched as globs
// against both the full path and the base filename.
func filterExcludes(data []byte, excludes []string) []byte {
	lines := strings.Split(string(data), "\n")
	out := lines[:0]
	for _, line := range lines {
		if !numstatLineMatchesExclude(line, excludes) {
			out = append(out, line)
		}
	}
	return []byte(strings.Join(out, "\n"))
}

func numstatLineMatchesExclude(line string, excludes []string) bool {
	// numstat lines: "<added>\t<deleted>\t<path>"
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return false
	}
	path := parts[2]
	for _, pattern := range excludes {
		if matchesExcludePattern(path, pattern) {
			return true
		}
	}
	return false
}

func matchesExcludePattern(path, pattern string) bool {
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern)
	}
	if matched, _ := filepath.Match(pattern, path); matched {
		return true
	}
	if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
		return true
	}
	return false
}
