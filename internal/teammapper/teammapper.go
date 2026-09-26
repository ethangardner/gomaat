package teammapper

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/ethangardner/gomaat/internal/loadfile"
	"github.com/ethangardner/gomaat/internal/model"
)

// LoadFile reads a CSV file mapping author → team.
// Expected format (header optional): author,team
func LoadFile(path string) (map[string]string, error) {
	return loadfile.Parse(path, "team map", load)
}

func load(r io.Reader) (map[string]string, error) {
	lookup := map[string]string{}
	err := readRecords(r, "team map", false, func(record []string) {
		if len(record) < 2 {
			return
		}
		author := strings.TrimSpace(record[0])
		team := strings.TrimSpace(record[1])
		// skip header row
		if strings.EqualFold(author, "author") && strings.EqualFold(team, "team") {
			return
		}
		lookup[author] = team
	})
	return lookup, err
}

// LoadAuthorsFile reads a list of author names, one per line. The file may
// also be a CSV whose first column is the author (header optional); other
// columns are ignored. Names containing a comma must be double-quoted; a
// quote inside an unquoted name is read literally.
func LoadAuthorsFile(path string) (map[string]struct{}, error) {
	return loadfile.Parse(path, "authors", loadAuthors)
}

func loadAuthors(r io.Reader) (map[string]struct{}, error) {
	authors := map[string]struct{}{}
	err := readRecords(r, "authors", true, func(record []string) {
		author := strings.TrimSpace(record[0])
		// skip header row and blank names
		if author == "" || strings.EqualFold(author, "author") {
			return
		}
		authors[author] = struct{}{}
	})
	return authors, err
}

// readRecords calls fn for each CSV record in r, skipping '#' comment lines.
// By default every record must have as many fields as the first; lenient
// allows differing field counts and bare quotes in unquoted fields.
func readRecords(r io.Reader, what string, lenient bool, fn func([]string)) error {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.Comment = '#'
	if lenient {
		cr.FieldsPerRecord = -1
		cr.LazyQuotes = true
	}
	for {
		record, err := cr.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading %s: %w", what, err)
		}
		fn(record)
	}
}

// Apply replaces each commit's Author with its team name.
// Commits for unmapped authors are discarded.
func Apply(commits []model.Commit, lookup map[string]string) []model.Commit {
	if len(lookup) == 0 {
		return commits
	}
	out := make([]model.Commit, 0, len(commits))
	for _, c := range commits {
		if team, ok := lookup[c.Author]; ok {
			c.Author = team
			out = append(out, c)
		}
	}
	return out
}
