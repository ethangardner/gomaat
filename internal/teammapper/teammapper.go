package teammapper

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ethangardner/gomaat/internal/model"
)

// LoadFile reads a CSV file mapping author → team.
// Expected format (header optional): author,team
func LoadFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening team map file: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error closing team map file %s: %v\n", path, err)
		}
	}(f)
	return load(f)
}

func load(r io.Reader) (map[string]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.Comment = '#'

	lookup := map[string]string{}
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading team map: %w", err)
		}
		if len(record) < 2 {
			continue
		}
		author := strings.TrimSpace(record[0])
		team := strings.TrimSpace(record[1])
		// skip header row
		if strings.EqualFold(author, "author") && strings.EqualFold(team, "team") {
			continue
		}
		lookup[author] = team
	}
	return lookup, nil
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
