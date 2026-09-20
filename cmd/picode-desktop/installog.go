package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// The elevated child runs in its own console, so a failure there is invisible
// to whatever launched the install — the run-1 dogfood died this way, and the
// parent had already exited 0. Three small answers: the parent waits for the
// child and reports its exit code (elevate_windows.go), the child leaves its
// last line in InstallLogPath, and it pauses for Enter so a human can read
// the window. Nothing here is Windows-only, so it all tests from Linux.

// InstallLogPath is where a failed install leaves its last line. ProgramData
// is the same directory elevated or not, which %TEMP% is not.
func InstallLogPath() string {
	if pd := os.Getenv("ProgramData"); pd != "" {
		return filepath.Join(pd, "PiCode Desktop", "install.log")
	}
	return filepath.Join(os.TempDir(), "picode-install.log")
}

// childExitError turns the elevated child's exit code into the parent's
// report. Zero is the child having taken over cleanly.
func childExitError(code uint32) error {
	if code == 0 {
		return nil
	}
	return fmt.Errorf("setup did not finish (exit %d) — last error in %s", code, InstallLogPath())
}

// logInstallError appends one timestamped line. A nil error writes nothing.
func logInstallError(path string, logErr error) error {
	if logErr == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s install: %v\n", time.Now().Format(time.RFC3339), logErr)
	return err
}

// pauseForEnter holds a failed install's window open for a human. A closed
// stdin refuses at once instead of hanging a pipe.
func pauseForEnter(w io.Writer, r io.Reader) {
	fmt.Fprint(w, "Press Enter to close this window...")
	_, _ = bufio.NewReader(r).ReadString('\n')
}
