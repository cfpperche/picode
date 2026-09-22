package desktop

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/version"
)

// This file is the probe-and-build half of ADR-0098's install-picode and
// install-runtime stages: one observation script, its parser, and the
// command builders the executors run. Everything here is pure so it tests
// from Linux; the executors in cmd/picode-desktop own the side effects.

// MarkerPath records that this program registered the distro. The runtime
// stage installs packages without asking there, and asks first everywhere
// else. A file inside the distro (not on Windows) survives the reboot
// resume, which runs in a new process that remembers nothing.
const MarkerPath = "/etc/picode-desktop-registered"

// RuntimeTools is what install-runtime converges, in display order. mkcert
// is deliberately absent: it is a best effort inside the apt phase, and its
// absence stays the provision cert step's existing blocked message. No agent
// CLI is in the list (ADR-0179): the user installs the ones they use from
// Agent CLIs after the first login; node and npm are here for that.
var RuntimeTools = []string{"tmux", "git", "curl", "node", "npm"}

// RuntimeNodeMajor is the Node.js major the runtime stage installs when node
// or npm is missing — the version CI builds with (.github/workflows), not a
// number read from any CLI's package. A node already present is left alone.
const RuntimeNodeMajor = "22"

// NpmUserPrefix names the probe's "the account's npm installs need root"
// item: a global prefix under /usr (NodeSource's, or the distro's) is
// root-owned, and Agent CLIs' Install runs `npm install -g` as the account.
// The runtime stage points that account's prefix at ~/.local, whose bin the
// login shell already carries (ADR-0179). An nvm prefix is left alone.
const NpmUserPrefix = "npm-user-prefix"

// NodeUpgrade names the probe's "node is older than RuntimeNodeMajor" item:
// the CLIs a user installs later (Pi needs >=22.19) refuse an older engine.
var NodeUpgrade = "node-" + RuntimeNodeMajor

// NpmUserPrefixScript sets the account's npm prefix to ~/.local. It runs as
// that account; no `$` crosses the wsl.exe boundary.
func NpmUserPrefixScript() string {
	return "set -e\nmkdir -p ~/.local/bin\nnpm config set prefix ~/.local"
}

// BasePackages is the apt half of the runtime. gnupg is not in ADR-0098's
// list; the NodeSource keyring needs it and a minimal image may not have it.
var BasePackages = []string{"tmux", "git", "curl", "ca-certificates", "gnupg"}

// probeScript observes one distro in a single wsl call: the installed
// picode version, the missing runtime tools, the installed node major, the
// distro family, and the registered marker. Sections are @@-fenced so one
// failing probe cannot corrupt another's parse.
const probeScript = `set -e
echo "@@picode@@"
which picode >/dev/null 2>&1 && picode --version || echo MISSING
echo "@@tools@@"
which tmux >/dev/null 2>&1 || echo "missing:tmux"
which git >/dev/null 2>&1 || echo "missing:git"
which curl >/dev/null 2>&1 || echo "missing:curl"
which node >/dev/null 2>&1 || echo "missing:node"
which npm >/dev/null 2>&1 || echo "missing:npm"
npm config get prefix 2>/dev/null | grep -q '^/usr' && echo "missing:npm-user-prefix"
node --version 2>/dev/null || echo "nodeversion:"
echo "@@family@@"
grep ^ID= /etc/os-release 2>/dev/null || echo ID=unknown
echo "@@marker@@"
test -f /etc/picode-desktop-registered && echo registered || echo adopted`

// ProbeArgs runs the observation script as the distro's default account.
// Every probe is a read; a missing tool is data, not a failure. The login
// shell matters: ~/.local/bin (where the binary lives) is on PATH there,
// and PicodePath resolves through the same shell — probing otherwise would
// report a present picode as missing.
func ProbeArgs() []string { return []string{"sh", "-lc", probeScript} }

