package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// Grok manages Grok CLI's MCP servers in its native TOML configs: the user
// file (~/.grok/config.toml) and the workspace file
// (<workspace>/.grok/config.toml). Both are plain files, so PiCode edits
// them directly and never shells out.
//
// Writes share the surgical TOML splice of the package (toml.go, ADR-0150):
// only the byte span of the `[mcp_servers.<name>]` table is replaced or
// deleted, so comments, unrelated tables and ${VAR}/${VAR:-default}
// placeholders survive byte-for-byte, and Grok's own entry fields
// (startup_timeout_sec, tool_timeout_sec, …) survive an update through
// merge-by-key.
//
// Grok's config has no per-server switch, so Toggle is refused and the pane
// (driver toggle: "none") renders no switch at all.
type Grok struct{}

const grokBin = "grok"

// ToggleRefusal is the honest answer to an enable/disable ask: Grok loads
// every configured server; there is no flag to flip.
const grokToggleRefusal = "Grok connectors have no on/off switch; remove and re-add instead"

func (Grok) ID() string { return "grok" }

func (Grok) Bin() string { return grokBin }

// AuthHint: Grok signs in through its own first use of a server — there is
// no headless sign-in command, so the hint is plain text with nothing to
// copy.
func (Grok) AuthHint(name string) AuthHint {
	return AuthHint{Text: "sign in happens on first use inside Grok"}
}

type grokPaths struct{ Paths }

func (c grokPaths) user() string {
	return filepath.Join(c.home(), ".grok", "config.toml")
}

func (c grokPaths) project() string {
	if c.Cwd == "" {
		return ""
	}
	return filepath.Join(c.Cwd, ".grok", "config.toml")
}

func (Grok) Layers(p Paths) []mcp.Layer {
	pv := grokPaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "grok-user", Label: "Grok user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	out = append(out, user)
	if path := pv.project(); path != "" {
		layer := mcp.Layer{ID: "grok-project", Label: "This folder", Path: path, Scope: "project", Writable: true}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			layer.Exists = true
		}
		out = append(out, layer)
	}
	return out
}

// List reports the servers Grok would load. Live stays empty — the driver
// never invents state it cannot see (ADR-0150 decision 4).
func (d Grok) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:grok", Installed: installed(grokBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
		Imports:           []string{},
		Found:             []mcp.HostInfo{},
	}
	seen := map[string]bool{}
	for i, layer := range rep.Layers {
		raw, err := readTOMLFile(layer.Path)
		if err != nil {
			// A file that exists but does not parse blocks only its own
			// layer (ADR-0150); the other layers still list.
			blockLayer(&rep.Layers[i], err)
			continue
		}
		for _, s := range grokServers(raw, layer) {
			if !seen[s.Name] {
				rep.Servers = append(rep.Servers, s)
				seen[s.Name] = true
			}
		}
	}
	sort.Slice(rep.Servers, func(i, j int) bool {
		return strings.ToLower(rep.Servers[i].Name) < strings.ToLower(rep.Servers[j].Name)
	})
	return rep, nil
}

// Add upserts one server by rewriting only that server's table span. It may
// create <workspace>/.grok/config.toml when asked to save there.
func (d Grok) Add(p Paths, scope, name string, entry mcp.Entry) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	if err := validEntry(entry); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := grokAddPath(grokPaths{p}, sc)
	if err != nil {
		return err
	}
	text, err := readTOMLText(path)
	if err != nil {
		return err
	}
	entries, err := parseTOMLServers(text, path)
	if err != nil {
		return err
	}
	block, err := tomlBlock("grok", name, grokEntryMap(entry, entries[name]))
	if err != nil {
		return err
	}
	return spliceTOMLTable(path, text, name, block)
}

// Toggle is refused: Grok has no per-server switch to flip.
func (Grok) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	if _, err := guestScope(scope); err != nil {
		return err
	}
	return fmt.Errorf(grokToggleRefusal)
}

// Remove deletes the table span and nothing else.
func (d Grok) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := grokWritePath(grokPaths{p}, sc)
	if err != nil {
		return err
	}
	text, err := readTOMLText(path)
	if err != nil {
		return err
	}
	entries, err := parseTOMLServers(text, path)
	if err != nil {
		return err
	}
	if _, ok := entries[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	return spliceTOMLTable(path, text, name, "")
}

// grokAddPath resolves Add targets: Add creates the workspace file on
// request, the user path is fixed.
func grokAddPath(p grokPaths, scope string) (string, error) {
	if scope == "user" {
		return p.user(), nil
	}
	if p.Cwd == "" {
		return "", fmt.Errorf("select a workspace first")
	}
	return p.project(), nil
}

// grokWritePath resolves Remove targets: they edit what exists and never
// create a config folder the way Add may.
func grokWritePath(p grokPaths, scope string) (string, error) {
	if scope == "user" {
		return p.user(), nil
	}
	path := p.project()
	if path == "" {
		return "", fmt.Errorf("select a workspace that has a .grok folder first")
	}
	if st, err := os.Stat(filepath.Dir(path)); err != nil || !st.IsDir() {
		return "", fmt.Errorf("select a workspace that has a .grok folder first")
	}
	return path, nil
}

func grokServers(raw map[string]any, layer mcp.Layer) []mcp.Server {
	servers, _ := raw["mcp_servers"].(map[string]any)
	rows := make([]mcp.Server, 0, len(servers))
	for name, e := range servers {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope, Owned: layer.Writable,
			Command: anyStr(entry["command"]),
			URL:     anyStr(entry["url"]),
			Args:    stringsOf(entry["args"]),
			Env:     redact(stringMap(entry["env"])),
			Headers: redact(stringMap(entry["headers"])),
		}
		if s.URL != "" {
			s.Transport = "url"
		} else {
			s.Transport = "stdio"
		}
		rows = append(rows, s)
	}
	return rows
}

// grokEntryMap merges the new definition into the previous entry: unknown
// keys (startup_timeout_sec, tool_timeout_sec, …) survive an update;
// switching transport drops the other side's fields (same merge rule as
// internal/mcp).
func grokEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range prev {
		out[k] = v
	}
	url := strings.TrimSpace(e.URL)
	cmd := strings.TrimSpace(e.Command)
	if url == "" && cmd == "" {
		return out
	}
	if url != "" {
		delete(out, "command")
		delete(out, "args")
		delete(out, "env")
		out["url"] = url
		if hs := headersOf(e); len(hs) > 0 {
			out["headers"] = toAnyMap(hs)
		} else {
			delete(out, "headers")
		}
		return out
	}
	delete(out, "url")
	delete(out, "headers")
	out["command"] = cmd
	if len(e.Args) > 0 {
		out["args"] = toAnySlice(e.Args)
	} else {
		delete(out, "args")
	}
	if len(e.Env) > 0 {
		out["env"] = toAnyMap(e.Env)
	} else {
		delete(out, "env")
	}
	return out
}
