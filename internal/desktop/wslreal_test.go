package desktop

import (
	"os"
	"os/exec"
	"testing"
)

// realWSL skips unless an actual wsl.exe is reachable. Like the tmux-gated
// tests, this runs on a developer's machine and skips in CI — the parsing
// itself is covered by the captured fixture in wsl_test.go, so nothing here is
// load-bearing for correctness.
func realWSL(t *testing.T) Runner {
	t.Helper()
	exe := ResolveWSLExe()
	if _, err := os.Stat(exe); err != nil {
		if _, err := exec.LookPath(exe); err != nil {
			t.Skip("no wsl.exe on this machine")
		}
	}
	WSLExe = exe
	return liveRunner{}
}

type liveRunner struct{}

func (liveRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (liveRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func TestListDistrosAgainstRealWSL(t *testing.T) {
	r := realWSL(t)

	distros, err := ListDistros(r)
	if err != nil {
		t.Fatalf("ListDistros: %v", err)
	}
	if len(distros) == 0 {
		t.Fatal("no distros parsed from a live wsl.exe")
	}
	for _, d := range distros {
		if d.Name == "" || d.State == "" || d.Version == 0 {
			t.Errorf("incomplete distro parsed: %+v", d)
		}
	}

	picked, err := Pick(distros, "")
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	t.Logf("picked %q (state %s, WSL %d)", picked.Name, picked.State, picked.Version)
}

func TestDefaultUserAgainstRealWSL(t *testing.T) {
	r := realWSL(t)

	distros, err := ListDistros(r)
	if err != nil {
		t.Skipf("ListDistros: %v", err)
	}
	picked, err := Pick(distros, "")
	if err != nil {
		t.Skipf("Pick: %v", err)
	}

	user, err := DefaultUser(r, picked.Name)
	if err != nil {
		t.Fatalf("DefaultUser: %v", err)
	}
	if user == "" || user == "root" {
		t.Errorf("DefaultUser = %q", user)
	}
	t.Logf("%s logs in as %q", picked.Name, user)
}
