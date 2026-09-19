package connectors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// Claude manages Claude Code's MCP servers (ADR-0150 phase 1).
//
// Project scope is the plain `.mcp.json` next to the workspace, edited
// directly with encoding/json over map[string]any so unknown keys survive.
// User scope lives inside ~/.claude.json — a Claude Code-owned store that
// also holds unrelated state — so PiCode never reads or writes that file;
// the vendor CLI (`claude mcp add/remove/list --scope user`) is the only
// path in and out. Claude has no per-server disabled flag, so Toggle is
// refused and the pane (driver toggle: "none") renders no switch at all.
type Claude struct{}

const claudeBin = "claude"

// ToggleRefusal is the honest answer to an enable/disable ask: Claude Code
// turns connectors on and off in its own interface.
const ToggleRefusal = "Claude Code connectors turn on and off in Claude Code; remove instead"

// ID matches the CLI catalog id (internal/clilaunch) so deep links such as
// #/clis/claude-code/connectors resolve to this driver.
func (Claude) ID() string { return "claude-code" }

// Bin is the vendor binary; it is not the driver id.
func (Claude) Bin() string { return claudeBin }

// AuthHint is Claude Code's own sign-in command for one server.
func (Claude) AuthHint(name string) AuthHint {
	cmd := claudeBin + " mcp login " + name
	return AuthHint{Text: "sign in from a terminal instead: " + cmd, Command: cmd}
}

type claudePaths struct{ Paths }

func (c claudePaths) project() string {
	if c.Cwd == "" {
		return ""
	}
	return filepath.Join(c.Cwd, ".mcp.json")
}

// Layers lists Claude's config in merge order (last wins, matching Claude
// Code: the workspace file overrides the user store). The user layer has no
// path on purpose — its store is vendor-managed and never opened here.
func (Claude) Layers(p Paths) []mcp.Layer {
	out := []mcp.Layer{{
		ID: "claude-user", Label: "Claude Code user", Path: "", Scope: "user",
		Exists: installed(claudeBin), Writable: installed(claudeBin),
	}}
	if path := (claudePaths{p}).project(); path != "" {
		layer := mcp.Layer{ID: "claude-project", Label: "This folder", Path: path, Scope: "project", Writable: true}
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			layer.Exists = true
		}
		out = append(out, layer)
	}
	return out
}

