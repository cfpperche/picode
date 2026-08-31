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
