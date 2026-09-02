package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitFile(t *testing.T) {
	got := UnitFile("/home/x/.local/bin/picode", "/home/x/.nvm/bin:/usr/bin", "/home/x")
	for _, want := range []string{
		"ExecStart=/home/x/.local/bin/picode",
		"Restart=on-failure",
		"WantedBy=default.target",
		"Environment=HOME=/home/x",
		"Environment=PATH=/home/x/.nvm/bin:/usr/bin",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestWithLocalBin(t *testing.T) {
	if got := withLocalBin("/usr/bin", "/home/x/.local/bin"); got != "/home/x/.local/bin:/usr/bin" {
		t.Fatalf("got %q", got)
	}
	if got := withLocalBin("/home/x/.local/bin:/usr/bin", "/home/x/.local/bin"); got != "/home/x/.local/bin:/usr/bin" {
		t.Fatalf("dup %q", got)
	}
}

func TestCopyExeAndWriteUnit(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "srcbin")
	if err := os.WriteFile(src, []byte("fake"), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	p := ForHome(home)
	if err := CopyExe(src, p.Bin); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p.Bin)
	if err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("bin %v %v", st, err)
	}
	if err := writeUnit(p, "/usr/bin"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(p.Unit)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "ExecStart="+p.Bin) {
		t.Fatalf("unit:\n%s", body)
	}
}

func TestDeployRequiresUnit(t *testing.T) {
	if !systemdAvailable() {
		t.Skip("systemd not running")
	}
	err := Deploy("/bin/true", t.TempDir(), "/usr/bin")
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
}

func TestLockPID(t *testing.T) {
	p := filepath.Join(t.TempDir(), "picode.lock")
	if lockPID(p) != 0 {
		t.Fatal("missing")
	}
	if err := os.WriteFile(p, []byte("12345\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if lockPID(p) != 12345 {
		t.Fatalf("got %d", lockPID(p))
	}
}

func TestGatewayUnitFile(t *testing.T) {
	u := GatewayUnitFile("/usr/local/bin/picode", "/etc/picode/gateway.json")
	for _, want := range []string{"ExecStart=/usr/local/bin/picode gateway --config /etc/picode/gateway.json", "After=network-online.target tailscaled.service", "WantedBy=multi-user.target", "Restart=on-failure"} {
		if !strings.Contains(u, want) {
			t.Errorf("missing %q in:\n%s", want, u)
		}
	}
}
