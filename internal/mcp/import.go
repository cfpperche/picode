package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HostKind is one adapter compatibility import (pi-mcp-adapter 2.28).
type HostKind string

const (
	HostCursor        HostKind = "cursor"
	HostClaudeCode    HostKind = "claude-code"
	HostClaudeDesktop HostKind = "claude-desktop"
	HostCodex         HostKind = "codex"
	HostOpenCode      HostKind = "opencode"
	HostWindsurf      HostKind = "windsurf"
	HostVSCode        HostKind = "vscode"
	HostGrok          HostKind = "grok"
)

// HostKinds is adapter order, plus grok (PiCode lists it; adapter ignores unknown imports).
var HostKinds = []HostKind{
	HostCursor, HostClaudeCode, HostClaudeDesktop, HostCodex, HostOpenCode, HostWindsurf, HostVSCode, HostGrok,
}

// ImportResult is POST /api/mcp/import.
type ImportResult struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Already []string `json:"already"`
	Found   []string `json:"found"`
}

// HostInfo is one detected host app for the Use-from picker.
type HostInfo struct {
	Kind    string `json:"kind"`
	Label   string `json:"label"`
	On      bool   `json:"on"`
	Servers int    `json:"servers"`
}

func hostLabel(k HostKind) string {
	switch k {
	case HostCursor:
		return "Cursor"
	case HostClaudeCode:
		return "Claude Code"
	case HostClaudeDesktop:
		return "Claude Desktop"
	case HostCodex:
		return "Codex"
	case HostOpenCode:
		return "OpenCode"
	case HostWindsurf:
		return "Windsurf"
	case HostVSCode:
		return "VS Code"
	case HostGrok:
		return "Grok"
	default:
		return string(k)
	}
}

