// Package clilifecycle orchestrates the lifecycle commands the CLIs already
// ship: update check, update, reinstall, uninstall (ADR-0087). PiCode drives
// the vendors' own commands; it does not implement their package managers.
package clilifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Method classifies how the resolved CLI executable was installed.
type Method string

const (
	MethodNpm     Method = "npm"
	MethodNative  Method = "native"
	MethodVendor  Method = "vendor"
	MethodGit     Method = "git"
	MethodUnknown Method = "unknown"
)

// Action is a mutating lifecycle operation.
type Action string

const (
	ActionInstall   Action = "install"
	ActionUpdate    Action = "update"
	ActionReinstall Action = "reinstall"
	ActionUninstall Action = "uninstall"
)

// UninstallKind says what uninstall support exists for a CLI + method pair.
type UninstallKind string

const (
	UninstallNone   UninstallKind = "" // lifecycle not offered
	UninstallVendor UninstallKind = "vendor"
	UninstallNpm    UninstallKind = "npm"
	UninstallGuided UninstallKind = "guided"
)

// Plan pins the exact argv PiCode runs for one CLI + install method. The
// vectors were verified against each vendor's --help output on 2026-09-06;
// the tests assert the recorded shapes.
type Plan struct {
	CLI           string   `json:"cli"`
	Method        Method   `json:"method"`
	NpmPackage    string   `json:"npmPackage,omitempty"`
	LatestFrom    string   `json:"latestFrom"` // "npm" or "vendor"
	UpdateViaNpm  bool     `json:"updateViaNpm,omitempty"`
	CheckArgs     []string `json:"checkArgs,omitempty"`
	UpdateArgs    []string `json:"updateArgs,omitempty"`
	ReinstallArgs []string `json:"reinstallArgs,omitempty"`
	// UpdateEnv overlays env for update and reinstall (Muse's launcher
	// entrypoint). An action is offered when its argv OR env is present.
	UpdateEnv     []string      `json:"updateEnv,omitempty"`
	Uninstall     UninstallKind `json:"uninstall"`
	UninstallArgs []string      `json:"uninstallArgs,omitempty"`
	Docs          string        `json:"docs,omitempty"`
}

type spec struct {
	npmPackage    string
	updateArgs    []string
	reinstallArgs []string
	// updateEnv overlays process env for update and reinstall. Muse ships
	// no update argv: its launcher updates when MUSE_LAUNCHER_INSTALL=1
	// (verified against the installed launcher script, 2026-09-16).
	updateEnv     []string
	uninstall     UninstallKind
	uninstallArgs []string
	docs          string
	latestFrom    string // "channel" for HTTP version checks without a vendor command
	// checkArgs is the vendor's own check command for LatestFrom
	// "vendor" (omp update --check, measured 2026-09-17).
	checkArgs []string
	// npmUpdateOnly marks CLIs whose vendor update command refuses on
	// npm-managed installs (claude update → "use npm"). Their update and
	// reinstall go through npm instead.
	npmUpdateOnly bool
	// vendorOnNpm keeps vendor update/reinstall/uninstall argv when
	// DetectMethod returns npm. OpenCode's bun global lives under
	// node_modules, but `opencode upgrade` / `opencode uninstall` are the
	// installer-aware commands; npm install/remove -g would miss bun.
	vendorOnNpm bool
}