// List reports every server Claude Code would load. Live stays empty — the
// driver never invents state it cannot see (ADR-0150 decision 4).
func (d Claude) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:claude", Installed: installed(claudeBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
	}
	seen := map[string]bool{}
	for i, layer := range rep.Layers {
		var rows []mcp.Server
		var err error
		if layer.Scope == "user" {
			rows, err = d.listUser(layer)
		} else {
			rows, err = listClaudeFile(layer.Path, layer)
		}
		if err != nil {
			// A project file that exists but does not parse blocks only
			// its own layer (ADR-0150); a user-store failure is a vendor
			// CLI problem and still fails the pane with its message.
			if layer.Scope == "user" {
				return rep, err
			}
			blockLayer(&rep.Layers[i], err)
			continue
		}
		for _, s := range rows {
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

// listUser reads `claude mcp list`. Newer CLIs print JSON; older ones print
// one server per line. Both are accepted, and unparseable output degrades to
// fewer rows — never an error, because a formatting change upstream must not
// break the pane.
func (Claude) listUser(layer mcp.Layer) ([]mcp.Server, error) {
	if !installed(claudeBin) {
		return nil, nil
	}
	stdout, err := runVendorCLI(claudeBin, "mcp", "list")
	if err != nil {
		return nil, err
	}
	if rows := claudeRowsFromJSON(strings.TrimSpace(stdout), layer); rows != nil {
		return rows, nil
	}
	return claudeRowsFromLines(stdout, layer), nil
}

func claudeRowsFromJSON(out string, layer mcp.Layer) []mcp.Server {
	if out == "" || out[0] != '{' && out[0] != '[' {
		return nil
	}
	var doc any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		return nil
	}
	// Three shapes are tolerated: {"mcpServers": {name: entry}},
	// [entry, …] with a name field, and {name: entry}.
	entries := map[string]map[string]any{}
	switch v := doc.(type) {
	case map[string]any:
		if inner, ok := v["mcpServers"].(map[string]any); ok {
			for name, e := range inner {
				if m, ok := e.(map[string]any); ok {
					entries[name] = m
				}
			}
			break
		}
		for name, e := range v {
			if m, ok := e.(map[string]any); ok {
				entries[name] = m
			}
		}
	case []any:
		for _, e := range v {
			m, ok := e.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["name"].(string)
			if strings.TrimSpace(name) != "" {
				entries[name] = m
			}
		}
	default:
		return nil
	}
	if len(entries) == 0 {
		return nil
	}
	rows := make([]mcp.Server, 0, len(entries))
	for name, e := range entries {
		rows = append(rows, claudeServer(name, e, layer))
	}
	return rows
}

// claudeRowsFromLines parses `name: detail` lines. The detail (command, URL,
// health suffix) is display-only — rows carry it in Command so the pane shows
// what the CLI printed and nothing more. With no servers configured the
// vendor prints a sentence instead of a list ("No MCP servers configured. …");
// any line matching that prose means the whole list is empty. A line without
// a `:` separator is banner noise, never a row. Unparseable lines are
// skipped, never an error: a formatting change upstream must not break the
// pane.
func claudeRowsFromLines(out string, layer mcp.Layer) []mcp.Server {
	var rows []mcp.Server
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "#") ||
			strings.HasSuffix(line, "...") || strings.HasSuffix(line, "…") {
			continue
		}
		if claudeEmptyList(line) {
			return nil
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			continue
		}
		name, rest := strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
		if name == "" || !claudeRowName(name) {
			continue
		}
		rest = trimHealth(rest)
		row := mcp.Server{Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope, Owned: layer.Writable}
		if rest != "" {
			row.Command = rest
			// A URL target is a remote server: report it like the file
			// codec does, so the pane's transport handling matches across
			// scopes (the line format is display-only otherwise).
			if strings.HasPrefix(rest, "https://") || strings.HasPrefix(rest, "http://") {
				row.URL = rest
				row.Transport = "url"
			} else {
				row.Transport = "stdio"
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// claudeEmptyList reports the vendor's "nothing configured" prose. The match
// is anchored at the start of the line so a server detail that happens to
// contain the words never reads as an empty list.
func claudeEmptyList(line string) bool {
	l := strings.ToLower(line)
	return strings.HasPrefix(l, "no mcp servers") ||
		strings.HasPrefix(l, "no servers") ||
		strings.HasPrefix(l, "no connectors")
}

// claudeRowName accepts only server-name-shaped tokens so banner prose
// ("Checking MCP server health") never fabricates a connector row.
func claudeRowName(name string) bool {
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func trimHealth(s string) string {
	// Health rides a " - <glyph> …" tail; the glyph and wording moved
	// between releases (✓ Connected, ✗/✘ Failed to connect) and a failure
	// carries a "— reason" detail behind it. Cut at the first health
	// marker, whatever follows it.
	for _, marker := range []string{" - ✓", " - ✗", " - ✘", " - ⏸", "- ✓", "- ✗", "- ✘", "- ⏸"} {
		if i := strings.Index(s, marker); i >= 0 {
			s = strings.TrimSpace(s[:i])
			break
		}
	}
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "-"))
	// The target carries a transport tag on current CLIs: "…/mcp (HTTP)".
	for _, tag := range []string{" (HTTP)", " (SSE)", " (stdio)"} {
		s = strings.TrimSuffix(s, tag)
	}
	return strings.TrimSpace(s)
}

func listClaudeFile(path string, layer mcp.Layer) ([]mcp.Server, error) {
	raw, err := readJSONFile(path)
	if err != nil || raw == nil {
		return nil, err
	}
	obj, _ := raw["mcpServers"].(map[string]any)
	rows := make([]mcp.Server, 0, len(obj))
	for name, e := range obj {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, claudeServer(name, entry, layer))
	}
	return rows, nil
}

func claudeServer(name string, e map[string]any, layer mcp.Layer) mcp.Server {
	s := mcp.Server{
		Name: name, Layer: layer.ID, Path: layer.Path, Scope: layer.Scope, Owned: layer.Writable,
		Command: strOf(e["command"]),
		URL:     strOf(e["url"]),
		Auth:    strOf(e["auth"]),
		Args:    stringsOf(e["args"]),
		Env:     redact(stringMap(e["env"])),
		Headers: redact(stringMap(e["headers"])),
	}
	if s.URL != "" {
		s.Transport = "url"
	} else {
		s.Transport = "stdio"
	}
	return s
}

// Add upserts one server. Project scope edits `.mcp.json` in place; user
// scope goes through `claude mcp add … --scope user` — the vendor binary is
// the only writer for its user store.
func (d Claude) Add(p Paths, scope, name string, entry mcp.Entry) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	if err := validEntry(entry); err != nil {
		return err
	}
	if err := claudeHeadersOK(entry); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	if sc == "user" {
		return d.addUser(name, entry)
	}
	path := claudePaths{p}.project()
	if path == "" {
		return fmt.Errorf("select a workspace first")
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
	servers[name] = claudeEntryMap(entry, prev)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

func claudeHeadersOK(e mcp.Entry) error {
	if e.Auth == "bearer" && strings.TrimSpace(e.BearerToken) == "" && len(e.Headers) == 0 {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (Claude) addUser(name string, e mcp.Entry) error {
	if !installed(claudeBin) {
		return fmt.Errorf("%s is not installed or not on PATH", claudeBin)
	}
	_, err := runVendorCLI(claudeBin, claudeAddArgs(name, e)...)
	return err
}

// claudeAddArgs builds `claude mcp add` for the user store. Measured against
// claude 2.1.277 (2026-09-18): --env and --header are variadic and swallow
// the positionals that follow them, so the name comes first and every flag
// rides behind it; a stdio command goes after `--` or an argument starting
// with "-" would be parsed as an option (the vendor's own examples use the
// same shape).
func claudeAddArgs(name string, e mcp.Entry) []string {
	args := []string{"mcp", "add", name, "--scope", "user"}
	if url := strings.TrimSpace(e.URL); url != "" {
		args = append(args, url, "--transport", "http")
		for _, k := range sortKeys(headersOf(e)) {
			args = append(args, "--header", k+": "+headersOf(e)[k])
		}
		return args
	}
	for _, k := range sortKeys(e.Env) {
		args = append(args, "--env", k+"="+e.Env[k])
	}
	args = append(args, "--")
	args = append(args, strings.TrimSpace(e.Command))
	args = append(args, e.Args...)
	return args
}

// Toggle is refused in both scopes: `.mcp.json` has no disabled field and
// the user store is Claude Code's to toggle.
func (Claude) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	if _, err := guestScope(scope); err != nil {
		return err
	}
	return fmt.Errorf(ToggleRefusal)
}

// Remove deletes the server. Project scope edits the file; user scope runs
// `claude mcp remove <name> --scope user`.
func (d Claude) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	if sc == "user" {
		if !installed(claudeBin) {
			return fmt.Errorf("%s is not installed or not on PATH", claudeBin)
		}
		_, err := runVendorCLI(claudeBin, "mcp", "remove", name, "--scope", "user")
		return err
	}
	path := claudePaths{p}.project()
	if path == "" {
		return fmt.Errorf("select a workspace first")
	}
	raw, err := readJSONFile(path)
	if err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("no MCP config at %s", path)
	}
	servers, _ := raw["mcpServers"].(map[string]any)
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q is not in %s", name, path)
	}
	delete(servers, name)
	raw["mcpServers"] = servers
	return writeJSONFile(path, raw)
}

