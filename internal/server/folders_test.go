package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	got, err := expandPath("")
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Fatalf("empty = %s want %s", got, home)
	}
	got, err = expandPath("~/picode")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(home, "picode") {
		t.Fatalf("~ = %s", got)
	}
}

func TestExpandPathAbs(t *testing.T) {
	dir := t.TempDir()
	got, err := expandPath(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("got %s", got)
	}
}
