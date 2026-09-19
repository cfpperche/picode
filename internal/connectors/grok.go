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
// Toggle mirrors what grok mcp disable/enable writes (measured against grok
// 1.0.34, 2026-09-18): the entry's `enabled` flag plus — in the user file
// only — the top-level `disabled_mcp_servers` array (the vendor's personal
// overlay; it never rewrites project configs). Remove strips the name from
// the array of the same file and nothing else.
type Grok struct{}

const grokBin = "grok"

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
	block, err := tomlBlock("grok", name, grokEntryMap(entry, entries[name], entries[name] == nil))
	if err != nil {
		return err
	}
	return spliceTOMLTable(path, text, name, block)
}

// Toggle mirrors the vendor's own enable/disable writes (driver toggle:
// "entry"): the table's `enabled` flag everywhere, plus the user file's
// top-level `disabled_mcp_servers` overlay for user scope — the mechanism
// `grok mcp disable` uses when the entry lives in a project config.
func (d Grok) Toggle(p Paths, scope, name string, disabled bool) error {
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
	prev, ok := entries[name]
	if !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	merged := grokEntryMap(mcp.Entry{}, prev, false)
	merged["enabled"] = !disabled
	block, err := tomlBlock("grok", name, merged)
	if err != nil {
		return err
	}
	if err := spliceTOMLTable(path, text, name, block); err != nil {
		return err
	}
	if sc != "user" {
		return nil
	}
	// The overlay array rides the same file; mirror the vendor: disable
	// appends the name, enable clears it, an empty array removes the key.
	after, err := readTOMLText(path)
	if err != nil {
		return err
	}
	names := tomlStringArray(after, "disabled_mcp_servers")
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	if disabled {
		set[name] = true
	} else {
		delete(set, name)
	}
	return rewriteTOMLStringArray(path, after, "disabled_mcp_servers", set)
}

// Remove deletes the table span; in the user file it also drops the name
// from the `disabled_mcp_servers` overlay — what `grok mcp remove` leaves
// behind (the vendor cleans only the file it edits, never the other scope).
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
	if err := spliceTOMLTable(path, text, name, ""); err != nil {
		return err
	}
	after, err := readTOMLText(path)
	if err != nil {
		return err
	}
	set := map[string]bool{}
	for _, n := range tomlStringArray(after, "disabled_mcp_servers") {
		if n != name {
			set[n] = true
		}
	}
	return rewriteTOMLStringArray(path, after, "disabled_mcp_servers", set)
}

// tomlStringArray reads a top-level string array key ("disabled_mcp_servers")
// from config text via a full parse; a missing key or a non-array value
// yields nil. Values that are not strings are skipped.
func tomlStringArray(text, key string) []string {
	var raw map[string]any
	if err := toml.Unmarshal([]byte(text), &raw); err != nil {
		return nil
	}
	arr, ok := raw[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// rewriteTOMLStringArray replaces the byte span of a top-level `key = [...]`
// with the sorted names (empty set removes the key, matching what the
// vendor leaves behind), inserting the key at the top of the file when it
// does not exist yet and the set is non-empty — the position and shape the
// vendor's own disable writes. The result is validated as TOML and written
// atomically; nothing is touched when there is nothing to do.
func rewriteTOMLStringArray(path, text, key string, set map[string]bool) error {
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	quoted := make([]string, len(names))
	for i, n := range names {
		quoted[i] = `"` + strings.ReplaceAll(n, `"`, `\"`) + `"`
	}
	rendered := key + " = [" + strings.Join(quoted, ", ") + "]\n"

	start, end := tomlTopLevelKeySpan(text, key)
	var out string
	switch {
	case start < 0 && len(names) == 0:
		return nil
	case start < 0:
		out = rendered + text
	case len(names) == 0:
		out = text[:start] + text[end:]
	default:
		out = text[:start] + rendered + text[end:]
	}
	var check map[string]any
	if err := toml.Unmarshal([]byte(out), &check); err != nil {
		return fmt.Errorf("the edit would make %s invalid TOML: %v", path, err)
	}
	return atomicWrite(path, []byte(out))
}

// tomlTopLevelKeySpan finds `key = …` at column 0 before the first table
// header and returns the span of its logical line (brackets may span lines).
// start < 0 when the key is absent or never appears at top level.
func tomlTopLevelKeySpan(text, key string) (start, end int) {
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
	prefix := key + " ="
	for i, ln := range lines {
		t := strings.TrimSpace(texts[i])
		if strings.HasPrefix(t, "[") {
			break // a table opened; the top-level region ended
		}
		if !strings.HasPrefix(t, prefix) {
			continue
		}
		depth := strings.Count(t, "[") - strings.Count(t, "]")
		for j := i + 1; j < len(lines) && depth > 0; j++ {
			depth += strings.Count(texts[j], "[") - strings.Count(texts[j], "]")
			return ln.start, min(lines[j].end+1, len(text))
		}
		return ln.start, min(ln.end+1, len(text))
	}
	return -1, -1
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

// grokEntryMap merges the new definition into the previous entry: unknown
// keys (startup_timeout_sec, tool_timeout_sec, …) and the previous enabled
// flag survive an update — only Toggle flips it. A brand-new entry starts
// `enabled = true`, the shape the vendor's own add writes; a missing flag
// means enabled too (documented default), so updates never add one back.
// Switching transport drops the other side's fields (same merge rule as
// internal/mcp).
func grokEntryMap(e mcp.Entry, prev map[string]any, isNew bool) map[string]any {
	out := map[string]any{}
	for k, v := range prev {
		out[k] = v
	}
	url := strings.TrimSpace(e.URL)
	cmd := strings.TrimSpace(e.Command)
	if url == "" && cmd == "" {
		return out
	}
	if isNew {
		out["enabled"] = true
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
