package osopen

import "testing"

func TestWSLToWin(t *testing.T) {
	got, ok := WSLToWin("/mnt/e/picode-backups/x")
	if !ok || got != `E:\picode-backups\x` {
		t.Fatalf("got %q %v", got, ok)
	}
	if _, ok := WSLToWin("/home/goat"); ok {
		t.Fatal("linux path")
	}
}

func TestWindowsExplorerPath(t *testing.T) {
	if !RunningWSL() {
		t.Skip("not WSL")
	}
	if windowsExplorer() == "" {
		t.Fatal("explorer.exe not found under /mnt/c/Windows")
	}
}

func TestWindowsPathUnderWSL(t *testing.T) {
	t.Setenv("WSL_DISTRO_NAME", "Test")
	cases := []struct{ in, want string }{
		{"/mnt/c/Users/x/repo", `C:\Users\x\repo`},
		{"/home/x/repo", `\\wsl.localhost\Test\home\x\repo`},
	}
	for _, c := range cases {
		if got, ok := WindowsPath(c.in); !ok || got != c.want {
			t.Errorf("WindowsPath(%q) = %q %v; want %q", c.in, got, ok, c.want)
		}
	}
	if _, ok := WindowsPath(""); ok {
		t.Error("an empty path has no Windows form")
	}
}
