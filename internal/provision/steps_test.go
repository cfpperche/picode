package provision

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/cfpperche/picode/internal/tlsutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	want := []string{"wsl-conf", "systemd", "linger", "cert", "service", "health", "clis", "tailnet", "tailnet-cert", "reach"}
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

// The CLI step is informational (ADR-0179): it names what is on PATH and
// never blocks — a machine with no agent CLI still converges.
func TestCLIsStep(t *testing.T) {
	old := lookPath
	lookPath = func(name string) (string, error) {
		if name == "claude" || name == "omp" {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("no")
	}
	t.Cleanup(func() { lookPath = old })
	got := clisStep().Check(Env{})
	if got.Status != StatusOK || !strings.Contains(got.Detail, "Claude Code") || !strings.Contains(got.Detail, "Omp") || strings.Contains(got.Detail, "Pi,") {
		t.Fatalf("%+v", got)
	}
	lookPath = func(string) (string, error) { return "", errors.New("no") }
	if got := clisStep().Check(Env{}); got.Status != StatusOK || !strings.Contains(got.Detail, "none yet") {
		t.Fatalf("%+v", got)
	}
	if clisStep().Fix(Env{}) != nil {
		t.Fatal("the CLI step has nothing to fix")
	}
}

func TestTailnetStep(t *testing.T) {
	oldL, oldO := lookPath, output
	t.Cleanup(func() { lookPath, output = oldL, oldO })

	lookPath = func(string) (string, error) { return "", errors.New("no") }
	if got := tailnetStep().Check(Env{}); got.Status != StatusOK || !strings.Contains(got.Detail, "LAN only") {
		t.Fatalf("absent: %+v", got)
	}
	lookPath = func(string) (string, error) { return "/usr/bin/tailscale", nil }
	output = func(string, ...string) (string, error) {
		return `{"BackendState":"Running","Self":{"DNSName":"box.tail1234.ts.net.","TailscaleIPs":["100.64.0.9"]}}`, nil
	}
	if got := tailnetStep().Check(Env{}); got.Status != StatusOK || !strings.Contains(got.Detail, "box.tail1234.ts.net") {
		t.Fatalf("running: %+v", got)
	}
	output = func(string, ...string) (string, error) { return `{"BackendState":"Stopped"}`, nil }
	if got := tailnetStep().Check(Env{}); got.Status != StatusBlocked {
		t.Fatalf("stopped: %+v", got)
	}
}

func TestReachStep(t *testing.T) {
	oldL, oldO := lookPath, output
	t.Cleanup(func() { lookPath, output = oldL, oldO })
	lookPath = func(string) (string, error) { return "", errors.New("no") }
	dir := t.TempDir()
	env := Env{DataDir: dir}
	if got := reachStep().Check(env); got.Status != StatusBlocked {
		t.Fatalf("no server.json: %+v", got)
	}
	write := func(body string) { _ = os.WriteFile(filepath.Join(dir, "server.json"), []byte(body), 0o644) }
	write(`{"url":"https://localhost:8445","bind":"127.0.0.1","port":8445,"publicUrl":""}`)
	if got := reachStep().Check(env); got.Status != StatusBlocked || !strings.Contains(got.Detail, "loopback") {
		t.Fatalf("loopback: %+v", got)
	}
	write(`{"url":"https://localhost:8445","bind":"0.0.0.0","port":8445,"publicUrl":"https://box.tail.ts.net:8445"}`)
	if got := reachStep().Check(env); got.Status != StatusOK || !strings.Contains(got.Detail, "box.tail.ts.net") {
		t.Fatalf("public: %+v", got)
	}
	write(`{"url":"https://localhost:8445","bind":"0.0.0.0","port":8445,"publicUrl":""}`)
	lookPath = func(string) (string, error) { return "/usr/bin/tailscale", nil }
	output = func(string, ...string) (string, error) { return "100.64.0.9\n", nil }
	if got := reachStep().Check(env); got.Status != StatusOK || !strings.Contains(got.Detail, "100.64.0.9:8445") {
		t.Fatalf("tailnet ip: %+v", got)
	}
}

func TestTailnetCertStep(t *testing.T) {
	oldL, oldO := lookPath, output
	t.Cleanup(func() { lookPath, output = oldL, oldO })
	dir := t.TempDir()
	env := Env{DataDir: dir}

	lookPath = func(string) (string, error) { return "", errors.New("no") }
	if got := tailnetCertStep().Check(env); got.Status != StatusOK {
		t.Fatalf("absent: %+v", got)
	}
	lookPath = func(string) (string, error) { return "/usr/bin/tailscale", nil }
	output = func(string, ...string) (string, error) { return `{"BackendState":"Stopped"}`, nil }
	if got := tailnetCertStep().Check(env); got.Status != StatusOK {
		t.Fatalf("stopped: %+v", got)
	}
	output = func(string, ...string) (string, error) {
		return `{"BackendState":"Running","Self":{"DNSName":"box.tail1234.ts.net."}}`, nil
	}
	if got := tailnetCertStep().Check(env); got.Status != StatusFix || !strings.Contains(got.Detail, "no certificate") {
		t.Fatalf("missing: %+v", got)
	}
	// A leaf on disk for other names (the local one, renamed) is "another name".
	if _, err := tlsutil.Ensure(dir); err != nil {
		t.Fatal(err)
	}
	_ = os.Rename(filepath.Join(dir, tlsutil.CertFile), filepath.Join(dir, tlsutil.TailscaleCertFile))
	_ = os.Rename(filepath.Join(dir, tlsutil.KeyFile), filepath.Join(dir, tlsutil.TailscaleKeyFile))
	if got := tailnetCertStep().Check(env); got.Status != StatusFix || !strings.Contains(got.Detail, "another name") {
		t.Fatalf("other name: %+v", got)
	}
}

// writeLeaf writes a self-signed certificate with the given names, valid
// for a year, to <dir>/cert.pem.
func writeLeaf(t *testing.T, dir string, dns []string, ips []net.IP) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{SerialNumber: big.NewInt(1), DNSNames: dns, IPAddresses: ips,
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(365 * 24 * time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	out := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(filepath.Join(dir, "cert.pem"), out, 0o644); err != nil {
		t.Fatal(err)
	}
}

// 2026-09-24: the owner's mkcert certificate (from setup-cert.sh) covered
// localhost, picode.local, a Tailscale name and two LAN addresses, but not
// 127.0.0.1 or ::1. A reissue adds them and keeps every DNS name it had;
// addresses come from the interfaces as they are now, so an old one (a
// Docker bridge, a LAN the machine left) is not carried forward.
func TestCertStepReissuesACertificateWithoutLoopback(t *testing.T) {
	oldLook, oldRun := lookPath, run
	t.Cleanup(func() { lookPath, run = oldLook, oldRun })
	lookPath = func(string) (string, error) { return "/usr/bin/mkcert", nil }
	var got []string
	run = func(name string, args ...string) error { got = append([]string{name}, args...); return nil }

	dir := filepath.Join(t.TempDir(), "data")
	writeLeaf(t, dir, []string{"localhost", "picode.local", "Box-1.Tail057039.ts.net"}, []net.IP{net.ParseIP("192.168.15.28"), net.ParseIP("203.0.113.77")})
	env := Env{DataDir: dir}
	st := certStep().Check(env)
	if st.Status != StatusFix || !strings.Contains(st.Detail, "127.0.0.1") || !strings.Contains(st.Detail, "::1") {
		t.Fatalf("check = %q (%s), want fix naming 127.0.0.1 and ::1", st.Status, st.Detail)
	}
	if err := certStep().Fix(env); err != nil {
		t.Fatal(err)
	}
	joined := " " + strings.Join(got, " ") + " "
	for _, want := range []string{"mkcert", " box-1.tail057039.ts.net ", " 127.0.0.1 ", " ::1 ", " localhost ", " picode.local "} {
		if !strings.Contains(joined, want) {
			t.Errorf("mkcert args %q lack %q", joined, strings.TrimSpace(want))
		}
	}
	// 203.0.113.77 (documentation range) is on no interface of any machine.
	if strings.Contains(joined, " 203.0.113.77 ") {
		t.Errorf("an old address was carried into the reissue: %q", joined)
	}
}

func TestCertStepLeavesLoopbackAloneWithoutMkcert(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })
	lookPath = func(string) (string, error) { return "", errors.New("no mkcert") }
	dir := filepath.Join(t.TempDir(), "data")
	writeLeaf(t, dir, []string{"localhost"}, nil)
	if st := certStep().Check(Env{DataDir: dir}); st.Status != StatusOK {
		t.Fatalf("check = %q (%s): without mkcert nothing can reissue it trusted", st.Status, st.Detail)
	}
}

func TestCertStepAcceptsACertificateThatCoversLoopback(t *testing.T) {
	old := lookPath
	t.Cleanup(func() { lookPath = old })
	lookPath = func(string) (string, error) { return "/usr/bin/mkcert", nil }
	dir := filepath.Join(t.TempDir(), "data")
	writeLeaf(t, dir, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")})
	if st := certStep().Check(Env{DataDir: dir}); st.Status != StatusOK {
		t.Fatalf("check = %q (%s), want ok", st.Status, st.Detail)
	}
}