// plans maps each catalogued CLI to its vendor lifecycle shapes. npm is
// filled per method in For(); vendor entries stay nil there.
var plans = map[string]spec{
	"pi": {
		// npm installs update through npm. Measured 2026-09-20: `pi update`
		// on an npm-global copy refuses when the package dir is not W_OK
		// (a sudo npm install into an nvm prefix) and prints the npm
		// command instead of running it. Same npmUpdateOnly shape as
		// claude-code / omp.
		npmPackage:    "@earendil-works/pi-coding-agent",
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"update", "--force"},
		npmUpdateOnly: true,
	},
	"omp": {
		// Same shape as pi — omp is its fork — with the real npm package
		// name. npm installs update through npm (deterministic target; the
		// vendor updater's npm shim handling is a known weak spot), native
		// installs through the vendor's own updater, whose update check
		// output is measured: "Current version: X" plus "New version
		// available: Y" when one exists.
		npmPackage:    "@oh-my-pi/pi-coding-agent",
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"update", "--force"},
		uninstall:     UninstallGuided,
		docs:          "https://omp.sh/docs",
		npmUpdateOnly: true,
		latestFrom:    "vendor",
		checkArgs:     []string{"update", "--check"},
	},
	"claude-code": {
		npmPackage:    "@anthropic-ai/claude-code",
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"install"},
		uninstall:     UninstallGuided,
		docs:          "https://code.claude.com/docs/en/setup#uninstall-claude-code",
		npmUpdateOnly: true,
	},
	"codex": {
		npmPackage:    "@openai/codex",
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"update"},
	},
	"grok": {
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"update", "--force-reinstall"},
		uninstall:     UninstallGuided,
		docs:          "https://docs.x.ai/build/cli/reference",
	},
	"hermes": {
		updateArgs:    []string{"update", "--yes"},
		reinstallArgs: []string{"update", "--force", "--yes"},
		uninstall:     UninstallVendor,
		uninstallArgs: []string{"uninstall", "--yes"},
	},
	"opencode": {
		npmPackage:    "opencode-ai",
		updateArgs:    []string{"upgrade"},
		reinstallArgs: []string{"upgrade"},
		uninstall:     UninstallVendor,
		uninstallArgs: []string{"uninstall", "--keep-config", "--keep-data", "--force"},
		vendorOnNpm:   true,
	},
	"muse": {
		latestFrom: "channel",
		// No update argv: MUSE_LAUNCHER_INSTALL=1 turns a bare run into
		// the vendor updater (it exits 0 before arg parsing).
		updateEnv: []string{"MUSE_LAUNCHER_INSTALL=1"},
		uninstall: UninstallGuided,
		docs:      "https://dev.meta.ai/docs/muse-code",
	},
	"agy": {
		latestFrom:    "channel",
		updateArgs:    []string{"update"},
		reinstallArgs: []string{"update"},
		uninstall:     UninstallGuided,
		docs:          "https://antigravity.google/docs/cli/",
	},
}

// DetectMethod classifies an executable path. npm-installed binaries
// resolve into a node_modules tree; the native/vendor/git shapes come from
// the observed 2026-09 install layouts. Real installs are symlinks
// (~/.local/bin/claude → ~/.local/share/claude/versions/N) or wrapper
// scripts (hermes execs the venv path), so the link is resolved first and
// the script head is scanned before giving up.
func DetectMethod(path string) Method {
	if path == "" {
		return MethodUnknown
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		resolved = path
	}
	if m := classifyMethod(filepath.ToSlash(resolved)); m != MethodUnknown {
		return m
	}
	if f, err := os.Open(path); err == nil {
		head := make([]byte, 4096)
		n, _ := f.Read(head)
		f.Close()
		if m := classifyMethod(filepath.ToSlash(string(head[:n]))); m != MethodUnknown {
			return m
		}
	}
	return MethodUnknown
}

func classifyMethod(p string) Method {
	switch {
	case strings.Contains(p, "/.bun/") && strings.Contains(p, "/@oh-my-pi/"):
		// Measured 2026-09-17: a bun-global omp's realpath carries
		// node_modules, so the npm pattern below would claim it — but npm
		// commands miss the bun copy, and the vendor updater resolves its
		// target by PATH (run from a bun install it updated an npm copy).
		// No deterministic mutation target: honestly unknown. (OpenCode's
		// bun installs keep classifying npm: its vendor commands are
		// installer-aware.)
		return MethodUnknown
	case strings.Contains(p, "/node_modules/"):
		return MethodNpm
	case strings.Contains(p, "/.local/share/claude/"), strings.Contains(p, "/claude/versions/"):
		return MethodNative
	case strings.Contains(p, "/.local/bin/omp"):
		// The vendor curl installer lands a single native binary here
		// (PI_INSTALL_DIR can move it; this is the measured default).
		return MethodVendor
	case strings.Contains(p, "/.grok/"):
		return MethodVendor
	case strings.Contains(p, "/.hermes/"):
		return MethodGit
	case strings.Contains(p, "api.meta.ai/muse-code"), strings.Contains(p, "muse-code/launcher"), strings.Contains(p, "/muse-bin-"):
		return MethodVendor
	case filepath.Base(p) == "agy":
		// Google's installer writes $HOME/.local/bin/agy and nothing else
		// identifies the layout; the vendor command name does.
		return MethodVendor
	default:
		return MethodUnknown
	}
}

