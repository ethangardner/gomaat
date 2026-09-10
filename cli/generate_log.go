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

	"github.com/ethangardner/gomaat/internal/model"
	"github.com/ethangardner/gomaat/internal/parser"
)

// newGenerateLogCmd's after/before/exclude/exclude-author/ignore-revs-file/
// use-mailmap flags are registered as root-persistent flags (see root.go) so
// --repo can reuse them via the same package-level vars without a second,
// divergent flag set.
func newGenerateLogCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "generate-log",
		Short: "Generate a formatted log file from a git repository",
		Long: `Runs the git log command with the predefined options to generate a formatted log file.

Examples:
  gomaat generate-log -o logfile.log
  gomaat generate-log --after 2023-01-01 -o logfile.log
  gomaat generate-log --after 2023-01-01 --before 2023-12-31 -o logfile.log
  gomaat generate-log --path /path/to/repo --after 2022-06-01 -o logfile.log
  gomaat generate-log --exclude vendor/ --exclude '*.pb.go' -o logfile.log
  gomaat generate-log --use-mailmap -o logfile.log
  gomaat generate-log --exclude-author "dependabot[bot]" --exclude-author "renovate*" -o logfile.log
  gomaat generate-log --ignore-revs-file .git-blame-ignore-revs -o logfile.log`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err := validateOutputFormat(); err != nil {
				return err
			}
			if outputFormat == "json" {
				return fmt.Errorf("--format (-f): generate-log writes raw git log text, not tabular data, so JSON output is not supported")
			}

			var ignoreRevs map[string]struct{}
			if ignoreRevsFile != "" {
				ignoreRevs, err = loadIgnoreRevs(ignoreRevsFile)
				if err != nil {
					return err
				}
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

			filters := logFilters{
				Excludes:       excludes,
				ExcludeAuthors: excludeAuthors,
				IgnoreRevs:     ignoreRevs,
				UseMailmap:     useMailmap,
			}
			if err := runGitLog(path, after, before, filters, dst); err != nil {
				return err
			}

			if outFile != "" {
				fmt.Fprintf(os.Stderr, "Log written to %s\n", outFile)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "path to the git repository")

	return cmd
}

// logFilters bundles the git-log filtering knobs runGitLog accepts beyond a
// plain path/date range.
type logFilters struct {
	Excludes       []string
	ExcludeAuthors []string
	IgnoreRevs     map[string]struct{}
	UseMailmap     bool
}

// runGitLog runs git log against path and streams output to dst.
func runGitLog(path, after, before string, filters logFilters, dst io.Writer) error {
	gitArgs := []string{
		"-C", path,
		"log", "--all", "--numstat",
		"--date=short",
		"--pretty=format:--%H--%ad--%aN",
		"--no-renames",
		"--no-merges",
	}
	if filters.UseMailmap {
		gitArgs = append(gitArgs, "--use-mailmap")
	}
	if after != "" {
		gitArgs = append(gitArgs, "--after="+after)
	}
	if before != "" {
		gitArgs = append(gitArgs, "--before="+before)
	}
	gitArgs = append(gitArgs, buildExcludePathspecArgs(filters.Excludes)...)

	var stderr strings.Builder
	gitCmd := exec.Command("git", gitArgs...)
	gitCmd.Stderr = &stderr

	stdout, err := gitCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating git stdout pipe: %w", err)
	}

	if err := gitCmd.Start(); err != nil {
		return fmt.Errorf("starting git log: %w", err)
	}

	if len(filters.Excludes) > 0 || len(filters.ExcludeAuthors) > 0 || len(filters.IgnoreRevs) > 0 {
		err = filterLog(stdout, dst, filters.Excludes, filters.ExcludeAuthors, filters.IgnoreRevs)
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
	return nil
}

