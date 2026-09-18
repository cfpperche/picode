package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// Omp manages Omp's (oh-my-pi) MCP servers in its native JSON configs: the
// user file (~/.omp/agent/mcp.json) and the workspace file
// (<workspace>/.omp/mcp.json). Both are plain files, so — unlike Claude
// Code's user store — PiCode edits them directly and never shells out.
//
// The document is {"$schema"?, "mcpServers": {name: entry},
// "disabledServers"?, "enabledServers"?}; every key outside mcpServers and
// every unknown entry field (timeout, auth, oauth, …) survives a write.
// A server turns on and off through its own `enabled` field (driver toggle:
// "entry"), and ${VAR} placeholders or !command indirections are stored
// verbatim — the vendor interprets them, never PiCode.
type Omp struct{}

const ompBin = "omp"

func (Omp) ID() string { return "omp" }

func (Omp) Bin() string { return ompBin }

// AuthHint: Omp's OAuth is TUI-only — there is no headless sign-in command,
// so the copyable piece is a command typed inside the Omp TUI.
func (Omp) AuthHint(name string) AuthHint {
	return AuthHint{
		Text:    "run /mcp reauth " + name + " in the Omp TUI",
		Command: "/mcp reauth " + name,
	}
}

type ompPaths struct{ Paths }

func (c ompPaths) user() string {
	return filepath.Join(c.home(), ".omp", "agent", "mcp.json")
}

func (c ompPaths) project() string {
	if c.Cwd == "" {
		return ""
	}
	return filepath.Join(c.Cwd, ".omp", "mcp.json")
}

func (Omp) Layers(p Paths) []mcp.Layer {
	pv := ompPaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "omp-user", Label: "Omp user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	out = append(out, user)
	if path := pv.project(); path != "" {
		layer := mcp.Layer{ID: "omp-project", Label: "This folder", Path: path, Scope: "project", Writable: true}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			layer.Exists = true
		}
		out = append(out, layer)
	}
	return out
}

// List reports the servers Omp would load. Live stays empty — the driver
// never invents state it cannot see (ADR-0150 decision 4).
func (d Omp) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:omp", Installed: installed(ompBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
		Imports:           []string{},
		Found:             []mcp.HostInfo{},
	}
	seen := map[string]bool{}
	for i, layer := range rep.Layers {
		raw, err := readJSONFile(layer.Path)
		if err != nil {
			// A file that exists but does not parse blocks only its own
			// layer: the pane shows the file and a Retry/Open pair, and
			// the other layers still list (ADR-0150).
			blockLayer(&rep.Layers[i], err)
			continue
		}
		for _, s := range ompServers(raw, layer) {
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

func ompServers(raw map[string]any, layer mcp.Layer) []mcp.Server {
	servers, _ := raw["mcpServers"].(map[string]any)
	rows := make([]mcp.Server, 0, len(servers))
	for name, e := range servers {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope, Owned: layer.Writable,
			Command: strOf(entry["command"]),
			URL:     strOf(entry["url"]),
			Auth:    strOf(entry["auth"]),
			Args:    stringsOf(entry["args"]),
			Env:     redact(stringMap(entry["env"])),
			Headers: redact(stringMap(entry["headers"])),
		}
		if enabled, ok := entry["enabled"].(bool); ok {
			s.Disabled = !enabled
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

// Add upserts one server and may create the config file on request.
func (d Omp) Add(p Paths, scope, name string, entry mcp.Entry) error {
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
	path, err := ompWritePath(ompPaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFileOrEmpty(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	prev, _ := servers[name].(map[string]any)
	servers[name] = ompEntryMap(entry, prev)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// Toggle flips the entry's `enabled` field — Omp has a real per-server
// switch (driver toggle: "entry").
func (d Omp) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := ompWritePath(ompPaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	prev, ok := servers[name].(map[string]any)
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	merged := ompEntryMap(mcp.Entry{}, prev)
	merged["enabled"] = !disabled
	servers[name] = merged
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// Remove deletes the entry and nothing else.
func (d Omp) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := ompWritePath(ompPaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	delete(servers, name)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// ompWritePath resolves Toggle/Remove targets: they edit what exists and
// never create a workspace file the way Add may.
func ompWritePath(p ompPaths, scope string) (string, error) {
	if scope == "user" {
		return p.user(), nil
	}
	if p.Cwd == "" {
		return "", fmt.Errorf("select a workspace first")
	}
	return p.project(), nil
}

// ompEntryMap merges the new definition into the previous entry: unknown
// keys and Omp's own fields (timeout, auth, oauth, …) survive an update;
// switching transport drops the other side's fields (same merge rule as
// internal/mcp). An empty entry (Toggle's shape) is a pure passthrough — it
// must not clear the definition.
func ompEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
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
		delete(out, "cwd")
		out["type"] = "http"
		out["url"] = url
		if hs := headersOf(e); len(hs) > 0 {
			out["headers"] = toAnyMap(hs)
		} else {
			delete(out, "headers")
		}
		return out
	}
	delete(out, "type")
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
	if cwd := strings.TrimSpace(e.Cwd); cwd != "" {
		out["cwd"] = cwd
	}
	return out
}
