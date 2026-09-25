package clisession

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// piForkWait bounds the one headless run that makes the copy (measured
// 2026-09-24 on pi 0.87.1: 0.31 s with extensions off).
const piForkWait = 30 * time.Second

// ForkIntoDir copies a Pi conversation with pi's own `--fork`: pi writes
// the copy at once under the exact `--session-id`, in `--session-dir`,
// with the source's entries unchanged and `parentSession` naming the
// source; only the header's cwd follows the folder it ran in, which is why
// start runs in the fork's folder. Extensions, skills, prompt templates and
// themes stay off — the run only writes the file — and stdin is closed, so
// the RPC loop ends as soon as the copy exists.
func (PISource) ForkIntoDir(ctx context.Context, src Ref, dir, newID string, start func(ctx context.Context, args ...string) *exec.Cmd) (string, error) {
	if strings.TrimSpace(src.Path) == "" {
		return "", errors.New("This Pi agent has no conversation file to fork yet.")
	}
	if _, err := os.Stat(src.Path); err != nil {
		return "", fmt.Errorf("The conversation file is gone: %s", src.Path)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, piForkWait)
	defer cancel()
	cmd := start(ctx, "--mode", "rpc", "--no-extensions", "--no-skills", "--no-prompt-templates", "--no-themes",
		"--fork", src.Path, "--session-id", newID, "--session-dir", dir)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil
	cmd.Stdin = nil
	runErr := cmd.Run()
	matches, _ := filepath.Glob(filepath.Join(dir, "*_"+newID+".jsonl"))
	if len(matches) == 1 {
		return matches[0], nil
	}
	msg := strings.TrimSpace(stderr.String())
	if len(msg) > 300 {
		msg = msg[:300]
	}
	switch {
	case runErr != nil && msg != "":
		return "", fmt.Errorf("Pi could not fork this conversation: %s", msg)
	case runErr != nil:
		return "", fmt.Errorf("Pi could not fork this conversation: %v", runErr)
	}
	return "", errors.New("Pi ran but wrote no copy of the conversation.")
}
