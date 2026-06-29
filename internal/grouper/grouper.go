package grouper

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ethangardner/gomaat/internal/model"
)

type group struct {
	pattern *regexp.Regexp
	name    string
}

// LoadFile reads a grouping spec file and returns a slice of (pattern, name) pairs.
// File format (one rule per line):
//
//	some/path => GroupName
//	^some-regexp$ => GroupName
//
// Plain paths are matched as prefix: ^some/path/
// Lines starting with # or blank lines are ignored.
func LoadFile(path string) ([]group, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening group file: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "error closing file %s: %v\n", path, err)
		}
	}(f)
	return load(f)
}

func load(r io.Reader) ([]group, error) {
	var groups []group
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		before, after, found := strings.Cut(line, "=>")
		if !found {
			return nil, fmt.Errorf("invalid group spec line: %q", line)
		}
		rawPath := strings.TrimSpace(before)
		name := strings.TrimSpace(after)

		pattern := "^" + regexp.QuoteMeta(rawPath) + "/"
		if strings.HasPrefix(rawPath, "^") {
			pattern = rawPath
		}
		pat, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", rawPath, err)
		}
		groups = append(groups, group{pat, name})
	}
	return groups, scanner.Err()
}

// Apply remaps each commit's Entity to its group name.
// Commits that don't match any group are discarded.
func Apply(commits []model.Commit, groups []group) []model.Commit {
	if len(groups) == 0 {
		return commits
	}
	out := make([]model.Commit, 0, len(commits))
	for _, c := range commits {
		if name := match(c.Entity, groups); name != "" {
			c.Entity = name
			out = append(out, c)
		}
	}
	return out
}

func match(entity string, groups []group) string {
	for _, g := range groups {
		if g.pattern.MatchString(entity) {
			return g.name
		}
	}
	return ""
}