// ProbeResult is the parsed observation of one distro.
type ProbeResult struct {
	// Picode is the installed picode version ("", when missing or broken).
	Picode string
	// Missing names the absent RuntimeTools.
	Missing []string
	// NodeMajor is the installed node major ("", when node is missing).
	NodeMajor string
	// Family is the os-release ID in lowercase ("ubuntu", "debian",
	// "unknown", ...).
	Family string
	// Registered is true when this program registered the distro.
	Registered bool
}

// ParseProbe turns one probe run into a result. Unparseable corners degrade
// to "missing" — the stage re-checks before changing anything.
func ParseProbe(out []byte) ProbeResult {
	var r ProbeResult
	r.Family = "unknown"
	sections := map[string][]string{}
	var cur string
	for _, line := range strings.Split(DecodeWindows(out), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(line, "@@") && strings.HasSuffix(line, "@@") {
			cur = line
			continue
		}
		if cur != "" {
			sections[cur] = append(sections[cur], line)
		}
	}
	for _, line := range sections["@@picode@@"] {
		if v := parsePicodeVersionLine(line); v != "" {
			r.Picode = v
		}
	}
	for _, line := range sections["@@tools@@"] {
		if name, ok := strings.CutPrefix(line, "missing:"); ok && name != "" {
			r.Missing = append(r.Missing, name)
		} else if m := parseNodeVersionLine(line); line != "nodeversion:" && m != "" {
			r.NodeMajor = m
		}
	}
	if r.NodeMajor != "" && NodeOlder(r.NodeMajor, RuntimeNodeMajor) {
		r.Missing = append(r.Missing, NodeUpgrade)
	}
	for _, line := range sections["@@family@@"] {
		if id, ok := strings.CutPrefix(line, "ID="); ok {
			r.Family = strings.ToLower(strings.Trim(id, `"'`))
		}
	}
	for _, line := range sections["@@marker@@"] {
		if line == "registered" {
			r.Registered = true
		}
	}
	return r
}

// parsePicodeVersionLine reads `picode --version` ("picode 0.3.1"). A source
// build reports "0.3.1+a892ede"; the revision is stripped because the stage
// compares releases, and a source checkout inside the distro already
// satisfies "picode is installed".
func parsePicodeVersionLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[0] != "picode" {
		return ""
	}
	if v, _, _ := strings.Cut(fields[1], "+"); isDotted(v) {
		return v
	}
	return ""
}

