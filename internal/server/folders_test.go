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

func TestWinPathToWSL(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\Users\cfpp`, "/mnt/c/Users/cfpp"},
		{"C:/Users/cfpp", "/mnt/c/Users/cfpp"},
		{"d:", "/mnt/d"},
		{"E:\\Pics", "/mnt/e/Pics"},
	}
	for _, c := range cases {
		got, ok := winPathToWSL(c.in)
		if !ok || got != c.want {
			t.Fatalf("%q → %q ok=%v want %q", c.in, got, ok, c.want)
		}
	}
	if _, ok := winPathToWSL("/home/goat"); ok {
		t.Fatal("posix path converted")
	}
}

func TestWinLabel(t *testing.T) {
	if got := winLabel("/mnt/c"); got != `C:\` {
		t.Fatalf("c = %q", got)
	}
	if got := winLabel("/mnt/c/Users/cfpp"); got != `C:\Users\cfpp` {
		t.Fatalf("users = %q", got)
	}
	if winLabel("/home/goat") != "" {
		t.Fatal("linux path labeled")
	}
}

func TestWindowsMounts(t *testing.T) {
	if !isWSL() {
		t.Skip("not WSL")
	}
	m := windowsMounts()
	if len(m) == 0 {
		t.Fatal("expected at least one Windows drive under /mnt")
	}
	for _, e := range m {
		if len(e.Name) != 2 || e.Name[1] != ':' {
			t.Fatalf("bad drive name %q", e.Name)
		}
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