// For returns the lifecycle plan for one CLI and install method. ok is false
// when the pair offers no managed lifecycle (unknown method, or a method the
// vendor command cannot serve).
func For(cliID string, m Method) (Plan, bool) {
	s, ok := plans[cliID]
	if !ok || m == MethodUnknown {
		return Plan{CLI: cliID, Method: m, Uninstall: UninstallNone}, false
	}
	p := Plan{CLI: cliID, Method: m, Uninstall: s.uninstall, Docs: s.docs}
	if m == MethodNpm && s.npmUpdateOnly {
		// Vendor update refuses here; npm serves update/reinstall/uninstall.
		p.NpmPackage = s.npmPackage
		p.LatestFrom = "npm"
		p.UpdateViaNpm = true
		p.UpdateArgs = []string{"install", "-g", s.npmPackage + "@latest"}
		p.ReinstallArgs = p.UpdateArgs
		p.Uninstall = UninstallNpm
		p.UninstallArgs = []string{"remove", "-g", s.npmPackage}
		p.Docs = ""
		return p, true
	}
	if m == MethodNpm && s.vendorOnNpm {
		if s.npmPackage == "" {
			return Plan{CLI: cliID, Method: m, Uninstall: UninstallNone}, false
		}
		p.NpmPackage = s.npmPackage
		p.LatestFrom = "npm"
		p.UpdateArgs = s.updateArgs
		p.ReinstallArgs = s.reinstallArgs
		p.Uninstall = s.uninstall
		p.UninstallArgs = s.uninstallArgs
		return p, true
	}
	if m == MethodNpm {
		if s.npmPackage == "" {
			return Plan{CLI: cliID, Method: m, Uninstall: UninstallNone}, false
		}
		p.NpmPackage = s.npmPackage
		p.LatestFrom = "npm"
		// The vendor command stays first choice for update; npm guarantees a
		// fresh install for reinstall and owns uninstall of its own package.
		p.UpdateArgs = s.updateArgs
		p.ReinstallArgs = []string{"install", "-g", s.npmPackage + "@latest"}
		p.Uninstall = UninstallNpm
		p.UninstallArgs = []string{"remove", "-g", s.npmPackage}
		return p, true
	}
	if m != MethodNative && m != MethodVendor && m != MethodGit {
		return Plan{CLI: cliID, Method: m, Uninstall: UninstallNone}, false
	}
	if len(s.updateArgs) == 0 && s.latestFrom == "" {
		return Plan{CLI: cliID, Method: m, Uninstall: UninstallNone}, false
	}
	p.LatestFrom = "npm"
	if s.npmPackage != "" {
		p.NpmPackage = s.npmPackage
	}
	if s.latestFrom != "" {
		p.LatestFrom = s.latestFrom
	}
	p.CheckArgs = s.checkArgs
	switch cliID {
	case "grok":
		p.LatestFrom = "vendor"
		p.CheckArgs = []string{"update", "--check", "--json"}
	case "hermes":
		p.LatestFrom = "vendor"
		p.CheckArgs = []string{"update", "--check"}
	}
	p.UpdateArgs = s.updateArgs
	p.ReinstallArgs = s.reinstallArgs
	p.UpdateEnv = s.updateEnv
	p.UninstallArgs = s.uninstallArgs
	return p, true
}

// CanUpdate reports whether the plan offers an update. Muse offers an
// env-only entrypoint, so env presence counts alongside argv.
func (p Plan) CanUpdate() bool {
	return len(p.UpdateArgs) > 0 || len(p.UpdateEnv) > 0
}

// CanReinstall reports whether the plan offers a reinstall.
func (p Plan) CanReinstall() bool {
	return len(p.ReinstallArgs) > 0 || len(p.UpdateEnv) > 0
}

