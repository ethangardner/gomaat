package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
		RunE: func(cmd *cobra.Command, args []string) (err error) {
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

			stdout, err := gitCmd.StdoutPipe()
			if err != nil {
				return fmt.Errorf("creating git stdout pipe: %w", err)
			}

			dst := os.Stdout
			var outHandle *os.File
			if outFile != "" {
				outHandle, err = os.Create(outFile)
				if err != nil {
					return fmt.Errorf("creating output file: %w", err)
				}
				defer func() {
					if closeErr := outHandle.Close(); closeErr != nil && err == nil {
						err = closeErr
					}
				}()
				dst = outHandle
			}

			if err := gitCmd.Start(); err != nil {
				return fmt.Errorf("starting git log: %w", err)
			}

			if len(excludes) > 0 {
				err = filterExcludesStream(stdout, dst, excludes)
			} else {
				_, err = io.Copy(dst, stdout)
			}
			if err != nil {
				_ = stdout.Close()
				_ = gitCmd.Wait()
				return fmt.Errorf("processing git log output: %w", err)
			}

			if err := gitCmd.Wait(); err != nil {
				return fmt.Errorf("git log failed: %w\n%s\nCommand: git %s", err, strings.TrimSpace(stderr.String()), strings.Join(gitArgs, " "))
			}

			if outFile != "" {
				fmt.Fprintf(os.Stderr, "Log written to %s\n", outFile)
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
	var out bytes.Buffer
	if err := filterExcludesStream(bytes.NewReader(data), &out, excludes); err != nil {
		return data
	}
	return out.Bytes()
}

func filterExcludesStream(src io.Reader, dst io.Writer, excludes []string) error {
	reader := bufio.NewReader(src)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNoNewline := bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
			if !numstatLineMatchesExclude(string(lineNoNewline), excludes) {
				if _, writeErr := dst.Write(line); writeErr != nil {
					return writeErr
				}
			}
		}

		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
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