func parseNodeVersionLine(line string) string {
	v := strings.TrimPrefix(strings.TrimSpace(line), "v")
	major, _, _ := strings.Cut(v, ".")
	if major == "" {
		return ""
	}
	for _, r := range major {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return major
}

func isDotted(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

// WantPicode is the picode version install-picode converges on: this
// build's release when stamped, "" when unstamped — an unstamped tool only
// checks presence, and its executor tracks latest when it installs.
func WantPicode() string {
	if version.Stamped == "release" && version.Version != "" {
		return version.Version
	}
	return ""
}

// PicodeWanted reports whether the install-picode stage has work: a missing
// binary always, otherwise a version that disagrees with want ("" want
// means any present binary satisfies).
func PicodeWanted(installed, want string) bool {
	if installed == "" {
		return true
	}
	return want != "" && installed != want
}

// LinuxAssetName is the release asset install-picode downloads for a distro
// on goarch. The release workflow publishes exactly amd64 and arm64.
func LinuxAssetName(goarch string) (string, error) {
	switch goarch {
	case "amd64", "arm64":
		return "picode-linux-" + goarch, nil
	default:
		return "", fmt.Errorf("no Linux picode binary for %s (amd64 and arm64 exist)", goarch)
	}
}

// WSLPath maps a Windows path onto the distro's /mnt view, so a file the
// tool downloaded can be referenced from inside. Only local drive paths
// qualify; UNC and relative paths are refused rather than guessed.
func WSLPath(win string) (string, error) {
	if len(win) < 3 || win[1] != ':' || (win[2] != '\\' && win[2] != '/') {
		return "", fmt.Errorf("not a local Windows path: %q", win)
	}
	drive := win[0]
	if (drive < 'a' || drive > 'z') && (drive < 'A' || drive > 'Z') {
		return "", fmt.Errorf("not a local Windows path: %q", win)
	}
	rest := strings.ReplaceAll(win[2:], "\\", "/")
	return "/mnt/" + strings.ToLower(string(drive)) + rest, nil
}

// SetRegisteredCommand leaves the marker create-user's distro carries, so a
// later stage — possibly after the reboot resume — knows asking is not
// needed. Runs as root next to the account creation.
func SetRegisteredCommand() []string {
	return []string{"sh", "-c", "touch " + MarkerPath}
}

var majorRe = regexp.MustCompile(`[0-9]+`)

// FirstMajor is the first number in a semver constraint (">=", "^" and
// ranges all lead with the lowest supported major).
func FirstMajor(constraint string) (string, error) {
	m := majorRe.FindString(constraint)
	if m == "" {
		return "", fmt.Errorf("no major in node constraint %q", constraint)
	}
	return m, nil
}

// NodeOlder reports whether installed (a major, "" when node is missing)
// is below required. Unparseable input counts as older: reinstalling node
// over a broken one is the fix, not a risk.
func NodeOlder(installed, required string) bool {
	if installed == "" {
		return true
	}
	i, err1 := strconv.Atoi(installed)
	w, err2 := strconv.Atoi(required)
	if err1 != nil || err2 != nil {
		return true
	}
	return i < w
}

// NodeSourceScript adds the NodeSource apt repository for major and
// installs nodejs from it. It replicates the key, source, and pin lines of
// NodeSource's own setup script as our argv instead of piping their script
// (ADR-0093 refuses executing a vendor's curl|bash on the user's behalf),
// and ends with the install so one wsl call converges node end to end.
// Runs as root; major must be digits. No `$`: the architecture gates the
// run through a bare grep (anything else fails closed under set -e), and
// the source file omits Architectures — without it apt resolves the native
// arch, which the gate just proved supported.
func NodeSourceScript(major string) (string, error) {
	if major == "" {
		return "", fmt.Errorf("node major is empty")
	}
	for _, r := range major {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("node major is not a number: %q", major)
		}
	}
	return fmt.Sprintf(`set -e
dpkg --print-architecture | grep -qxE '(amd64|arm64)'
mkdir -p /usr/share/keyrings
rm -f /usr/share/keyrings/nodesource.gpg /etc/apt/sources.list.d/nodesource.list /etc/apt/sources.list.d/nodesource.sources
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /usr/share/keyrings/nodesource.gpg
chmod 644 /usr/share/keyrings/nodesource.gpg
echo 'Types: deb' > /etc/apt/sources.list.d/nodesource.sources
echo 'URIs: https://deb.nodesource.com/node_%s.x' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Suites: nodistro' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Components: main' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Signed-By: /usr/share/keyrings/nodesource.gpg' >> /etc/apt/sources.list.d/nodesource.sources
echo 'Package: nodejs' > /etc/apt/preferences.d/nodejs
echo 'Pin: origin deb.nodesource.com' >> /etc/apt/preferences.d/nodejs
echo 'Pin-Priority: 600' >> /etc/apt/preferences.d/nodejs
apt-get update
apt-get install -y nodejs
`, major), nil
}

// PlacePicodeScript moves a staged binary into ~/.local/bin/picode. inside
// is the /mnt path of a file the tool downloaded on Windows; it is quoted
// here because the temp path carries the Windows account name.
func PlacePicodeScript(inside string) string {
	return "set -e\nmkdir -p ~/.local/bin\ncp " + shQuote(inside) + " ~/.local/bin/picode\nchmod +x ~/.local/bin/picode"
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