// Args returns the argv for one action, or an error when the plan does not
// offer it. Uninstall args are empty for guided plans — the server refuses
// and the UI shows the guided steps instead.
func (p Plan) Args(a Action) ([]string, error) {
	switch a {
	case ActionInstall:
		if p.NpmPackage == "" {
			return nil, fmt.Errorf("%s has no managed install; follow its documentation.", p.CLI)
		}
		return []string{"install", "-g", p.NpmPackage + "@latest"}, nil
	case ActionUpdate:
		if !p.CanUpdate() {
			return nil, fmt.Errorf("%s cannot be updated through PiCode for this install.", p.CLI)
		}
		return p.UpdateArgs, nil
	case ActionReinstall:
		if !p.CanReinstall() {
			return nil, fmt.Errorf("%s cannot be reinstalled through PiCode for this install.", p.CLI)
		}
		return p.ReinstallArgs, nil
	case ActionUninstall:
		switch p.Uninstall {
		case UninstallVendor, UninstallNpm:
			return p.UninstallArgs, nil
		default:
			return nil, fmt.Errorf("%s has no uninstall command; remove it following its documentation.", p.CLI)
		}
	default:
		return nil, fmt.Errorf("Unknown lifecycle action.")
	}
}

// ExtractSemver pulls the first dotted-numeric token from a vendor's
// version line ("pi 1.0.0", "2.1.263 (Claude Code)", "grok 1.0.13 (hash)").
func ExtractSemver(line string) string {
	start, end := -1, -1
	for i := 0; i <= len(line); i++ {
		ch := byte(' ')
		if i < len(line) {
			ch = line[i]
		}
		numeric := ch >= '0' && ch <= '9' || ch == '.'
		if numeric && start < 0 {
			start = i
		}
		if !numeric && start >= 0 {
			end = i
			break
		}
	}
	if start < 0 || end < 0 {
		return ""
	}
	return line[start:end]
}

// ForMissing returns the install plan for a CLI that is not installed.
// Only npm-backed CLIs get a managed install (same argv as reinstall);
// vendor curl installers stay guided and return ok=false with the docs URL.
func ForMissing(cliID string) (Plan, bool) {
	s, ok := plans[cliID]
	if !ok || s.npmPackage == "" {
		return Plan{CLI: cliID, Method: MethodUnknown, Uninstall: UninstallNone}, false
	}
	argv := []string{"install", "-g", s.npmPackage + "@latest"}
	return Plan{
		CLI:           cliID,
		Method:        MethodNpm,
		NpmPackage:    s.npmPackage,
		LatestFrom:    "npm",
		UpdateViaNpm:  true,
		UpdateArgs:    argv,
		ReinstallArgs: argv,
		Uninstall:     UninstallNpm,
		UninstallArgs: []string{"remove", "-g", s.npmPackage},
	}, true
}

// InstallDocs is the vendor's official install guide for CLIs whose
// installer PiCode does not execute.
func InstallDocs(cliID string) string {
	switch cliID {
	case "grok":
		return "https://docs.x.ai/build/cli/reference"
	case "hermes":
		return "https://hermes-agent.nousresearch.com/docs/getting-started/installation"
	case "claude-code":
		return "https://code.claude.com/docs/en/setup"
	case "muse":
		return "https://dev.meta.ai/docs/muse-code"
	case "agy":
		return "https://antigravity.google/docs/cli/"
	default:
		return ""
	}
}

// GrokCheck mirrors `grok update --check --json` (verified 2026-09-06,
// grok 1.0.13: currentVersion, latestVersion, updateAvailable, installer,
// channel, autoUpdate, error).
type GrokCheck struct {
	CurrentVersion  string  `json:"currentVersion"`
	LatestVersion   string  `json:"latestVersion"`
	UpdateAvailable bool    `json:"updateAvailable"`
	Installer       string  `json:"installer"`
	Channel         string  `json:"channel"`
	AutoUpdate      bool    `json:"autoUpdate"`
	Error           *string `json:"error"`
}

func ParseGrokCheck(data []byte) (GrokCheck, error) {
	var c GrokCheck
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("grok update --check returned unreadable output")
	}
	if c.Error != nil && *c.Error != "" {
		return c, fmt.Errorf("grok update --check: %s", *c.Error)
	}
	if c.LatestVersion == "" {
		return c, fmt.Errorf("grok update --check returned no latest version")
	}
	return c, nil
}

