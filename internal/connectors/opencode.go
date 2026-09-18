package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// OpenCode manages OpenCode's MCP servers in its native JSON configs: the
// user file ($XDG_CONFIG_HOME/opencode/opencode.json, default
// ~/.config/opencode/opencode.json) and the workspace file
// (<workspace>/opencode.json). Both are plain files, so PiCode edits them
// directly and never shells out.
//
// The document's `mcp` block holds {name: {type: "local"|"remote",
// command: […]|url, enabled?, environment?, headers?, …}}; every other key
// of the config ($schema, theme, provider, …) and every unknown entry field
// (oauth, timeout, …) survives a write, and the entry's `enabled` field is
// the toggle (driver toggle: "entry"). This driver manages only these two
// config files — the per-launch session config PiCode generates through
// OPENCODE_CONFIG is a different document and is never touched.
type OpenCode struct{}

const opencodeBin = "opencode"

func (OpenCode) ID() string { return "opencode" }

func (OpenCode) Bin() string { return opencodeBin }

// AuthHint is OpenCode's own sign-in command for one server.
func (OpenCode) AuthHint(name string) AuthHint {
	cmd := opencodeBin + " mcp auth " + name
	return AuthHint{Text: "sign in from a terminal instead: " + cmd, Command: cmd}
}

type opencodePaths struct{ Paths }

// user honors XDG_CONFIG_HOME: OpenCode resolves its config through the XDG
// base-directory rule, so an override must also move the file PiCode edits.
func (c opencodePaths) user() string {
	if x := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); x != "" {
		return filepath.Join(x, "opencode", "opencode.json")
	}
	return filepath.Join(c.home(), ".config", "opencode", "opencode.json")
}

// project is the workspace's own opencode.json (no dot-folder), listed
// whenever a workspace is selected; Add creates it on request.
func (c opencodePaths) project() string {
	if c.Cwd == "" {
		return ""
	}
	return filepath.Join(c.Cwd, "opencode.json")
}

func (OpenCode) Layers(p Paths) []mcp.Layer {
	pv := opencodePaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "opencode-user", Label: "OpenCode user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	out = append(out, user)
	if path := pv.project(); path != "" {
		layer := mcp.Layer{ID: "opencode-project", Label: "This folder", Path: path, Scope: "project", Writable: true}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			layer.Exists = true
		}
		out = append(out, layer)
	}
	return out
}

// List reports the servers OpenCode would load. Live stays empty — the
// driver never invents state it cannot see (ADR-0150 decision 4).
func (d OpenCode) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:opencode", Installed: installed(opencodeBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
		Imports:           []string{},
		Found:             []mcp.HostInfo{},
	}
	seen := map[string]bool{}
	for _, layer := range rep.Layers {
		raw, err := readJSONFile(layer.Path)
		if err != nil {
			return rep, err
		}
		for _, s := range opencodeServers(raw, layer) {
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

func opencodeServers(raw map[string]any, layer mcp.Layer) []mcp.Server {
	servers, _ := raw["mcp"].(map[string]any)
	rows := make([]mcp.Server, 0, len(servers))
	for name, e := range servers {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope, Owned: layer.Writable,
			URL:     strOf(entry["url"]),
			Env:     redact(stringMap(entry["environment"])),
			Headers: redact(stringMap(entry["headers"])),
		}
		// The command is an array whose first element is the binary.
		if cmd := stringsOf(entry["command"]); len(cmd) > 0 {
			s.Command = cmd[0]
			s.Args = cmd[1:]
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

// Add upserts one server and may create the workspace file on request.
func (d OpenCode) Add(p Paths, scope, name string, entry mcp.Entry) error {
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
	path, err := opencodeWritePath(opencodePaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFileOrEmpty(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcp"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	prev, _ := servers[name].(map[string]any)
	servers[name] = opencodeEntryMap(entry, prev)
	raw["mcp"] = servers
	return writeJSONFile(path, raw)
}

// Toggle flips the entry's `enabled` field — OpenCode has a real per-server
// switch (driver toggle: "entry").
func (d OpenCode) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := opencodeWritePath(opencodePaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcp"].(map[string]any)
	prev, ok := servers[name].(map[string]any)
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	merged := opencodeEntryMap(mcp.Entry{}, prev)
	merged["enabled"] = !disabled
	servers[name] = merged
	raw["mcp"] = servers
	return writeJSONFile(path, raw)
}

// Remove deletes the entry and nothing else.
func (d OpenCode) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := opencodeWritePath(opencodePaths{p}, sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcp"].(map[string]any)
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	delete(servers, name)
	raw["mcp"] = servers
	return writeJSONFile(path, raw)
}

// opencodeWritePath resolves Toggle/Remove targets: they edit what exists
// and never create a workspace file the way Add may.
func opencodeWritePath(p opencodePaths, scope string) (string, error) {
	if scope == "user" {
		return p.user(), nil
	}
	if p.Cwd == "" {
		return "", fmt.Errorf("select a workspace first")
	}
	return p.project(), nil
}

// opencodeEntryMap merges the new definition into the previous entry:
// unknown keys and OpenCode's own fields (oauth, timeout, …) survive an
// update; the previous `enabled` flag survives an update too — only Toggle
// flips it. Switching transport drops the other side's fields (same merge
// rule as internal/mcp). An empty entry (Toggle's shape) is a pure
// passthrough — it must not clear the definition.
func opencodeEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
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
		delete(out, "environment")
		out["type"] = "remote"
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
	out["type"] = "local"
	command := append([]string{cmd}, e.Args...)
	out["command"] = toAnySlice(command)
	if len(e.Env) > 0 {
		out["environment"] = toAnyMap(e.Env)
	} else {
		delete(out, "environment")
	}
	return out
}
