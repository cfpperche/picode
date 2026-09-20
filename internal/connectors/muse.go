package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// Muse manages Muse Code's MCP servers in its native settings file: the
// single JSON document at ~/.config/muse/settings.json. It is a plain file,
// so PiCode edits it directly and never shells out; there is no
// per-workspace variant, so project scope refuses with a pointer to the one
// file.
//
// The document's `mcp_servers` block holds {name: {transport:
// "stdio"|"streamable_http", command, args, env | url, headers, enabled?,
// mode?, …}}. The entry's `enabled` field is the toggle (driver toggle:
// "entry"), `schema_version` and every unknown key of the document and of
// the entry (`mode`, vendor fields PiCode does not model) survive every
// write through merge-by-key, and the whole document is re-marshalled with
// 2-space indent (same codec as the other JSON drivers).
type Muse struct{}

const museBin = "muse"

func (Muse) ID() string { return "muse" }

func (Muse) Bin() string { return museBin }

// AuthHint is Muse's own sign-in command for one server (the muse CLI
// dispatches `mcp` and `login`, so `muse mcp login <name>` is its path).
func (Muse) AuthHint(name string) AuthHint {
	cmd := museBin + " mcp login " + name
	return AuthHint{Text: "sign in from a terminal instead: " + cmd, Command: cmd}
}

type musePaths struct{ Paths }

// user is the one Muse settings file. Muse resolves it under the home
// directory (measured: PiCode's own muse hooks landed in
// ~/.config/muse/settings.json); no XDG branch is honored.
func (c musePaths) user() string {
	return filepath.Join(c.home(), ".config", "muse", "settings.json")
}

// scopePath resolves the one file or refuses: Muse keeps a single settings
// file — there is no per-workspace variant to create.
func (c musePaths) scopePath(scope string) (string, error) {
	switch scope {
	case "user", "":
		return c.user(), nil
	default:
		return "", fmt.Errorf("Muse keeps one settings file (%s); there is no per-workspace file", c.user())
	}
}

func (Muse) Layers(p Paths) []mcp.Layer {
	pv := musePaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "muse-user", Label: "Muse user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	return append(out, user)
}

// List reports the servers Muse would load. Live stays empty — the driver
// never invents state it cannot see (ADR-0150 decision 4).
func (d Muse) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:muse", Installed: installed(museBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
	}
	raw, err := readJSONFile(rep.Layers[0].Path)
	if err != nil {
		// The one file exists but does not parse: the layer reports the
		// reason and the pane shows a blocked line for it (ADR-0150) —
		// the report still answers 200.
		blockLayer(&rep.Layers[0], err)
		raw = nil
	}
	servers, _ := raw["mcp_servers"].(map[string]any)
	for name, e := range servers {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: rep.Layers[0].ID, Path: rep.Layers[0].Path, Scope: "user", Owned: true,
			Command: strOf(entry["command"]),
			URL:     strOf(entry["url"]),
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
		rep.Servers = append(rep.Servers, s)
	}
	sort.Slice(rep.Servers, func(i, j int) bool {
		return strings.ToLower(rep.Servers[i].Name) < strings.ToLower(rep.Servers[j].Name)
	})
	return rep, nil
}

// Add upserts one server in the settings file, creating it on request.
func (d Muse) Add(p Paths, scope, name string, entry mcp.Entry) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	if err := validEntry(entry); err != nil {
		return err
	}
	sc, err := cliScope(scope)
	if err != nil {
		return err
	}
	path, err := musePaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFileOrEmpty(path)
	if err != nil {
		return err
	}
	ensureMuseSchema(raw)
	servers, _ := raw["mcp_servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	prev, _ := servers[name].(map[string]any)
	servers[name] = museEntryMap(entry, prev)
	raw["mcp_servers"] = servers
	return writeJSONFile(path, raw)
}

// ensureMuseSchema keeps the settings file valid for Muse's own loader:
// measured against Muse Code 1.3.0 (2026-09-18), a settings.json without
// schema_version is refused as malformed ("missing field `schema_version`"),
// so a file PiCode creates must carry it. An existing value survives —
// merge-by-key, never overwritten.
func ensureMuseSchema(raw map[string]any) {
	if _, ok := raw["schema_version"]; !ok {
		raw["schema_version"] = 1
	}
}

// Toggle flips the entry's `enabled` field — Muse has a real per-server
// switch (driver toggle: "entry").
func (d Muse) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := cliScope(scope)
	if err != nil {
		return err
	}
	path, err := musePaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcp_servers"].(map[string]any)
	prev, ok := servers[name].(map[string]any)
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	ensureMuseSchema(raw)
	merged := museEntryMap(mcp.Entry{}, prev)
	merged["enabled"] = !disabled
	servers[name] = merged
	raw["mcp_servers"] = servers
	return writeJSONFile(path, raw)
}

// Remove deletes the entry and nothing else.
func (d Muse) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := cliScope(scope)
	if err != nil {
		return err
	}
	path, err := musePaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	servers, _ := raw["mcp_servers"].(map[string]any)
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	ensureMuseSchema(raw)
	delete(servers, name)
	raw["mcp_servers"] = servers
	return writeJSONFile(path, raw)
}

// museEntryMap merges the new definition into the previous entry: unknown
// keys and Muse's own fields (`mode`, …) survive an update; the previous
// `enabled` flag survives an update too — only Toggle flips it. Switching
// transport drops the other side's fields (same merge rule as
// internal/mcp). An empty entry (Toggle's shape) is a pure passthrough — it
// must not clear the definition.
func museEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
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
		out["transport"] = "streamable_http"
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
	out["transport"] = "stdio"
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