// ParseHermesCheck reads `hermes update --check` text (verified 2026-09-06,
// Hermes Agent v0.18.2: "Update available (behind origin/main)" /
// "Run 'hermes update' to install."). Hermes does not print a latest
// version, so the caller shows availability without a number.
func ParseHermesCheck(out string) (bool, error) {
	lower := strings.ToLower(out)
	switch {
	case strings.Contains(lower, "update available"):
		return true, nil
	case strings.Contains(lower, "up to date"), strings.Contains(lower, "up-to-date"),
		strings.Contains(lower, "no update"), strings.Contains(lower, "already"):
		return false, nil
	default:
		return false, fmt.Errorf("hermes update --check returned unrecognized output")
	}
}

// ParseOmpCheck reads `omp update --check` text (measured on omp 18.2.4,
// 2026-09-17):
//
//	Current version: 18.2.4\n✔ Already up to date
//	Current version: 18.2.3\nNew version available: 18.2.4
//
// The latest version appears only when one exists; an up-to-date check
// returns ("", nil).
func ParseOmpCheck(out string) (string, error) {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(line, "New version available: "); ok {
			v = strings.TrimSpace(v)
			if v == "" {
				return "", fmt.Errorf("omp update --check returned an empty version")
			}
			return v, nil
		}
	}
	if !strings.Contains(out, "Already up to date") {
		return "", fmt.Errorf("omp update --check returned unreadable output")
	}
	return "", nil
}

// MuseChannelURL is the stable Muse Code release channel (verified 2026-09-14).
const MuseChannelURL = "https://api.meta.ai/muse-code/channels/muse-stable"

// AntigravityManifestURL is the release manifest the vendor's installer reads
// (verified 2026-09-14 through antigravity.google/cli/install.sh). The file
// carries `version`, `url` and `sha512`; PiCode reads only the version.
const AntigravityManifestURL = "https://antigravity-cli-auto-updater-974169037036.us-central1.run.app/manifests/"

// AntigravityPlatform mirrors the installer's platform string: linux_amd64,
// linux_arm64_musl, darwin_arm64 …
func AntigravityPlatform(goos, goarch string, musl bool) string {
	if goos == "linux" && musl {
		return goos + "_" + goarch + "_musl"
	}
	return goos + "_" + goarch
}

// ChannelURLFor returns the read-only version source for a CLI whose latest
// version lives in a vendor channel or manifest. No command runs, so nothing
// can install or change anything.
func ChannelURLFor(cliID, goos, goarch string, musl bool) (string, bool) {
	switch cliID {
	case "muse":
		return MuseChannelURL, true
	case "agy":
		return AntigravityManifestURL + AntigravityPlatform(goos, goarch, musl) + ".json", true
	default:
		return "", false
	}
}

// ChannelVersion is the `version` field a vendor channel or manifest carries.
type ChannelVersion struct {
	Version string `json:"version"`
}

func ParseChannelVersion(data []byte) (ChannelVersion, error) {
	var c ChannelVersion
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("the version channel returned unreadable output")
	}
	if c.Version == "" {
		return c, fmt.Errorf("the version channel returned no version")
	}
	return c, nil
}

// ExtractMuseVersion prefers the launcher token (1.2.1-R2847.1) over the
// leading dotted semver in `muse --version`.
func ExtractMuseVersion(line string) string {
	best := ""
	s := line
	for {
		i := strings.Index(s, "-R")
		if i < 0 {
			break
		}
		start := i
		for start > 0 && (s[start-1] >= '0' && s[start-1] <= '9' || s[start-1] == '.') {
			start--
		}
		end := i + 2
		for end < len(s) && (s[end] >= '0' && s[end] <= '9' || s[end] == '.') {
			end++
		}
		cand := s[start:end]
		if strings.Count(cand, ".") >= 2 && strings.Contains(cand, "-R") {
			best = cand
		}
		s = s[i+2:]
	}
	if best != "" {
		return best
	}
	return ExtractSemver(line)
}
