package snippet

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunnable(t *testing.T) {
	if !Runnable("Python") || !Runnable("go") || Runnable("yaml") {
		t.Fatal("lang set")
	}
}

func TestRunPython(t *testing.T) {
	if lookBin([]string{"python3", "python"}) == "" {
		t.Skip("no python")
	}
	dir := t.TempDir()
	res, err := Run(dir, "python", "print(1+1)")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || strings.TrimSpace(res.Stdout) != "2" {
		t.Fatalf("%+v", res)
	}
}

func TestRunUnknown(t *testing.T) {
	_, err := Run(t.TempDir(), "dockerfile", "FROM scratch")
	if err == nil {
		t.Fatal("want error")
	}
}

func TestPrepGo(t *testing.T) {
	if !strings.HasPrefix(prepGo("func main() {}"), "package main") {
		t.Fatal("wrap")
	}
	if runtime.GOOS == "windows" {
		return
	}
	if lookBin([]string{"go"}) == "" {
		t.Skip("no go")
	}
	dir := t.TempDir()
	res, err := Run(dir, "go", "package main\nimport \"fmt\"\nfunc main() { fmt.Print(\"ok\") }")
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Stdout != "ok" {
		t.Fatalf("%+v", res)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "picode-run-*"))
	if len(matches) != 0 {
		t.Fatalf("left files %v", matches)
	}
	_ = os.Remove
}
