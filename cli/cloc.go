package cli

import (
	"cmp"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/hhatto/gocloc"
	"github.com/spf13/cobra"

	"gomaat/internal/output"
)

func newClocProcessor() *gocloc.Processor {
	return gocloc.NewProcessor(gocloc.NewDefinedLanguages(), gocloc.NewClocOptions())
}

func newClocCmd() *cobra.Command {
	var path string
	var byFile bool
	var excludes []string

	cmd := &cobra.Command{
		Use:   "cloc",
		Short: "Count lines of code in a directory",
		Long: `Counts lines of code, comments, and blank lines for git-tracked files using gocloc.

Examples:
  gomaat cloc
  gomaat cloc --path /path/to/repo
  gomaat cloc --exclude vendor/ --exclude '*.pb.go'
  gomaat cloc --by-file -o results.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			trackedFiles, repoRoot, err := gitTrackedFiles(path)
			if err != nil {
				return err
			}
			if len(trackedFiles) == 0 {
				return fmt.Errorf("no git-tracked files found in %s", path)
			}

			result, err := newClocProcessor().Analyze(trackedFiles)
			if err != nil {
				return fmt.Errorf("cloc failed: %w", err)
			}

			relativizeResult(result, repoRoot)

			if len(excludes) > 0 {
				applyClocExcludes(result, excludes)
			}

			var rows [][]string
			if byFile {
				rows = clocFileRows(result)
			} else {
				rows = clocLanguageRows(result)
			}

			if outFile != "" {
				return output.WriteFile(outFile, rows, maxRows)
			}
			return output.Write(os.Stdout, rows, maxRows)
		},
	}

	cmd.Flags().StringVar(&path, "path", ".", "path to analyze")
	cmd.Flags().BoolVar(&byFile, "by-file", false, "show results per file instead of per language")
	cmd.Flags().StringArrayVar(&excludes, "exclude", nil, "exclude paths matching this pattern (repeatable, supports globs)")

	return cmd
}

// gitTrackedFiles returns the absolute paths of all files tracked by git under
// path, along with the resolved absolute path used as the repo root.
func gitTrackedFiles(path string) ([]string, string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", err
	}

	repoRootOut, err := exec.Command("git", "-C", absPath, "rev-parse", "--show-toplevel").CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("git rev-parse --show-toplevel failed: %w: %s", err, strings.TrimSpace(string(repoRootOut)))
	}
	repoRoot := strings.TrimSpace(string(repoRootOut))
	if repoRoot == "" {
		return nil, "", fmt.Errorf("git rev-parse --show-toplevel returned empty repository root")
	}

	out, err := exec.Command("git", "-C", repoRoot, "ls-files").CombinedOutput()
	if err != nil {
		return nil, "", fmt.Errorf("git ls-files failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var files []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			files = append(files, filepath.Join(repoRoot, line))
		}
	}
	return files, repoRoot, nil
}

// relativizeResult rewrites all file paths in result to be relative to root.
// This must be called before applyClocExcludes so patterns like "vendor/" match.
func relativizeResult(result *gocloc.Result, root string) {
	// Rebuild under relative-path keys rather than mutating in place (can't
	// safely rewrite map keys while ranging); pre-size to the 1:1 entry count.
	newFiles := make(map[string]*gocloc.ClocFile, len(result.Files))
	for absPath, f := range result.Files {
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			rel = absPath
		}
		f.Name = rel
		newFiles[rel] = f
	}
	result.Files = newFiles

	for _, lang := range result.Languages {
		for i, f := range lang.Files {
			if rel, err := filepath.Rel(root, f); err == nil {
				lang.Files[i] = rel
			}
		}
	}
}

// applyClocExcludes removes excluded files from result and adjusts language and total counts.
func applyClocExcludes(result *gocloc.Result, excludes []string) {
	for filePath, f := range result.Files {
		if !pathMatchesAnyExclude(filePath, excludes) {
			continue
		}
		lang := result.Languages[f.Lang]
		lang.Code -= f.Code
		lang.Blanks -= f.Blanks
		lang.Comments -= f.Comments
		result.Total.Code -= f.Code
		result.Total.Blanks -= f.Blanks
		result.Total.Comments -= f.Comments
		result.Total.Total--
		delete(result.Files, filePath)
	}

	for _, lang := range result.Languages {
		kept := lang.Files[:0]
		for _, f := range lang.Files {
			if !pathMatchesAnyExclude(f, excludes) {
				kept = append(kept, f)
			}
		}
		lang.Files = kept
	}
}

func pathMatchesAnyExclude(path string, excludes []string) bool {
	for _, pattern := range excludes {
		if matchesExcludePattern(path, pattern) {
			return true
		}
	}
	return false
}

func clocLanguageRows(result *gocloc.Result) [][]string {
	rows := [][]string{{"Language", "Files", "Blank", "Comment", "Code"}}

	langs := make([]*gocloc.Language, 0, len(result.Languages))
	for _, lang := range result.Languages {
		if len(lang.Files) > 0 {
			langs = append(langs, lang)
		}
	}
	slices.SortFunc(langs, func(a, b *gocloc.Language) int {
		return cmp.Compare(b.Code, a.Code)
	})

	for _, lang := range langs {
		rows = append(rows, []string{
			lang.Name,
			strconv.Itoa(len(lang.Files)),
			strconv.Itoa(int(lang.Blanks)),
			strconv.Itoa(int(lang.Comments)),
			strconv.Itoa(int(lang.Code)),
		})

	}

	t := result.Total
	rows = append(rows, []string{
		"TOTAL",
		strconv.Itoa(int(t.Total)),
		strconv.Itoa(int(t.Blanks)),
		strconv.Itoa(int(t.Comments)),
		strconv.Itoa(int(t.Code)),
	})

	return rows
}

func clocFileRows(result *gocloc.Result) [][]string {
	rows := [][]string{{"File", "Language", "Blank", "Comment", "Code"}}

	files := make([]*gocloc.ClocFile, 0, len(result.Files))
	for _, f := range result.Files {
		files = append(files, f)
	}
	slices.SortFunc(files, func(a, b *gocloc.ClocFile) int {
		return cmp.Compare(a.Name, b.Name)
	})

	for _, f := range files {
		rows = append(rows, []string{
			f.Name,
			f.Lang,
			strconv.Itoa(int(f.Blanks)),
			strconv.Itoa(int(f.Comments)),
			strconv.Itoa(int(f.Code)),
		})
	}

	return rows
}
