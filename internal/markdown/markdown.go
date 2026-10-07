// Package markdown renders analysis output as Markdown for report-style
// commands like brief.
package markdown

import (
	"fmt"
	"slices"
	"strings"
)

// none stands in for an empty table or list, so an empty section still reads
// as a sentence rather than a bare header.
const none = "_None._\n"

// escaper backslash-escapes the characters that would turn a file or author
// name into markup (e.g. "__init__.py" into bold "init.py") or, for "|", end a
// table cell early.
var escaper = strings.NewReplacer(`\`, `\\`, "|", `\|`, "*", `\*`, "_", `\_`, "`", "\\`", "<", `\<`, "[", `\[`, "]", `\]`)

// Table renders rows as a GitHub-flavored Markdown table. The first row is
// the header, as with every analysis's FormatXXX output. If limit > 0, at
// most limit data rows are rendered. Cells are escaped with escaper.
func Table(rows [][]string, limit int) string {
	if len(rows) < 2 {
		return none
	}
	var b strings.Builder
	writeRow(&b, rows[0])
	writeRow(&b, slices.Repeat([]string{"---"}, len(rows[0])))
	for _, r := range truncate(rows[1:], limit) {
		writeRow(&b, r)
	}
	return b.String()
}

// List renders items as a numbered Markdown list, capped at limit items if
// limit > 0. Items are escaped with escaper.
func List(items []string, limit int) string {
	if len(items) == 0 {
		return none
	}
	var b strings.Builder
	for i, item := range truncate(items, limit) {
		fmt.Fprintf(&b, "%d. %s\n", i+1, escaper.Replace(item))
	}
	return b.String()
}

func truncate[T any](s []T, limit int) []T {
	if limit > 0 && len(s) > limit {
		return s[:limit]
	}
	return s
}

func writeRow(b *strings.Builder, cells []string) {
	b.WriteString("|")
	for _, c := range cells {
		b.WriteString(" " + escaper.Replace(c) + " |")
	}
	b.WriteString("\n")
}
