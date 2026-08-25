package binwatch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChanged(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bin")
	if err := os.WriteFile(p, []byte("a"), 0o755); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	start := Stamp{Path: p, Mtime: st.ModTime(), Size: st.Size()}
	if Changed(start) {
		t.Fatal("fresh file must not look changed")
	}
	if err := os.WriteFile(p, []byte("ab"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !Changed(start) {
		t.Fatal("rewritten file must look changed")
	}
}
