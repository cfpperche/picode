package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// AGY manages Antigravity's MCP servers in its native JSON config: the
// single file at ~/.gemini/config/mcp_config.json. It is a plain file,
// edited directly — never through the vendor CLI. There is no per-workspace
// variant: measured against agy 1.2.6 (2026-09-18), `agy mcp list/add`
// resolve only the user file and never load a workspace config, so project
// scope refuses with a pointer to the one file.
//
// Remote servers ride the `serverUrl` field. An entry that keeps its URL
// under the legacy `url` key still displays, but it is reported as not
// owned and Toggle/Remove refuse: Antigravity owns that shape. The on/off
// switch is the entry's `disabled` field (driver toggle: "entry"), and
// Antigravity's own fields (authProviderType, oauth, disabledTools, …)
// survive every write untouched.
type AGY struct{}

const agyBin = "agy"

func (AGY) ID() string { return "agy" }

func (AGY) Bin() string { return agyBin }

// AuthHint: Antigravity signs in inside its own settings, not a terminal —
// the hint is plain text with nothing to copy.
func (AGY) AuthHint(name string) AuthHint {
	return AuthHint{Text: "authenticate in Antigravity (Agent Settings → Authenticate)"}
}

type agyPaths struct{ Paths }

func (c agyPaths) user() string {
	return filepath.Join(c.home(), ".gemini", "config", "mcp_config.json")
}

// scopePath resolves the one file or refuses: Antigravity keeps a single
// config file — there is no per-workspace file the CLI would load.
func (c agyPaths) scopePath(scope string) (string, error) {
	switch scope {
	case "user", "":
		return c.user(), nil
	default:
		return "", fmt.Errorf("Antigravity keeps one config file (%s); there is no per-workspace file", c.user())
	}
}

func (AGY) Layers(p Paths) []mcp.Layer {
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "agy-user", Label: "Antigravity user", Path: agyPaths{p}.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	return append(out, user)
}

// List reports the servers Antigravity would load. Live stays empty — the
// driver never invents state it cannot see (ADR-0150 decision 4).
func (d AGY) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:agy", Installed: installed(agyBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
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
		for _, s := range agyServers(raw, layer) {
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

func agyServers(raw map[string]any, layer mcp.Layer) []mcp.Server {
	servers, _ := raw["mcpServers"].(map[string]any)
	rows := make([]mcp.Server, 0, len(servers))
	for name, e := range servers {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope,
			Command: strOf(entry["command"]),
			URL:     agyURLOf(entry),
			Args:    stringsOf(entry["args"]),
			Env:     redact(stringMap(entry["env"])),
			Headers: redact(stringMap(entry["headers"])),
		}
		// A legacy url-only entry displays but is not PiCode's to manage.
		s.Owned = layer.Writable && !agyLegacy(entry)
		if disabled, ok := entry["disabled"].(bool); ok {
			s.Disabled = disabled
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

// agyURLOf reads the native serverUrl; a legacy url shows only when no
// native field shadows it.
func agyURLOf(entry map[string]any) string {
	if url := strOf(entry["serverUrl"]); url != "" {
		return url
	}
	return strOf(entry["url"])
}

// agyLegacy reports the url-only shape Antigravity wrote before serverUrl:
// displayed read-only, refused on write.
func agyLegacy(entry map[string]any) bool {
	_, hasURL := entry["url"]
	_, hasServerURL := entry["serverUrl"]
	return hasURL && !hasServerURL
}

// agyLegacyRefusal is the honest answer to a write against the legacy
// url-only shape.
const agyLegacyRefusal = "legacy entry managed in Antigravity — edit or remove it there"

// Add upserts one server and may create the config file on request. A
// legacy url-only entry of the same name converts to the managed shape.
func (d AGY) Add(p Paths, scope, name string, entry mcp.Entry) error {
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
	path, err := agyPaths{p}.scopePath(sc)
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
	servers[name] = agyEntryMap(entry, prev)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// Toggle flips the entry's `disabled` field — Antigravity has a real
// per-server switch (driver toggle: "entry").
func (d AGY) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := agyPaths{p}.scopePath(sc)
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
	if agyLegacy(prev) {
		return fmt.Errorf(agyLegacyRefusal)
	}
	merged := agyEntryMap(mcp.Entry{}, prev)
	merged["disabled"] = disabled
	servers[name] = merged
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// Remove deletes the entry and nothing else.
func (d AGY) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := agyPaths{p}.scopePath(sc)
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
	if agyLegacy(prev) {
		return fmt.Errorf(agyLegacyRefusal)
	}
	delete(servers, name)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// agyEntryMap merges the new definition into the previous entry:
// Antigravity's own fields (authProviderType, oauth, disabledTools, …)
// survive an update; switching transport drops the other side's fields. A
// managed write never leaves a legacy `url` behind — it becomes serverUrl.
// An empty entry (Toggle's shape) is a pure passthrough plus the flag.
func agyEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
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
		delete(out, "url")
		out["serverUrl"] = url
		if hs := headersOf(e); len(hs) > 0 {
			out["headers"] = toAnyMap(hs)
		} else {
			delete(out, "headers")
		}
		return out
	}
	delete(out, "serverUrl")
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
