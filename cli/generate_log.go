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
	var before string
	var path string
	var excludes []string
	var excludeAuthors []string
	var ignoreRevsFile string
	var useMailmap bool

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

	cmd.Flags().StringVar(&after, "after", "", "only include commits after this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "only include commits before this date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&path, "path", ".", "path to the git repository")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "exclude paths matching this pattern (repeatable, supports globs)")
	cmd.Flags().StringArrayVar(&excludeAuthors, "exclude-author", nil, "exclude commits by this author name (repeatable, supports globs, case-sensitive)")
	cmd.Flags().StringVar(&ignoreRevsFile, "ignore-revs-file", "", "drop commits listed in this file (one SHA per line, '#' comments; same format as git blame --ignore-revs-file)")
	cmd.Flags().BoolVar(&useMailmap, "use-mailmap", false, "resolve author identities via .mailmap (requires a mailmap file at the repo root; no-op otherwise)")

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
		"--pretty=format:%x00%H%x00%ad%x00%aN%x00%B%x00",
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

// filterLog reads all of src, dropping whole commits whose author matches
// excludeAuthors or whose rev appears in ignoreRevs, and dropping numstat
// lines under a kept commit that match excludes, then writes the surviving
// commits to dst in the same NUL-delimited record format they arrived in.
//
// Commit records are NUL-delimited (see internal/parser's doc comment for
// the exact framing), so filtering can't be done line-by-line the way it was
// under the old single-line-header format: a commit message can legitimately
// span many lines, including blank lines. Splitting the whole input on NUL
// bytes instead mirrors internal/parser's own field-grouping exactly, so
// filtering agrees with what the parser will later see.
func filterLog(src io.Reader, dst io.Writer, excludes, excludeAuthors []string, ignoreRevs map[string]struct{}) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	fields := bytes.Split(data, []byte{0})
	fields = fields[1:] // drop the empty prefix before the first commit's leading NUL

	for i := 0; i+4 < len(fields); i += 5 {
		rev := string(fields[i])
		date := fields[i+1]
		author := string(fields[i+2])
		message := fields[i+3]
		numstatBlock := fields[i+4]

		if _, ignored := ignoreRevs[rev]; ignored || authorExcluded(author, excludeAuthors) {
			continue
		}

		if _, err := dst.Write([]byte{0}); err != nil {
			return err
		}
		if _, err := io.WriteString(dst, rev); err != nil {
			return err
		}
		if _, err := dst.Write([]byte{0}); err != nil {
			return err
		}
		if _, err := dst.Write(date); err != nil {
			return err
		}
		if _, err := dst.Write([]byte{0}); err != nil {
			return err
		}
		if _, err := io.WriteString(dst, author); err != nil {
			return err
		}
		if _, err := dst.Write([]byte{0}); err != nil {
			return err
		}
		if _, err := dst.Write(message); err != nil {
			return err
		}
		if _, err := dst.Write([]byte{0}); err != nil {
			return err
		}
		if _, err := dst.Write(filterNumstatBlock(numstatBlock, excludes)); err != nil {
			return err
		}
	}
	return nil
}

// filterNumstatBlock drops numstat lines matching excludes from a commit's
// raw numstat block, preserving blank lines (which the parser treats as
// separators, not data) unchanged.
func filterNumstatBlock(block []byte, excludes []string) []byte {
	if len(excludes) == 0 {
		return block
	}
	lines := strings.Split(string(block), "\n")
	kept := lines[:0]
	for _, line := range lines {
		if numstatLineMatchesExclude(line, excludes) {
			continue
		}
		kept = append(kept, line)
	}
	return []byte(strings.Join(kept, "\n"))
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
