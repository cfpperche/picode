package desktop

import (
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/version"
)

const fullProbe = `@@picode@@
picode 0.3.1
@@tools@@
missing:tmux
missing:pi
v22.14.0
@@family@@
ID=ubuntu
@@marker@@
adopted
`

func TestParseProbe(t *testing.T) {
	r := ParseProbe([]byte(fullProbe))
	if r.Picode != "0.3.1" {
		t.Errorf("Picode = %q", r.Picode)
	}
	if strings.Join(r.Missing, ",") != "tmux,pi" {
		t.Errorf("Missing = %q", r.Missing)
	}
	if r.NodeMajor != "22" {
		t.Errorf("NodeMajor = %q", r.NodeMajor)
	}
	if r.Family != "ubuntu" {
		t.Errorf("Family = %q", r.Family)
	}
	if r.Registered {
		t.Error("adopted distro reported as registered")
	}
}

func TestParseProbeDegrades(t *testing.T) {
	rows := []struct {
		name  string
		out   string
		check func(ProbeResult) bool
	}{
		{"missing picode", "@@picode@@\nMISSING\n", func(r ProbeResult) bool { return r.Picode == "" }},
		{"source revision stripped", "@@picode@@\npicode 0.3.1+abc1234\n", func(r ProbeResult) bool { return r.Picode == "0.3.1" }},
		{"broken version line", "@@picode@@\nversion unknown\n", func(r ProbeResult) bool { return r.Picode == "" }},
		{"quoted family", "@@family@@\nID=\"Debian\"\n", func(r ProbeResult) bool { return r.Family == "debian" }},
		{"absent family", "", func(r ProbeResult) bool { return r.Family == "unknown" }},
		{"registered marker", "@@marker@@\nregistered\n", func(r ProbeResult) bool { return r.Registered }},
		{"node missing", "@@tools@@\nmissing:node\nnodeversion:\n", func(r ProbeResult) bool { return r.NodeMajor == "" }},
		{"garbage", "not a probe\nat all\n", func(r ProbeResult) bool { return r.Picode == "" && r.Family == "unknown" }},
	}
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(ParseProbe([]byte(tt.out))) {
				t.Errorf("wrong parse of %q", tt.out)
			}
		})
	}
}

func TestPicodeWanted(t *testing.T) {
	rows := []struct {
		installed, want string
		expected        bool
	}{
		{"", "", true},
		{"", "0.3.1", true},
		{"0.3.1", "", false},
		{"0.3.1", "0.3.1", false},
		{"0.2.0", "0.3.1", true},
	}
	for _, tt := range rows {
		if got := PicodeWanted(tt.installed, tt.want); got != tt.expected {
			t.Errorf("PicodeWanted(%q, %q) = %v, want %v", tt.installed, tt.want, got, tt.expected)
		}
	}
}

func TestWantPicode(t *testing.T) {
	oldV, oldS := version.Version, version.Stamped
	defer func() { version.Version, version.Stamped = oldV, oldS }()
	version.Version, version.Stamped = "0.3.1", "release"
	if got := WantPicode(); got != "0.3.1" {
		t.Errorf("stamped WantPicode = %q", got)
	}
	version.Stamped = ""
	if got := WantPicode(); got != "" {
		t.Errorf("unstamped WantPicode = %q", got)
	}
}

func TestLinuxAssetName(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		if got, err := LinuxAssetName(arch); err != nil || got != "picode-linux-"+arch {
			t.Errorf("LinuxAssetName(%q) = %q, %v", arch, got, err)
		}
	}
	if _, err := LinuxAssetName("riscv64"); err == nil {
		t.Error("riscv64 should have no asset")
	}
}

func TestWSLPath(t *testing.T) {
	rows := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{`C:\Users\goat\Temp\x`, "/mnt/c/Users/goat/Temp/x", false},
		{`c:/a/b`, "/mnt/c/a/b", false},
		{`D:\x`, "/mnt/d/x", false},
		{`\\server\share`, "", true},
		{`relative\path`, "", true},
		{``, "", true},
	}
	for _, tt := range rows {
		got, err := WSLPath(tt.in)
		if tt.wantErr != (err != nil) || got != tt.want {
			t.Errorf("WSLPath(%q) = %q, %v", tt.in, got, err)
		}
	}
}

func TestParsePiNodeMajor(t *testing.T) {
	doc := []byte(`{"name":"@earendil-works/pi-coding-agent","engines":{"node":">=22.19.0"}}`)
	if got, err := ParsePiNodeMajor(doc); err != nil || got != "22" {
		t.Errorf("ParsePiNodeMajor = %q, %v", got, err)
	}
	if _, err := ParsePiNodeMajor([]byte(`{"engines":{}}`)); err == nil {
		t.Error("missing engines.node should fail")
	}
	if _, err := ParsePiNodeMajor([]byte(`not json`)); err == nil {
		t.Error("bad json should fail")
	}
}