// claudeEntryMap merges the new definition into the previous entry so keys
// PiCode does not model survive an update (merge-by-key, ADR-0150).
func claudeEntryMap(e mcp.Entry, prev map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range prev {
		out[k] = v
	}
	if url := strings.TrimSpace(e.URL); url != "" {
		delete(out, "command")
		delete(out, "args")
		delete(out, "env")
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
	out["command"] = strings.TrimSpace(e.Command)
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

// readJSONFile is the shared reader for the JSON-codec drivers (Claude
// project files, Omp, Antigravity, OpenCode, Muse): a missing file is an
// empty config. Strict JSON is tried first; on failure the JSONC escapes
// vendors hand-write (// and /* */ comments, one trailing comma before a
// closer) are stripped and the parse retried — tolerate on read what the
// vendor tolerates, normalized in memory. Writes stay strict JSON, so a
// rewrite lands canonical. A file neither form parses refuses every write
// on top of it (ADR-0150).
func readJSONFile(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		if lenient, lerr := lenientJSON(b); lerr == nil {
			return lenient, nil
		}
		return nil, &parseError{path, "is not valid JSON"}
	}
	return raw, nil
}

func readJSONFileOrEmpty(path string) (map[string]any, error) {
	raw, err := readJSONFile(path)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw, nil
}

func strOf(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func stringsOf(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func stringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for k, x := range m {
		if s, ok := x.(string); ok {
			out[k] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toAnyMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toAnySlice(s []string) []any {
	out := make([]any, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