func hostCandidates(k HostKind, home, cwd string) []string {
	switch k {
	case HostCursor:
		return []string{filepath.Join(home, ".cursor", "mcp.json")}
	case HostClaudeCode:
		return []string{
			filepath.Join(home, ".claude", "mcp.json"),
			filepath.Join(home, ".claude.json"),
			filepath.Join(home, ".claude", "claude_desktop_config.json"),
		}
	case HostClaudeDesktop:
		return []string{filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")}
	case HostCodex:
		return []string{
			filepath.Join(home, ".codex", "config.toml"),
			filepath.Join(home, ".codex", "config.json"),
		}
	case HostOpenCode:
		out := []string{filepath.Join(home, ".config", "opencode", "opencode.json")}
		if cwd != "" {
			out = append(out, filepath.Join(cwd, "opencode.json"))
		}
		return out
	case HostWindsurf:
		return []string{filepath.Join(home, ".windsurf", "mcp.json")}
	case HostVSCode:
		if cwd == "" {
			return nil
		}
		return []string{filepath.Join(cwd, ".vscode", "mcp.json")}
	case HostGrok:
		return []string{filepath.Join(home, ".grok", "config.toml"), filepath.Join(home, ".grok", "mcp.json")}
	default:
		return nil
	}
}

func firstExisting(paths []string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		st, err := os.Stat(p)
		if err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

// DetectHosts returns adapter import kinds whose files exist. Existence only.
func DetectHosts(p Paths) []HostKind {
	home := p.home()
	if home == "" {
		return nil
	}
	var out []HostKind
	for _, k := range HostKinds {
		if firstExisting(hostCandidates(k, home, p.Cwd)) != "" {
			out = append(out, k)
		}
	}
	return out
}

func importKindsOf(raw map[string]any) []string {
	if raw == nil {
		return nil
	}
	arr, ok := raw["imports"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	seen := map[string]bool{}
	for _, v := range arr {
		s, _ := v.(string)
		s = string(HostKind(s))
		if s == "" || seen[s] || !knownHost(HostKind(s)) {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func knownHost(k HostKind) bool {
	for _, x := range HostKinds {
		if x == k {
			return true
		}
	}
	return false
}

func hostServerCount(path string) int {
	if strings.HasSuffix(path, ".toml") {
		return len(serversFromTomlFile(path))
	}
	raw, err := readFile(path)
	if err != nil || raw == nil {
		return 0
	}
	return len(serversOfHost(raw))
}

// FoundHosts is apps we can actually read servers from (or already mirrored).
func FoundHosts(p Paths) []HostInfo {
	on := map[string]bool{}
	if raw, err := readFile(p.PiGlobal()); err == nil && raw != nil {
		for _, s := range importKindsOf(raw) {
			on[s] = true
		}
	}
	home := p.home()
	var out []HostInfo
	for _, k := range HostKinds {
		path := firstExisting(hostCandidates(k, home, p.Cwd))
		if path == "" {
			continue
		}
		n := hostServerCount(path)
		linked := on[string(k)]
		if !linked && n == 0 {
			continue
		}
		out = append(out, HostInfo{Kind: string(k), Label: hostLabel(k), On: linked, Servers: n})
	}
	return out
}

// ImportHosts writes only the requested kinds into ~/.pi/agent/mcp.json.
// kinds must be known and present on this machine. It does not copy files.
func ImportHosts(p Paths, kinds []string) (ImportResult, error) {
	found := DetectHosts(p)
	foundStr := make([]string, len(found))
	foundSet := map[string]bool{}
	for i, k := range found {
		foundStr[i] = string(k)
		foundSet[string(k)] = true
	}
	seen := map[string]bool{}
	var want []string
	for _, s := range kinds {
		k := HostKind(s)
		if !knownHost(k) {
			return ImportResult{}, fmt.Errorf("unknown app %q", s)
		}
		if !foundSet[s] {
			return ImportResult{}, fmt.Errorf("%s is not on this machine", hostLabel(k))
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		want = append(want, s)
	}
	path := p.PiGlobal()
	raw, err := readFileOrEmpty(path)
	if err != nil {
		return ImportResult{}, err
	}
	already := importKindsOf(raw)
	have := map[string]bool{}
	for _, s := range already {
		have[s] = true
	}
	var next []string
	for _, s := range already {
		if !foundSet[s] {
			next = append(next, s)
		}
	}
	var added []string
	for _, s := range want {
		next = append(next, s)
		if !have[s] {
			added = append(added, s)
		}
	}
	var removed []string
	for _, s := range already {
		if foundSet[s] && !seen[s] {
			removed = append(removed, s)
		}
	}
	res := ImportResult{Added: added, Removed: removed, Already: already, Found: foundStr}
	if len(added) == 0 && len(removed) == 0 {
		return res, nil
	}
	if len(next) == 0 {
		delete(raw, "imports")
	} else {
		arr := make([]any, len(next))
		for i, s := range next {
			arr[i] = s
		}
		raw["imports"] = arr
	}
	if err := writeFile(path, raw); err != nil {
		return ImportResult{}, err
	}
	return res, nil
}

func hostLayers(p Paths) []Layer {
	raw, err := readFile(p.PiGlobal())
	if err != nil || raw == nil {
		return nil
	}
	home := p.home()
	var out []Layer
	for _, s := range importKindsOf(raw) {
		k := HostKind(s)
		path := firstExisting(hostCandidates(k, home, p.Cwd))
		if path == "" {
			continue
		}
		out = append(out, Layer{
			ID: "import-" + s, Label: hostLabel(k), Path: path,
			Scope: "import", Writable: false,
		})
	}
	return out
}

func serversOfHost(raw map[string]any) map[string]map[string]any {
	out := serversOf(raw)
	if len(out) > 0 {
		return out
	}
	if raw == nil {
		return out
	}
	for _, key := range []string{"mcp_servers", "mcp"} {
		m, ok := raw[key].(map[string]any)
		if !ok {
			continue
		}
		for name, v := range m {
			if inner, ok := v.(map[string]any); ok {
				out[name] = inner
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}
