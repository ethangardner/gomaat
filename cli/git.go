package cli

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// streamGit runs `git [-C dir] args...` and hands its stdout to consume as it
// is produced, so large outputs (e.g. full patch history) are never buffered
// whole. Errors name the git subcommand (args[0]). If consume fails, the git
// process is reaped and consume's error is returned; if git itself fails, the
// error carries its stderr and the full command line.
func streamGit(dir string, args []string, consume func(io.Reader) error) error {
	if len(args) == 0 {
		return errors.New("streamGit: no git arguments provided")
	}
	sub := args[0]
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}

	var stderr strings.Builder
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("creating git stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting git %s: %w", sub, err)
	}

	if err := consume(stdout); err != nil {
		_ = stdout.Close()
		_ = cmd.Wait()
		return fmt.Errorf("processing git %s output: %w", sub, err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git %s failed: %w\n%s\nCommand: git %s", sub, err, strings.TrimSpace(stderr.String()), strings.Join(args, " "))
	}
	return nil
}
