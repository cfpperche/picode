package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeExe(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readExe(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// The swap is pure file moves, so the whole dance runs on any OS — including
// the rollback half, which is the part that must never be wrong.
func TestSwapExeReplacesKeepingOriginalAside(t *testing.T) {
	dir := t.TempDir()
	exe, next := filepath.Join(dir, "picode-shell.exe"), filepath.Join(dir, "picode-shell.exe.new")
	writeExe(t, exe, "v1")
	writeExe(t, next, "v2")
	if err := swapExe(exe, next); err != nil {
		t.Fatal(err)
	}
	if got := readExe(t, exe); got != "v2" {
		t.Fatalf("exe = %q", got)
	}
	if got := readExe(t, exe+".old"); got != "v1" {
		t.Fatalf("aside = %q", got)
	}
	if _, err := os.Stat(next); !os.IsNotExist(err) {
		t.Fatal(".new survived the swap")
	}
}

func TestSwapExeMissingNextLeavesOriginalAlone(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "picode-shell.exe")
	writeExe(t, exe, "v1")
	if err := swapExe(exe, filepath.Join(dir, "missing.new")); err == nil {
		t.Fatal("swap of a missing download succeeded")
	}
	if got := readExe(t, exe); got != "v1" {
		t.Fatalf("exe = %q", got)
	}
	if _, err := os.Stat(exe + ".old"); !os.IsNotExist(err) {
		t.Fatal(".old survived the rollback")
	}
}

func TestUnswapExeRestoresTheOriginal(t *testing.T) {
	dir := t.TempDir()
	exe, next := filepath.Join(dir, "tool.exe"), filepath.Join(dir, "tool.exe.new")
	writeExe(t, exe, "v1")
	writeExe(t, next, "v2")
	if err := swapExe(exe, next); err != nil {
		t.Fatal(err)
	}
	unswapExe(exe)
	if got := readExe(t, exe); got != "v1" {
		t.Fatalf("exe = %q", got)
	}
}