func TestFirstMajor(t *testing.T) {
	for in, want := range map[string]string{">=22.19.0": "22", "^20.0.0": "20", "18": "18"} {
		if got, err := FirstMajor(in); err != nil || got != want {
			t.Errorf("FirstMajor(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := FirstMajor("latest"); err == nil {
		t.Error("constraint without a number should fail")
	}
}

func TestNodeOlder(t *testing.T) {
	rows := []struct {
		installed, required string
		want                bool
	}{
		{"", "22", true},
		{"18", "22", true},
		{"22", "22", false},
		{"24", "22", false},
		{"8", "22", true},
		{"bogus", "22", true},
	}
	for _, tt := range rows {
		if got := NodeOlder(tt.installed, tt.required); got != tt.want {
			t.Errorf("NodeOlder(%q, %q) = %v", tt.installed, tt.required, got)
		}
	}
}

func TestNodeSourceScript(t *testing.T) {
	script, err := NodeSourceScript("22")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"deb.nodesource.com/gpgkey/nodesource-repo.gpg.key",
		"node_22.x",
		"Suites: nodistro",
		"Signed-By: /usr/share/keyrings/nodesource.gpg",
		"Pin-Priority: 600",
		"apt-get install -y nodejs",
		"dpkg --print-architecture | grep",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script lacks %q", want)
		}
	}
	if _, err := NodeSourceScript("22;rm -rf /"); err == nil {
		t.Error("non-numeric major must be refused")
	}
	if _, err := NodeSourceScript(""); err == nil {
		t.Error("empty major must be refused")
	}
}

func TestPiInstallArgs(t *testing.T) {
	args := PiInstallArgs()
	if strings.Join(args, " ") != "install -g @earendil-works/pi-coding-agent@latest" {
		t.Errorf("argv = %q", args)
	}
}

func TestSetRegisteredCommand(t *testing.T) {
	if got := strings.Join(SetRegisteredCommand(), " "); !strings.Contains(got, MarkerPath) {
		t.Errorf("marker command = %q", got)
	}
}

// No `$` crosses the wsl.exe boundary intact (see WSLArgs): a dollar in
// argv reaches the distro shell eaten or wrong. Every script built here is
// asserted literally, so a `$VAR` reintroduction fails here instead of on a
// clean machine with a misleadingly successful exit status.
func TestNoDollarCrossesTheWSLBoundary(t *testing.T) {
	script, err := NodeSourceScript("22")
	if err != nil {
		t.Fatal(err)
	}
	scripts := map[string]string{
		"probe":   strings.Join(ProbeArgs(), " "),
		"create":  strings.Join(CreateUserCommand("goat"), " "),
		"default": strings.Join(SetDefaultUserCommand("goat"), " "),
		"nodesrc": script,
		"place":   PlacePicodeScript("/mnt/c/x"),
		"marker":  strings.Join(SetRegisteredCommand(), " "),
		"userpi":  UserPiInstallScript(),
	}
	for name, script := range scripts {
		if strings.Contains(script, "$") {
			t.Errorf("%s script carries a `$` across wsl.exe:\n%s", name, script)
		}
	}
	// ServerURL and ExportCA build argv at call time, so their calls are
	// captured through the fake runner instead of asserted as strings.
	r := &fakeRunner{replies: [][]byte{
		utf16le(`{"url":"https://localhost:8445"}`),
		utf16le("/home/goat/.local/share/mkcert\n"),
		utf16le("pretend-pem"),
	}}
	if _, err := ServerURL(r, "Ubuntu", "goat"); err != nil {
		t.Fatal(err)
	}
	for _, call := range r.calls {
		if joined := strings.Join(call, " "); strings.Contains(joined, "$") {
			t.Errorf("ServerURL argv carries a `$`: %q", joined)
		}
	}
	ca := &fakeRunner{replies: [][]byte{
		utf16le("/home/goat/.local/share/mkcert\n"),
		utf16le("pretend-pem"),
	}}
	if _, err := ExportCA(ca, "Ubuntu", "goat"); err == nil {
		t.Error("a pem without a certificate was accepted")
	}
	if len(ca.calls) != 2 {
		t.Fatalf("%d wsl calls, want CAROOT read plus cat", len(ca.calls))
	}
	for _, call := range ca.calls {
		if joined := strings.Join(call, " "); strings.Contains(joined, "$") {
			t.Errorf("ExportCA argv carries a `$`: %q", joined)
		}
	}
}

func TestPlacePicodeScript(t *testing.T) {
	got := PlacePicodeScript("/mnt/c/Users/a b/Temp/x")
	for _, want := range []string{"mkdir -p ~/.local/bin", "'/mnt/c/Users/a b/Temp/x'", "~/.local/bin/picode", "chmod +x"} {
		if !strings.Contains(got, want) {
			t.Errorf("script lacks %q:\n%s", want, got)
		}
	}
}

func TestUserPiInstallScript(t *testing.T) {
	got := UserPiInstallScript()
	for _, want := range []string{
		"mkdir -p ~/.npm-global ~/.local/bin",
		"npm config set prefix ~/.npm-global",
		"npm install -g " + pipkg.PiPackage + "@latest",
		"ln -sf ~/.npm-global/bin/pi ~/.local/bin/pi",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("script lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "$") {
		t.Errorf("user pi script carries a `$` across wsl.exe:\n%s", got)
	}
}
