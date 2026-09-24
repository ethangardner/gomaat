// Package gitdiff parses the patch output of
//
//	git log -p -U0 --no-renames --format=%x00%H%x00%ct%x00%aN%x00%P
//
// into per-commit, per-file hunks. Unlike internal/parser, which only sees
// numstat line counts, this keeps the text of every added and deleted line,
// for analyses that need to know *which* lines changed (e.g. rework).
package gitdiff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"
	"time"
)

// Format is the --format string Parse expects. Each field is prefixed with a
// NUL byte, which git never emits at the start of a diff line (a file whose
// content contains NUL is treated as binary), so a header line can never be
// confused with patch content.
const Format = "%x00%H%x00%ct%x00%aN%x00%P"

// Commit is one commit's patch.
type Commit struct {
	Rev     string
	Time    time.Time // committer date
	Author  string
	Parents []string
	Files   []FileDiff
}

// FileDiff is the change to one path within a commit.
type FileDiff struct {
	Path    string
	Removed bool // the file was deleted
	Binary  bool // git reported "Binary files ... differ"; Hunks is empty
	Hunks   []Hunk
}

// Hunk is one zero-context (-U0) hunk. OldStart/NewStart follow git's
// unified-diff convention: when a side has no lines, its start is the line
// *after which* the change applies (0 for the top of the file).
type Hunk struct {
	OldStart int
	NewStart int
	Deleted  []string
	Added    []string
}

// Parse streams commits from r. Iteration stops at the first error, which is
// yielded with a zero Commit.
func Parse(r io.Reader) iter.Seq2[Commit, error] {
	return func(yield func(Commit, error) bool) {
		p := parser{r: bufio.NewReader(r)}
		for {
			c, err := p.next()
			if errors.Is(err, io.EOF) {
				return
			}
			if !yield(c, err) || err != nil {
				return
			}
		}
	}
}

type parser struct {
	r       *bufio.Reader
	pending string // header line read ahead while finishing the previous commit
	lineNo  int
}

func (p *parser) readLine() (string, error) {
	line, err := p.r.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return "", err
	}
	p.lineNo++
	return strings.TrimSuffix(line, "\n"), nil
}

func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("gitdiff: line %d: %s", p.lineNo, fmt.Sprintf(format, args...))
}

// next returns the next commit, or io.EOF when the input is exhausted.
func (p *parser) next() (Commit, error) {
	header := p.pending
	p.pending = ""
	for header == "" {
		line, err := p.readLine()
		if err != nil {
			return Commit{}, err
		}
		if strings.HasPrefix(line, "\x00") {
			header = line
		}
	}

	c, err := p.parseHeader(header)
	if err != nil {
		return Commit{}, err
	}

	var file *FileDiff
	for {
		line, err := p.readLine()
		if errors.Is(err, io.EOF) {
			return c, nil
		}
		if err != nil {
			return Commit{}, err
		}

		switch {
		case strings.HasPrefix(line, "\x00"):
			p.pending = line
			return c, nil
		case strings.HasPrefix(line, "diff --git "):
			c.Files = append(c.Files, FileDiff{Path: pathFromDiffGit(line[len("diff --git "):])})
			file = &c.Files[len(c.Files)-1]
		case file == nil:
			// blank separator between the header and the first diff
		case strings.HasPrefix(line, "--- "):
			if path, ok := stripPrefix(line[len("--- "):], "a/"); ok {
				file.Path = path
			}
		case strings.HasPrefix(line, "+++ "):
			if path, ok := stripPrefix(line[len("+++ "):], "b/"); ok {
				file.Path = path
			} else {
				file.Removed = true
			}
		case strings.HasPrefix(line, "Binary files "):
			file.Binary = true
		case strings.HasPrefix(line, "@@ "):
			h, err := p.parseHunk(line)
			if err != nil {
				return Commit{}, err
			}
			file.Hunks = append(file.Hunks, h)
		}
		// Anything else (index, mode, "\ No newline at end of file") carries
		// no line content and is ignored.
	}
}

