package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// Codex manages Codex CLI's MCP servers in its TOML config file
// (~/.codex/config.toml). It is a plain file, so PiCode edits it directly
// and never shells out; there is no per-workspace variant — measured
// against codex-cli 0.155.0 (2026-09-18), `codex mcp list` reads only
// CODEX_HOME's config.toml and a <workspace>/.codex/config.toml is never
// loaded — so project scope refuses with a pointer to the one file.
//
// Writes share the surgical TOML splice of the package (toml.go, ADR-0150):
// only the byte span of the `[mcp_servers.<name>]` table — header to the
// next column-0 `[` or EOF — is replaced or deleted; every other byte
// (comments, unrelated tables, ${VAR} placeholders) survives untouched.
type Codex struct{}

const codexBin = "codex"

func (Codex) ID() string { return "codex" }

func (Codex) Bin() string { return codexBin }

// AuthHint is Codex's own sign-in command for one server.
func (Codex) AuthHint(name string) AuthHint {
	cmd := codexBin + " mcp login " + name
	return AuthHint{Text: "sign in from a terminal instead: " + cmd, Command: cmd}
}

type codexPaths struct{ Paths }

func (c codexPaths) user() string {
	return filepath.Join(c.home(), ".codex", "config.toml")
}

// scopePath resolves the one file or refuses: Codex keeps a single config
// file — there is no per-workspace file the CLI would load.
func (c codexPaths) scopePath(scope string) (string, error) {
	switch scope {
	case "user", "":
		return c.user(), nil
	default:
		return "", fmt.Errorf("Codex keeps one config file (%s); there is no per-workspace file", c.user())
	}
}

func (Codex) Layers(p Paths) []mcp.Layer {
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "codex-user", Label: "Codex user", Path: codexPaths{p}.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	return append(out, user)
}

// List reports the servers Codex would load. Live stays empty: Codex exposes
// no headless status signal, so the honest ceiling is "configured".
func (d Codex) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:codex", Installed: installed(codexBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
	}
	seen := map[string]bool{}
	for _, layer := range rep.Layers {
		raw, err := readTOMLFile(layer.Path)
		if err != nil {
			return rep, err
		}
		for _, s := range codexServers(raw, layer) {
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

// Add upserts one server by rewriting only that server's table span in the
// one config file Codex loads.
func (d Codex) Add(p Paths, scope, name string, entry mcp.Entry) error {
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
	path, err := codexPaths{p}.scopePath(sc)
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
	block, err := tomlBlock("codex", name, codexEntryMap(entry, entries[name]))
	if err != nil {
		return err
	}
	return spliceTOMLTable(path, text, name, block)
}

// Toggle flips the table's `enabled` flag — Codex has a real per-server
// switch, so the pane keeps it (driver toggle: "entry").
func (d Codex) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := codexPaths{p}.scopePath(sc)
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
	prev, ok := entries[name]
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	merged := codexEntryMap(mcp.Entry{}, prev)
	merged["enabled"] = !disabled
	block, err := tomlBlock("codex", name, merged)
	if err != nil {
		return err
	}
	return spliceTOMLTable(path, text, name, block)
}

// Remove deletes the table span and nothing else.
func (d Codex) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := codexPaths{p}.scopePath(sc)
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

func codexServers(raw map[string]any, layer mcp.Layer) []mcp.Server {
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
			Headers: redact(stringMap(entry["http_headers"])),
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

func anyStr(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

// codexEntryMap merges the new definition into the previous entry: unknown
// keys and the enabled flag survive an update; switching transport drops the
// other side's fields (same merge rule as internal/mcp). An empty entry
// (Toggle's shape) is a pure passthrough — it must not clear the definition.
func codexEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
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
			out["http_headers"] = toAnyMap(hs)
		} else {
			delete(out, "http_headers")
		}
		return out
	}
	delete(out, "url")
	delete(out, "http_headers")
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
