// Package pipkg reads and mutates pi packages (ADR-0010).
// User/project stay in pi settings.json. Agent extras are on the agent row
// and passed as `pi -e` (not copied into settings.json).
package pipkg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const Gallery = "https://pi.dev/packages"

// Pkg is one configured package from pi settings.
type Pkg struct {
	Source        string `json:"source"`
	Scope         string `json:"scope"` // user | project | agent
	Kind          string `json:"kind"`  // npm | git | path
	Filtered      bool   `json:"filtered,omitempty"`
	InstalledPath string `json:"installedPath,omitempty"`
}

// Capabilities are facts derived from installed sources (for later UI gates).
type Capabilities struct {
	WebSearch bool `json:"webSearch"`
}

// Report is GET /api/packages.
type Report struct {
	Packages     []Pkg        `json:"packages"`
	Capabilities Capabilities `json:"capabilities"`
	Gallery      string       `json:"gallery"`
}

// UserDir is ~/.pi/agent.
func UserDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent")
}

// ValidSource rejects empty values and shell metacharacters.
func ValidSource(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("source is required")
	}
	if len(s) > 400 {
		return fmt.Errorf("source is too long")
	}
	for _, r := range s {
		if r == ';' || r == '|' || r == '&' || r == '$' || r == '`' || r == '<' || r == '>' ||
			r == '\n' || r == '\r' || r == '\t' || unicode.IsSpace(r) {
			return fmt.Errorf("source has invalid characters")
		}
	}
	return nil
}

// KindOf classifies a pi package source string.
func KindOf(source string) string {
	s := strings.TrimSpace(source)
	switch {
	case strings.HasPrefix(s, "npm:"):
		return "npm"
	case strings.HasPrefix(s, "git:"),
		strings.HasPrefix(s, "https://"),
		strings.HasPrefix(s, "http://"),
		strings.HasPrefix(s, "ssh://"),
		strings.HasPrefix(s, "git@"):
		return "git"
	case strings.HasPrefix(s, "/") || strings.HasPrefix(s, ".") || strings.HasPrefix(s, "~"):
		return "path"
	default:
		// bare npm name: pi-web-search
		return "npm"
	}
}

// DetectWebSearch is true when a configured source is a known search package.
func DetectWebSearch(pkgs []Pkg) bool {
	for _, p := range pkgs {
		s := strings.ToLower(p.Source)
		if strings.Contains(s, "web-search") || strings.Contains(s, "websearch") || strings.Contains(s, "brave-search") {
			return true
		}
	}
	return false
}

// WithAgent appends packages remembered on one PiCode agent (pi -e).
func WithAgent(rep Report, sources []string) Report {
	for _, s := range sources {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		rep.Packages = append(rep.Packages, Pkg{Source: s, Scope: "agent", Kind: KindOf(s)})
	}
	rep.Capabilities.WebSearch = DetectWebSearch(rep.Packages)
	return rep
}

// List reads user (and optional project) settings. Missing files are empty.
func List(userDir, projectDir string) (Report, error) {
	rep := Report{Packages: []Pkg{}, Gallery: Gallery}
	user, err := readSettingsPackages(filepath.Join(userDir, "settings.json"), "user", userDir)
	if err != nil {
		return rep, err
	}
	rep.Packages = append(rep.Packages, user...)
	if projectDir != "" {
		proj, err := readSettingsPackages(filepath.Join(projectDir, ".pi", "settings.json"), "project", filepath.Join(projectDir, ".pi"))
		if err != nil {
			return rep, err
		}
		rep.Packages = append(rep.Packages, proj...)
	}
	rep.Capabilities.WebSearch = DetectWebSearch(rep.Packages)
	return rep, nil
}

func readSettingsPackages(path, scope, baseDir string) ([]Pkg, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw struct {
		Packages []json.RawMessage `json:"packages"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("packages: parse %s: %w", path, err)
	}
	out := make([]Pkg, 0, len(raw.Packages))
	for _, item := range raw.Packages {
		p, ok := parseEntry(item)
		if !ok {
			continue
		}
		p.Scope = scope
		p.Kind = KindOf(p.Source)
		p.InstalledPath = existingInstallPath(p, baseDir)
		out = append(out, p)
	}
	return out, nil
}

func parseEntry(raw json.RawMessage) (Pkg, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			return Pkg{}, false
		}
		return Pkg{Source: s}, true
	}
	var obj struct {
		Source     string   `json:"source"`
		Extensions []string `json:"extensions"`
		Skills     []string `json:"skills"`
		Prompts    []string `json:"prompts"`
		Themes     []string `json:"themes"`
	}
	if json.Unmarshal(raw, &obj) != nil || strings.TrimSpace(obj.Source) == "" {
		return Pkg{}, false
	}
	filtered := obj.Extensions != nil || obj.Skills != nil || obj.Prompts != nil || obj.Themes != nil
	return Pkg{Source: strings.TrimSpace(obj.Source), Filtered: filtered}, true
}

func existingInstallPath(p Pkg, baseDir string) string {
	var cand string
	switch p.Kind {
	case "npm":
		name := strings.TrimPrefix(p.Source, "npm:")
		if i := strings.Index(name, "@"); i > 0 && !strings.HasPrefix(name, "@") {
			name = name[:i]
		}
		// scoped @org/pkg@1.0 — keep @org/pkg
		if strings.HasPrefix(name, "@") {
			if i := strings.LastIndex(name, "@"); i > 0 {
				name = name[:i]
			}
		}
		cand = filepath.Join(baseDir, "npm", "node_modules", name)
	case "path":
		cand = p.Source
		if strings.HasPrefix(cand, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				cand = filepath.Join(home, cand[2:])
			}
		}
	default:
		return ""
	}
	if st, err := os.Stat(cand); err == nil && st.IsDir() {
		return cand
	}
	return ""
}

// MutateOpts selects user vs project (`-l`) install/remove.
type MutateOpts struct {
	Local bool
	Cwd   string
}

// MutateArgs is the pi CLI argv after the binary (tested).
func MutateArgs(verb, source string, local bool) []string {
	if local {
		return []string{verb, "-l", source, "--no-approve"}
	}
	return []string{verb, source, "--no-approve"}
}

func runPi(ctx context.Context, piCmd, verb, source string, opts MutateOpts) error {
	if err := ValidSource(source); err != nil {
		return err
	}
	if opts.Local && strings.TrimSpace(opts.Cwd) == "" {
		return fmt.Errorf("project scope needs a workspace")
	}
	if piCmd == "" {
		piCmd = "pi"
	}
	cmd := exec.CommandContext(ctx, piCmd, MutateArgs(verb, source, opts.Local)...)
	if opts.Cwd != "" {
		cmd.Dir = opts.Cwd
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("%s", firstLine(msg))
	}
	return nil
}

// Install runs `pi install <source> --no-approve` (add `-l` when opts.Local).
func Install(ctx context.Context, piCmd, source string, opts MutateOpts) error {
	return runPi(ctx, piCmd, "install", source, opts)
}

// Remove runs `pi remove <source> --no-approve`.
func Remove(ctx context.Context, piCmd, source string, opts MutateOpts) error {
	return runPi(ctx, piCmd, "remove", source, opts)
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