// runGitLogToCommits runs git log against path with filters and parses the
// result straight into commits, for --repo's direct-analysis shortcut. It
// writes git's (filtered) output to a temporary file and removes it before
// returning, rather than streaming through an in-memory pipe concurrently
// with parsing — simpler to get right, and satisfies --repo's "no
// intermediate file left behind" requirement just as well.
func runGitLogToCommits(path, after, before string, filters logFilters) ([]model.Commit, error) {
	tmp, err := os.CreateTemp("", "gomaat-repo-log-*.log")
	if err != nil {
		return nil, fmt.Errorf("creating temporary log file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	runErr := runGitLog(path, after, before, filters, tmp)
	closeErr := tmp.Close()
	if runErr != nil {
		return nil, runErr
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing temporary log file: %w", closeErr)
	}

	return parser.ParseFile(tmpPath)
}

// loadIgnoreRevs reads a newline-separated list of commit SHAs, in the same
// format git blame --ignore-revs-file uses: blank lines and lines starting
// with '#' are skipped.
func loadIgnoreRevs(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening ignore-revs file: %w", err)
	}
	defer func() { _ = f.Close() }()

	revs := make(map[string]struct{})
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		revs[line] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading ignore-revs file: %w", err)
	}
	return revs, nil
}

func buildExcludePathspecArgs(excludes []string) []string {
	var args []string
	for _, pattern := range excludes {
		if strings.HasSuffix(pattern, "/") {
			if len(args) == 0 {
				args = []string{"--", "."}
			}
			args = append(args, ":(exclude,literal)"+pattern)
		}
	}
	return args
}

// filterLog streams src to dst, dropping whole commits whose author matches
// excludeAuthors or whose rev appears in ignoreRevs, and dropping numstat
// lines under a kept commit that match excludes. Author and rev only appear
// on a commit's header line ("--<rev>--<date>--<author>"), so keep/drop is
// decided there and applied to every following line until the next header —
// only one commit's worth of state is held at a time, never the whole log.
func filterLog(src io.Reader, dst io.Writer, excludes, excludeAuthors []string, ignoreRevs map[string]struct{}) error {
	reader := bufio.NewReader(src)
	keepCurrent := true
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmed := string(bytes.TrimRight(line, "\r\n"))
			if rev, author, ok := parseHeaderLine(trimmed); ok {
				_, ignored := ignoreRevs[rev]
				keepCurrent = !ignored && !authorExcluded(author, excludeAuthors)
			}
			if keepCurrent && !numstatLineMatchesExclude(trimmed, excludes) {
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

// parseHeaderLine extracts rev and author from a commit header line
// ("--<rev>--<date>--<author>"), mirroring internal/parser's own header
// parsing so filtering agrees with what the parser will later see. ok is
// false for any other kind of line (numstat or blank).
func parseHeaderLine(line string) (rev, author string, ok bool) {
	if !strings.HasPrefix(line, "--") {
		return "", "", false
	}
	parts := strings.SplitN(line, "--", 4)
	if len(parts) != 4 {
		return "", "", false
	}
	return parts[1], parts[3], true
}

func authorExcluded(author string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesAuthorPattern(author, pattern) {
			return true
		}
	}
	return false
}

// matchesAuthorPattern reports whether author matches pattern. Only '*' is
// treated as a wildcard (unlike filepath.Match's '?' and '[...]' character
// classes) because author display names commonly contain literal square
// brackets (e.g. GitHub's "dependabot[bot]"), which filepath.Match would
// otherwise try to interpret as a character class and fail to match.
func matchesAuthorPattern(author, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return author == pattern
	}
	segments := strings.Split(pattern, "*")
	if !strings.HasPrefix(author, segments[0]) {
		return false
	}
	author = author[len(segments[0]):]
	for _, seg := range segments[1 : len(segments)-1] {
		idx := strings.Index(author, seg)
		if idx < 0 {
			return false
		}
		author = author[idx+len(seg):]
	}
	return strings.HasSuffix(author, segments[len(segments)-1])
}

func numstatLineMatchesExclude(line string, excludes []string) bool {
	// numstat lines: "<added>\t<deleted>\t<path>"
	parts := strings.SplitN(line, "\t", 3)
	return len(parts) == 3 && pathMatchesAnyExclude(parts[2], excludes)
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
