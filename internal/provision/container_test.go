package provision

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContainerStepsShape(t *testing.T) {
	want := []string{"container-tools", "account", "linger", "binary", "rootfs", "member-env", "container-unit", "member-health"}
	steps := ContainerSteps()
	if len(steps) != len(want) {
		t.Fatalf("%d steps", len(steps))
	}
	for i, id := range want {
		if steps[i].ID != id {
			t.Errorf("step %d = %q", i, steps[i].ID)
		}
	}
}

func TestContainerToolsStep(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })
	lookPath = func(string) (string, error) { return "", errors.New("no") }
	if got := containerToolsStep().Check(Env{}); got.Status != StatusBlocked || !strings.Contains(got.Detail, "apt install systemd-container") {
		t.Fatalf("%+v", got)
	}
	lookPath = func(string) (string, error) { return "/usr/bin/x", nil }
	if got := containerToolsStep().Check(Env{}); got.Status != StatusOK {
		t.Fatalf("%+v", got)
	}
}

func TestRootfsStepReadsTheMachineDir(t *testing.T) {
	oldSuite := hostSuite
	t.Cleanup(func() { hostSuite = oldSuite })
	hostSuite = func() string { return "noble" }
	if got := rootfsStep().Check(Env{User: "nobody-such-user"}); got.Status != StatusFix || !strings.Contains(got.Detail, "debootstrap noble") {
		t.Fatalf("%+v", got)
	}
}

func TestContainerUnitStepWantsTheExactUnit(t *testing.T) {
	old := gatewayHostname
	t.Cleanup(func() { gatewayHostname = old })
	gatewayHostname = func() (string, error) { return "", errors.New("no") }
	if got := containerUnitStep().Check(Env{User: "alice", Home: "/home/alice"}); got.Status != StatusBlocked {
		t.Fatalf("%+v", got)
	}
	gatewayHostname = func() (string, error) { return "box.tail1234.ts.net", nil }
	if got := containerUnitStep().Check(Env{User: "alice", Home: "/home/alice"}); got.Status != StatusFix {
		t.Fatalf("%+v", got)
	}
	_ = os.MkdirAll(filepath.Join(t.TempDir(), "x"), 0o755)
}

// A member root built before ADR-0179's Node fix has tmux but no member
// npmrc: the step asks to converge it (NodeSource Node, per-member prefix),
// and a converged root is ok.
func TestRootfsStepWantsTheMemberNpmPrefix(t *testing.T) {
	old := MachinesDir
	MachinesDir = t.TempDir()
	t.Cleanup(func() { MachinesDir = old })
	root := RootfsFor("alice")
	if err := os.MkdirAll(filepath.Join(root, "usr", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "usr", "bin", "tmux"), nil, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := rootfsStep().Check(Env{User: "alice"}); got.Status != StatusFix || !strings.Contains(got.Detail, "npm prefix") {
		t.Fatalf("old root = %+v", got)
	}
	if err := os.MkdirAll(filepath.Join(root, "usr", "etc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, memberNpmrc), []byte("prefix=${HOME}/.local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := rootfsStep().Check(Env{User: "alice"}); got.Status != StatusOK {
		t.Fatalf("converged root = %+v", got)
	}
}
