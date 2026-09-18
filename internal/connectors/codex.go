package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/cfpperche/picode/internal/mcp"
)

// Codex manages Codex CLI's MCP servers in TOML config files: the user file
// (~/.codex/config.toml) and the workspace file (<workspace>/.codex/config.toml).
//
// Writes are surgical by contract (ADR-0150): only the byte span of the
// `[mcp_servers.<name>]` table — header to the next column-0 `[` or EOF — is
// replaced or deleted; every other byte (comments, unrelated tables, ${VAR}
// placeholders) survives untouched. The spliced text is re-parsed before it
// is written and the write is atomic, so corruption fails loudly instead of
// landing on disk.
type Codex struct{}

const codexBin = "codex"

func (Codex) ID() string { return "codex" }

func (Codex) Bin() string { return codexBin }

type codexPaths struct{ Paths }

func (c codexPaths) user() string {
	return filepath.Join(c.home(), ".codex", "config.toml")
}

// project is listed only when the workspace already keeps a .codex folder;
// Add creates it on request, reading never invents it.
func (c codexPaths) project() string {
	if c.Cwd == "" {
		return ""
	}
	dir := filepath.Join(c.Cwd, ".codex")
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return ""
	}
	return filepath.Join(dir, "config.toml")
}

func (Codex) Layers(p Paths) []mcp.Layer {
	pv := codexPaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "codex-user", Label: "Codex user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	out = append(out, user)
	if path := pv.project(); path != "" {
		layer := mcp.Layer{ID: "codex-project", Label: "This folder", Path: path, Scope: "project", Writable: true}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			layer.Exists = true
		}
		out = append(out, layer)
	}
	return out
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
		Imports:           []string{},
		Found:             []mcp.HostInfo{},
	}
	seen := map[string]bool{}
	for _, layer := range rep.Layers {
		raw, err := readCodexFile(layer.Path)
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

// Add upserts one server by rewriting only that server's table span. It may
// create <workspace>/.codex/config.toml when asked to save there.
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
	pv := codexPaths{p}
	path := pv.user()
	if sc == "project" {
		if p.Cwd == "" {
			return fmt.Errorf("select a workspace first")
		}
		path = filepath.Join(p.Cwd, ".codex", "config.toml")
	}
	text, err := readCodexText(path)
	if err != nil {
		return err
	}
	entries, err := parseCodex(text, path)
	if err != nil {
		return err
	}
	block, err := codexBlock(name, codexEntryMap(entry, entries[name]))
	if err != nil {
		return err
	}
	return spliceCodex(path, text, name, block)
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
	path, err := codexWritePath(codexPaths{p}, sc)
	if err != nil {
		return err
	}
	text, err := readCodexText(path)
	if err != nil {
		return err
	}
	entries, err := parseCodex(text, path)
	if err != nil {
		return err
	}
	prev, ok := entries[name]
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	merged := codexEntryMap(mcp.Entry{}, prev)
	merged["enabled"] = !disabled
	block, err := codexBlock(name, merged)
	if err != nil {
		return err
	}
	return spliceCodex(path, text, name, block)
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
	path, err := codexWritePath(codexPaths{p}, sc)
	if err != nil {
		return err
	}
	text, err := readCodexText(path)
	if err != nil {
		return err
	}
	entries, err := parseCodex(text, path)
	if err != nil {
		return err
	}
	if _, ok := entries[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	return spliceCodex(path, text, name, "")
}

// codexWritePath resolves Toggle/Remove targets: they edit what exists and
// never create a config folder the way Add may.
func codexWritePath(p codexPaths, scope string) (string, error) {
	if scope == "user" {
		return p.user(), nil
	}
	path := p.project()
	if path == "" {
		return "", fmt.Errorf("select a workspace that has a .codex folder first")
	}
	return path, nil
}

// codexBlock renders one `[mcp_servers.<name>]` table from the merged entry.
// The entry is marshaled inside a wrapper document so nested maps (env,
// unknown tables) come out as proper [mcp_servers.<name>.…] sub-tables —
// marshaling the bare entry would emit them at the root level — then the
// wrapper's preamble is cut and everything from this table's header on is
// kept.
func codexBlock(name string, entry map[string]any) (string, error) {
	wrapper := map[string]any{"mcp_servers": map[string]any{name: entry}}
	b, err := toml.Marshal(wrapper)
	if err != nil {
		return "", fmt.Errorf("cannot encode %s for the codex config: %w", name, err)
	}
	lines := strings.SplitAfter(string(b), "\n")
	for i, line := range lines {
		if codexHeader(line, name) {
			return strings.Join(lines[i:], ""), nil
		}
	}
	return "", fmt.Errorf("cannot encode %s for the codex config", name)
}

// codexHeader reports whether a header line opens this server's table
// (bare, double-quoted or literal-quoted key form).
func codexHeader(line, name string) bool {
	switch strings.TrimRight(line, " \t\r\n") {
	case "[mcp_servers." + name + "]",
		`[mcp_servers."` + name + `"]`,
		"[mcp_servers.'" + name + "']":
		return true
	}
	return false
}

// spliceCodex replaces (block == "") or rewrites the table's byte span,
// validates the whole result, and lands it atomically.
func spliceCodex(path, text, name, block string) error {
	start, end, ok := tomlTableSpan(text, name)
	var out string
	switch {
	case !ok && block == "":
		return fmt.Errorf("server %q is not in %s", name, path)
	case !ok:
		appended := ensureTrailingNewline(text)
		// Separate the new table from whatever the file ended with, without
		// stacking blank lines.
		if appended != "" && !strings.HasSuffix(appended, "\n\n") {
			appended += "\n"
		}
		out = appended + block
	case block == "":
		out = text[:start] + text[end:]
	default:
		// The blank line that separated the old table from the next one sat
		// inside the span; keep one before the following table.
		replacement := block
		if strings.HasPrefix(text[end:], "[") && !strings.HasSuffix(replacement, "\n\n") {
			replacement += "\n"
		}
		out = text[:start] + replacement + text[end:]
	}
	var check map[string]any
	if err := toml.Unmarshal([]byte(out), &check); err != nil {
		return fmt.Errorf("the edit would make %s invalid TOML: %v", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return atomicWrite(path, []byte(out))
}

// tomlTableSpan returns the byte span of [mcp_servers.<name>]: from its
// header line to the next line that opens a table outside this entry, or
// EOF. The entry's own sub-tables ([mcp_servers.<name>.…]) sit inside the
// span and are rewritten with it; every other table — including a bare
// [mcp_servers] parent — starts with "[" at column 0 and ends it. Bare,
// double-quoted and literal-quoted key forms all match.
func tomlTableSpan(text, name string) (start, end int, ok bool) {
	type linePos struct{ start, end int }
	var lines []linePos
	texts := []string{}
	pos := 0
	for pos < len(text) {
		nl := strings.IndexByte(text[pos:], '\n')
		lineEnd, next := len(text), len(text)+1
		if nl >= 0 {
			lineEnd, next = pos+nl, pos+nl+1
		}
		lines = append(lines, linePos{start: pos, end: lineEnd})
		texts = append(texts, strings.TrimRight(text[pos:lineEnd], "\r"))
		if nl < 0 {
			break
		}
		pos = next
	}
	subPrefix := []string{
		"[mcp_servers." + name + ".",
		`[mcp_servers."` + name + `".`,
		"[mcp_servers.'" + name + "'.",
	}
	for i, ln := range lines {
		if !codexHeader(texts[i], name) {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			if !strings.HasPrefix(texts[j], "[") {
				continue
			}
			sub := false
			for _, p := range subPrefix {
				if strings.HasPrefix(texts[j], p) {
					sub = true
					break
				}
			}
			if sub {
				continue
			}
			return ln.start, lines[j].start, true
		}
		return ln.start, len(text), true
	}
	return 0, 0, false
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// readCodexFile parses a config into a generic map. A missing file is an
// empty config; malformed TOML is an error — refuse loudly, never write on
// top of a file we could not read (ADR-0150).
func readCodexFile(path string) (map[string]any, error) {
	text, err := readCodexText(path)
	if err != nil || strings.TrimSpace(text) == "" {
		return map[string]any{}, err
	}
	var raw map[string]any
	if err := toml.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("%s is not valid TOML", path)
	}
	return raw, nil
}

func readCodexText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// parseCodex returns the mcp_servers entries of a config text. Malformed
// TOML refuses the whole operation before any splice is attempted.
func parseCodex(text, path string) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	if strings.TrimSpace(text) == "" {
		return out, nil
	}
	var raw map[string]any
	if err := toml.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("%s is not valid TOML", path)
	}
	servers, _ := raw["mcp_servers"].(map[string]any)
	for name, e := range servers {
		if m, ok := e.(map[string]any); ok {
			out[name] = m
		}
	}
	return out, nil
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