// parseHeader parses a "\x00<rev>\x00<unix-time>\x00<author>\x00<parents>" line.
func (p *parser) parseHeader(line string) (Commit, error) {
	fields := strings.Split(line[1:], "\x00")
	if len(fields) != 4 {
		return Commit{}, p.errorf("malformed commit header %q", line)
	}
	secs, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return Commit{}, p.errorf("malformed commit time %q", fields[1])
	}
	c := Commit{Rev: fields[0], Time: time.Unix(secs, 0).UTC(), Author: fields[2]}
	if fields[3] != "" {
		c.Parents = strings.Fields(fields[3])
	}
	return c, nil
}

// parseHunk parses a "@@ -a[,b] +c[,d] @@" header and consumes exactly the b
// deleted and d added lines that follow. Reading by count rather than by
// prefix is what keeps a deleted line such as "-- foo" from being mistaken
// for a "--- a/path" file header.
func (p *parser) parseHunk(header string) (Hunk, error) {
	fields := strings.Fields(header)
	if len(fields) < 4 || fields[3] != "@@" {
		return Hunk{}, p.errorf("malformed hunk header %q", header)
	}
	oldStart, oldCount, err1 := parseRange(fields[1], '-')
	newStart, newCount, err2 := parseRange(fields[2], '+')
	if err := errors.Join(err1, err2); err != nil {
		return Hunk{}, p.errorf("malformed hunk header %q: %v", header, err)
	}

	h := Hunk{OldStart: oldStart, NewStart: newStart}
	for len(h.Deleted) < oldCount || len(h.Added) < newCount {
		line, err := p.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return Hunk{}, fmt.Errorf("gitdiff: reading hunk %q: %w", header, err)
		}
		switch {
		case strings.HasPrefix(line, "-") && len(h.Deleted) < oldCount:
			h.Deleted = append(h.Deleted, line[1:])
		case strings.HasPrefix(line, "+") && len(h.Added) < newCount:
			h.Added = append(h.Added, line[1:])
		case strings.HasPrefix(line, `\`):
			// "\ No newline at end of file"
		default:
			return Hunk{}, p.errorf("unexpected line in hunk %q: %q", header, line)
		}
	}
	return h, nil
}

// parseRange parses "-a,b" / "+c,d" (the count defaults to 1 when omitted).
func parseRange(s string, sign byte) (start, count int, err error) {
	if s == "" || s[0] != sign {
		return 0, 0, fmt.Errorf("range %q must start with %q", s, sign)
	}
	startStr, countStr, hasCount := strings.Cut(s[1:], ",")
	if start, err = strconv.Atoi(startStr); err != nil {
		return 0, 0, err
	}
	count = 1
	if hasCount {
		if count, err = strconv.Atoi(countStr); err != nil {
			return 0, 0, err
		}
	}
	return start, count, nil
}

// stripPrefix unquotes a ---/+++ path and removes its a/ or b/ prefix. ok is
// false for /dev/null. Git terminates an unquoted path containing a space
// with a tab, so that is trimmed first.
func stripPrefix(s, prefix string) (string, bool) {
	s = unquote(strings.TrimSuffix(s, "\t"))
	if s == "/dev/null" {
		return "", false
	}
	return strings.TrimPrefix(s, prefix), true
}

// pathFromDiffGit extracts the path from a "diff --git a/P b/P" line's
// arguments. It's only the fallback for entries without ---/+++ lines
// (binary or mode-only changes). --no-renames guarantees both sides name the
// same path, which is what makes an unquoted path containing spaces
// splittable: the arguments are exactly "a/" + P + " b/" + P.
func pathFromDiffGit(args string) string {
	if strings.HasPrefix(args, `"`) {
		if quoted, err := strconv.QuotedPrefix(args); err == nil {
			return strings.TrimPrefix(unquote(quoted), "a/")
		}
	}
	n := (len(args) - len("a/ b/")) / 2
	if n <= 0 {
		return args
	}
	return args[len("a/") : len("a/")+n]
}

// unquote undoes git's C-style quoting of paths with unusual characters,
// which Go's string-literal syntax is compatible with.
func unquote(s string) string {
	if !strings.HasPrefix(s, `"`) {
		return s
	}
	if u, err := strconv.Unquote(s); err == nil {
		return u
	}
	return s
}
