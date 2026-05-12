package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/hhatto/gocloc"
	"github.com/spf13/cobra"

	"gomaat/internal/output"
)

func newClocCmd() *cobra.Command {
	var path string
	var byFile bool
	var excludes []string

	cmd := &cobra.Command{
		Use:   "cloc",
		Short: "Count lines of code in a directory",
		Long: `Counts lines of code, comments, and blank lines using gocloc.

Examples:
  gomaat cloc
  gomaat cloc --path /path/to/repo
  gomaat cloc --exclude vendor/ --exclude '*.pb.go'
  gomaat cloc --by-file -o results.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			processor := gocloc.NewProcessor(gocloc.NewDefinedLanguages(), gocloc.NewClocOptions())
			result, err := processor.Analyze([]string{path})
			if err != nil {
				return fmt.Errorf("cloc failed: %w", err)
			}

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
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].Code > langs[j].Code
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
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
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
