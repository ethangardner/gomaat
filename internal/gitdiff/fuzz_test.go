package gitdiff

import (
	"strings"
	"testing"
)

// FuzzParse checks that any input parses without panicking or looping, that
// iteration stops at the first error, and that header fields and line text
// never carry the separators they were split on.
func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"",
		"\x00aaa\x001700000000\x00Alice\x00\n\ndiff --git a/new.go b/new.go\n" +
			"--- /dev/null\n+++ b/new.go\n@@ -0,0 +1,2 @@\n+package x\n+-- looks like a header\n",
		"\x00bbb\x001700086400\x00Bob Smith\x00aaa\n\ndiff --git a/f b/f\n--- a/f\n+++ b/f\n" +
			"@@ -2 +2 @@ func\n--- old\n+++ new\n@@ -3,0 +4 @@\n+tail\n\\ No newline at end of file\n",
		"\x00c\x001\x00C\x00a b\ndiff --git a/logo.png b/logo.png\nBinary files a/logo.png and b/logo.png differ\n" +
			"diff --git \"a/sp ace\\tq.go\" \"b/sp ace\\tq.go\"\n--- a/gone\n+++ /dev/null\n@@ -1 +0,0 @@\n-bye\n",
		"\x00d\x00notatime\x00D\x00\n",
		"\x00e\x001\x00E\x00\ndiff --git a/x b/x\n@@ -1,999 +1 @@\n-truncated\n",
		"\x00f\x001\x00F\x00\ndiff --git a/x b/x\n@@ -1,-1 +a @@\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		failed := false
		for c, err := range Parse(strings.NewReader(input)) {
			if failed {
				t.Fatalf("yielded %+v after an error", c)
			}
			if err != nil {
				failed = true
				continue
			}
			if strings.Contains(c.Rev, "\x00") || strings.Contains(c.Author, "\x00") {
				t.Fatalf("header field contains NUL: %+v", c)
			}
			for _, file := range c.Files {
				for _, h := range file.Hunks {
					for _, line := range append(h.Deleted, h.Added...) {
						if strings.Contains(line, "\n") {
							t.Fatalf("hunk line contains a newline: %q", line)
						}
					}
				}
			}
		}
	})
}
