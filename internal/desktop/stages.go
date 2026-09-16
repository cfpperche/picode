package desktop

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/pipkg"
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
// absence stays the provision cert step's existing blocked message.
var RuntimeTools = []string{"tmux", "git", "curl", "node", "npm", "pi"}

// BasePackages is the apt half of the runtime. gnupg is not in ADR-0098's
// list; the NodeSource keyring needs it and a minimal image may not have it.
var BasePackages = []string{"tmux", "git", "curl", "ca-certificates", "gnupg"}

// probeScript observes one distro in a single wsl call: the installed
// picode version, the missing runtime tools, the installed node major, the
// distro family, and the registered marker. Sections are @@-fenced so one
// failing probe cannot corrupt another's parse.
const probeScript = `set -e
echo "@@picode@@"
command -v picode >/dev/null 2>&1 && picode --version || echo MISSING
echo "@@tools@@"
for t in tmux git curl node npm pi; do
  command -v "$t" >/dev/null 2>&1 || echo "missing:$t"
done
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

// PiRegistryURL is where the runtime stage reads which node major pi's
// package declares in engines. Tests replace it.
var PiRegistryURL = "https://registry.npmjs.org/" + url.PathEscape(pipkg.PiPackage) + "/latest"

// ParsePiNodeMajor reads the engines.node constraint out of a registry
// document and returns its first major ("22" for ">=22.19.0"). The executor
// re-runs the whole stage on failure, so an error here must say so.
func ParsePiNodeMajor(doc []byte) (string, error) {
	engines := struct {
		Engines map[string]string `json:"engines"`
	}{}
	if err := json.Unmarshal(doc, &engines); err != nil {
		return "", fmt.Errorf("read pi's node engine: %w", err)
	}
	constraint, ok := engines.Engines["node"]
	if !ok || constraint == "" {
		return "", fmt.Errorf("pi's registry entry names no node engine")
	}
	return FirstMajor(constraint)
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
// Runs as root; major must be digits.
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
mkdir -p /usr/share/keyrings
rm -f /usr/share/keyrings/nodesource.gpg /etc/apt/sources.list.d/nodesource.list /etc/apt/sources.list.d/nodesource.sources
curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /usr/share/keyrings/nodesource.gpg
chmod 644 /usr/share/keyrings/nodesource.gpg
arch=$(dpkg --print-architecture)
case "$arch" in amd64|arm64) ;; *) echo "unsupported architecture: $arch" >&2; exit 1;; esac
printf 'Types: deb\nURIs: https://deb.nodesource.com/node_%s.x\nSuites: nodistro\nComponents: main\nArchitectures: %%s\nSigned-By: /usr/share/keyrings/nodesource.gpg\n' "$arch" > /etc/apt/sources.list.d/nodesource.sources
printf 'Package: nodejs\nPin: origin deb.nodesource.com\nPin-Priority: 600\n' > /etc/apt/preferences.d/nodejs
apt-get update
apt-get install -y nodejs
`, major), nil
}

// PiInstallArgs is the exact argv ADR-0093 runs for pi.
func PiInstallArgs() []string {
	return []string{"install", "-g", pipkg.PiPackage + "@latest"}
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
