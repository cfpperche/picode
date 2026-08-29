package provision

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withConf points the wsl.conf step at a scratch file seeded with content.
func withConf(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wsl.conf")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := ConfPath
	ConfPath = path
	t.Cleanup(func() { ConfPath = old })
	return path
}

func TestWSLConfStepSkipsPlainLinux(t *testing.T) {
	withConf(t, "")
	got := wslConfStep().Check(Env{InWSL: false})
	if got.Status != StatusOK {
		t.Errorf("status = %q (%s), want ok on non-WSL", got.Status, got.Detail)
	}
}

func TestWSLConfStepAddsSystemdAndBacksUp(t *testing.T) {
	const before = "[user]\ndefault=goat\n"
	path := withConf(t, before)
	step := wslConfStep()
	env := Env{InWSL: true}

	if got := step.Check(env); got.Status != StatusFix {
		t.Fatalf("status = %q (%s), want fix", got.Status, got.Detail)
	}
	if err := step.Fix(env); err != nil {
		t.Fatal(err)
	}
	if got := step.Check(env); got.Status != StatusOK {
		t.Errorf("after fix: status = %q (%s), want ok", got.Status, got.Detail)
	}

	want := "[user]\ndefault=goat\n\n[boot]\nsystemd=true\n"
	if got, _ := os.ReadFile(path); string(got) != want {
		t.Errorf("merged file = %q, want %q", got, want)
	}
	bak, err := os.ReadFile(path + BackupSuffix)
	if err != nil || string(bak) != before {
		t.Errorf("backup = %q (%v), want the original %q", bak, err, before)
	}
}

// The whole feature rests on this: a machine that is already configured must
// come out of provisioning with the same bytes it went in with, and with no
// stray backup suggesting something happened.
func TestWSLConfStepWritesNothingWhenAlreadySatisfied(t *testing.T) {
	path := withConf(t, ownerConf)
	step := wslConfStep()
	env := Env{InWSL: true}

	if got := step.Check(env); got.Status != StatusOK {
		t.Fatalf("status = %q (%s), want ok", got.Status, got.Detail)
	}
	if err := step.Fix(env); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != ownerConf {
		t.Errorf("the file was rewritten\n--- got ---\n%s\n--- want ---\n%s", got, ownerConf)
	}
	if _, err := os.Stat(path + BackupSuffix); !os.IsNotExist(err) {
		t.Error("a backup was written for a file that never changed")
	}
}

// A wsl.conf that cannot be read is blocked, not fixable: writing over a file
// we failed to read is how settings disappear.
func TestWSLConfStepBlocksOnAnUnreadableFile(t *testing.T) {
	old := ConfPath
	ConfPath = t.TempDir() // a directory reads as an error, not as absence
	t.Cleanup(func() { ConfPath = old })

	if got := wslConfStep().Check(Env{InWSL: true}); got.Status != StatusBlocked {
		t.Errorf("status = %q (%s), want blocked", got.Status, got.Detail)
	}
}

// `systemctl --user` answers for the calling account, always. Asked from root
// about somebody else's unit it reported "present but not enabled" — a
// confident, wrong sentence about a service that was enabled and running. A
// check that cannot see the truth has to say so.
func TestServiceStepRefusesToAnswerForAnotherAccount(t *testing.T) {
	home := t.TempDir()
	unit := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unit, "picode.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var asked bool
	old := output
	output = func(name string, args ...string) (string, error) {
		asked = true
		return "", nil
	}
	t.Cleanup(func() { output = old })

	got := serviceStep().Check(Env{Home: home, User: "goat", Acting: "root"})
	if got.Status != StatusBlocked {
		t.Errorf("status = %q (%s), want blocked", got.Status, got.Detail)
	}
	if asked {
		t.Error("systemctl was asked anyway — its answer would be about root, not goat")
	}
	if !strings.Contains(got.Detail, "goat") {
		t.Errorf("detail = %q, want it to name the account it cannot read", got.Detail)
	}
}

func TestLingerStep(t *testing.T) {
	dir := t.TempDir()
	old := lingerDir
	lingerDir = dir
	t.Cleanup(func() { lingerDir = old })

	step := lingerStep()
	env := Env{User: "goat"}
	if got := step.Check(env); got.Status != StatusFix {
		t.Errorf("status = %q (%s), want fix when linger is off", got.Status, got.Detail)
	}
	if err := os.WriteFile(filepath.Join(dir, "goat"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := step.Check(env); got.Status != StatusOK {
		t.Errorf("status = %q (%s), want ok when linger is on", got.Status, got.Detail)
	}
}

func TestLingerFixCallsLoginctl(t *testing.T) {
	var gotName string
	var gotArgs []string
	old := run
	run = func(name string, args ...string) error {
		gotName, gotArgs = name, args
		return nil
	}
	t.Cleanup(func() { run = old })

	if err := lingerStep().Fix(Env{User: "goat"}); err != nil {
		t.Fatal(err)
	}
	if gotName != "loginctl" || len(gotArgs) != 2 || gotArgs[0] != "enable-linger" || gotArgs[1] != "goat" {
		t.Errorf("ran %q %v, want loginctl enable-linger goat", gotName, gotArgs)
	}
}

// Without mkcert the step still has to leave a usable certificate behind, so
// HTTPS works on a machine that has never run setup-cert.sh.
func TestCertStepFallsBackToSelfSigned(t *testing.T) {
	old := lookPath
	lookPath = func(string) (string, error) { return "", errors.New("no mkcert") }
	t.Cleanup(func() { lookPath = old })

	env := Env{DataDir: filepath.Join(t.TempDir(), "data")}
	step := certStep()

	if got := step.Check(env); got.Status != StatusFix {
		t.Fatalf("status = %q (%s), want fix on an empty data dir", got.Status, got.Detail)
	}
	if err := step.Fix(env); err != nil {
		t.Fatal(err)
	}
	if got := step.Check(env); got.Status != StatusOK {
		t.Errorf("after fix: status = %q (%s), want ok", got.Status, got.Detail)
	}
}

func TestCertExpiryRejectsNonPEM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cert.pem")
	if err := os.WriteFile(path, []byte("not a certificate"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := certExpiry(path); err == nil {
		t.Error("a non-PEM file was accepted as a certificate")
	}
}

func TestServerURL(t *testing.T) {
	dir := t.TempDir()
	if _, err := serverURL(dir); err == nil {
		t.Error("a missing server.json was accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "server.json"), []byte(`{"url":"https://localhost:8445","port":8445}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := serverURL(dir)
	if err != nil || got != "https://localhost:8445" {
		t.Errorf("serverURL = %q (%v), want https://localhost:8445", got, err)
	}
}

// Steps must stay in dependency order: fixing the service before systemd runs,
// or checking health before the service exists, reports nonsense.
func TestStepsAreInDependencyOrder(t *testing.T) {
	want := []string{"wsl-conf", "systemd", "linger", "cert", "service", "health"}
	steps := Steps()
	if len(steps) != len(want) {
		t.Fatalf("got %d steps, want %d", len(steps), len(want))
	}
	for i, id := range want {
		if steps[i].ID != id {
			t.Errorf("step %d = %q, want %q", i, steps[i].ID, id)
		}
	}
}
