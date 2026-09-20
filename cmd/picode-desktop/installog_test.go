package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallLogPath(t *testing.T) {
	t.Setenv("ProgramData", `C:\ProgramData`)
	want := filepath.Join(`C:\ProgramData`, "PiCode Desktop", "install.log")
	if got := InstallLogPath(); got != want {
		t.Errorf("InstallLogPath = %q, want %q", got, want)
	}
	t.Setenv("ProgramData", "")
	want = filepath.Join(os.TempDir(), "picode-install.log")
	if got := InstallLogPath(); got != want {
		t.Errorf("InstallLogPath without ProgramData = %q, want %q", got, want)
	}
}

func TestChildExitError(t *testing.T) {
	if err := childExitError(0); err != nil {
		t.Errorf("childExitError(0) = %v, want nil", err)
	}
	err := childExitError(3)
	if err == nil || !strings.Contains(err.Error(), "exit 3") {
		t.Fatalf("err = %v, want the exit code named", err)
	}
	if !strings.Contains(err.Error(), InstallLogPath()) {
		t.Errorf("err = %v, want the log path named", err)
	}
}

func TestLogInstallError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "install.log")
	if err := logInstallError(path, errors.New("boom")); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "boom") {
		t.Errorf("log lacks the error: %q", body)
	}
	if err := logInstallError(path, nil); err != nil {
		t.Errorf("logInstallError(nil) = %v, want nil", err)
	}
}

func TestPauseForEnter(t *testing.T) {
	pauseForEnter(io.Discard, strings.NewReader("x\n"))
	pauseForEnter(io.Discard, strings.NewReader(""))
}
