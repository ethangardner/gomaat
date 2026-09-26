package parser

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/ethangardner/gomaat/internal/model"
)

// FuzzParseReader checks that any input parses without panicking, and that
// rendering the parsed commits back into the log format and parsing that
// again gives the same commits.
func FuzzParseReader(f *testing.F) {
	for _, seed := range []string{
		sampleLog,
		"",
		"--abc--2024-01-01--Alice--with--dashes\n1\t2\tpath with\ttab\n",
		"--abc--2024-01-01--Alice\n-\t-\timage.png\n\n--def--2024-01-02--Bob\n+3\tx\t  spaced.go  \n",
		"1\t2\torphan-before-any-header.go\n----\n--a--b\n",
		"--r--d--a\r\n5\t5\tcrlf.go\r\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		commits, err := ParseReader(strings.NewReader(input))
		if err != nil {
			// Only bufio.Scanner's line-length limit can fail, and the
			// fuzzer's inputs are too short to hit it.
			t.Fatalf("ParseReader: %v", err)
		}
		for _, c := range commits {
			if c.Rev == "" {
				t.Fatalf("commit with empty rev: %+v", c)
			}
			if c.Entity != strings.TrimSpace(c.Entity) {
				t.Fatalf("entity not trimmed: %q", c.Entity)
			}
		}

		again, err := ParseReader(strings.NewReader(render(commits)))
		if err != nil {
			t.Fatalf("reparsing rendered log: %v", err)
		}
		if !slices.Equal(commits, again) {
			t.Fatalf("round trip changed commits:\n got %+v\nwant %+v", again, commits)
		}
	})
}

// render writes commits in the generate-log format, one header per run of
// commits sharing a rev, date and author.
func render(commits []model.Commit) string {
	var b strings.Builder
	var prev model.Commit
	for i, c := range commits {
		if i == 0 || c.Rev != prev.Rev || c.Date != prev.Date || c.Author != prev.Author {
			fmt.Fprintf(&b, "--%s--%s--%s\n", c.Rev, c.Date, c.Author)
		}
		fmt.Fprintf(&b, "%d\t%d\t%s\n", c.LocAdded, c.LocDeleted, c.Entity)
		prev = c
	}
	return b.String()
}
