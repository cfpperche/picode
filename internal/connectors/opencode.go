package connectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
)

// OpenCode manages OpenCode's MCP servers in its native JSON configs: the
// user file ($XDG_CONFIG_HOME/opencode/opencode.json[.c], default
// ~/.config/opencode/opencode.json) and the workspace file
// (<workspace>/opencode.json[.c]). Both are plain files, so PiCode edits
// them directly and never shells out.
//
// Measured against opencode 1.18.31 (2026-09-18): the vendor merges
// opencode.json and opencode.jsonc, and its own `opencode mcp add` writes
// the .jsonc, which also wins a name conflict. This driver therefore lists
// both layers (jsonc first, the winner), and a mutation targets the file
// the name already lives in, else the jsonc when it exists, else
// opencode.json.
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

// userDir honors XDG_CONFIG_HOME: OpenCode resolves its config through the
// XDG base-directory rule, so an override must also move the file PiCode
// edits.
func (c opencodePaths) userDir() string {
	if x := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); x != "" {
		return filepath.Join(x, "opencode")
	}
	return filepath.Join(c.home(), ".config", "opencode")
}

// candidates returns the config files of one scope in vendor precedence
// order: the .jsonc (what `opencode mcp add` writes and what wins a name
// conflict) before opencode.json.
func (c opencodePaths) candidates(scope string) []string {
	dir := c.userDir()
	if scope == "project" {
		if c.Cwd == "" {
			return nil
		}
		dir = c.Cwd
	}
	return []string{
		filepath.Join(dir, "opencode.jsonc"),
		filepath.Join(dir, "opencode.json"),
	}
}

// opencodeLayerID distinguishes the two user files in reports: the pane's
// remove-confirm names the file the row is managed in.
func opencodeLayerID(scope, path string) string {
	if strings.HasSuffix(path, ".jsonc") {
		return "opencode-" + scope + "c"
	}
	return "opencode-" + scope
}

func (OpenCode) Layers(p Paths) []mcp.Layer {
	pv := opencodePaths{p}
	out := []mcp.Layer{}
	for _, scope := range []string{"user", "project"} {
		candidates := pv.candidates(scope)
		if candidates == nil {
			continue
		}
		anyExists := false
		for _, path := range candidates {
			if st, err := os.Stat(path); err == nil && !st.IsDir() {
				anyExists = true
				break
			}
		}
		if anyExists {
			// List what exists, winner first; a file the vendor ignores is
			// never fabricated as a layer.
			for _, path := range candidates {
				label := "OpenCode user"
				if scope == "project" {
					label = "This folder"
				}
				if strings.HasSuffix(path, ".jsonc") {
					label += " (jsonc)"
				}
				layer := mcp.Layer{ID: opencodeLayerID(scope, path), Label: label, Path: path, Scope: scope, Writable: true}
				if st, err := os.Stat(path); err == nil && !st.IsDir() {
					layer.Exists = true
					out = append(out, layer)
				}
			}
			continue
		}
		// Nothing yet: opencode.json is the default a write creates.
		layer := mcp.Layer{ID: opencodeLayerID(scope, candidates[1]), Label: "OpenCode user", Path: candidates[1], Scope: scope, Writable: true}
		if scope == "project" {
			layer.Label = "This folder"
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
		for _, s := range opencodeServers(raw, layer) {
			if !seen[s.Name] {
				rep.Servers = append(rep.Servers, s)
				seen[s.Name] = true
			}
		}
	}
	// The vendor list is the headless status signal (ADR-0150 decision 4):
	// report what it says, never invent. A slow or missing CLI is skipped.
	if live := opencodeLive(p); len(live) > 0 {
		for i := range rep.Servers {
			if state, ok := live[rep.Servers[i].Name]; ok {
				rep.Servers[i].Live = state
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

// Add upserts one server, targeting the file the name already lives in —
// the .jsonc when it holds it (the vendor's conflict winner) — else the
// jsonc when it exists, else opencode.json.
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
	pv := opencodePaths{p}
	candidates := pv.candidates(sc)
	if candidates == nil {
		return fmt.Errorf("select a workspace first")
	}
	path, err := opencodeTarget(candidates, name)
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

// opencodeCandidateFiles returns the candidate files that exist, in
// precedence order, parsed; an existing file that fails to parse is a
// refusal, never a silent skip — a shadowing malformed file would make any
// write invisible to the vendor.
type opencodeConfig struct {
	path string
	raw  map[string]any
}

func opencodeCandidateFiles(candidates []string) ([]opencodeConfig, error) {
	out := make([]opencodeConfig, 0, len(candidates))
	for _, path := range candidates {
		st, err := os.Stat(path)
		if err != nil || st.IsDir() {
			continue
		}
		raw, err := readJSONFile(path)
		if err != nil {
			return nil, err
		}
		out = append(out, opencodeConfig{path: path, raw: raw})
	}
	return out, nil
}

// opencodeTarget resolves the file a name belongs to: the file already
// holding it wins, else the first file that exists, else opencode.json.
func opencodeTarget(candidates []string, name string) (string, error) {
	files, err := opencodeCandidateFiles(candidates)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if servers, _ := f.raw["mcp"].(map[string]any); servers != nil {
			if _, ok := servers[name]; ok {
				return f.path, nil
			}
		}
	}
	if len(files) > 0 {
		return files[0].path, nil
	}
	return candidates[len(candidates)-1], nil
}

// opencodeExisting resolves Toggle/Remove targets: the file that holds the
// name is edited; when neither file has it, the error names where it would
// be — the same observable the single-file drivers give.
func opencodeExisting(candidates []string, name string) (string, error) {
	files, err := opencodeCandidateFiles(candidates)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if servers, _ := f.raw["mcp"].(map[string]any); servers != nil {
			if _, ok := servers[name]; ok {
				return f.path, nil
			}
		}
	}
	if len(files) > 0 {
		return "", fmt.Errorf("server %q is not in %s", name, files[0].path)
	}
	return "", fmt.Errorf("server %q is not in %s", name, candidates[len(candidates)-1])
}

// Toggle flips the entry's `enabled` field in the file that owns the name —
// OpenCode has a real per-server switch (driver toggle: "entry").
func (d OpenCode) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	pv := opencodePaths{p}
	candidates := pv.candidates(sc)
	if candidates == nil {
		return fmt.Errorf("select a workspace first")
	}
	path, err := opencodeExisting(candidates, name)
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

// Remove deletes the entry from the file that owns it and nothing else.
func (d OpenCode) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	pv := opencodePaths{p}
	candidates := pv.candidates(sc)
	if candidates == nil {
		return fmt.Errorf("select a workspace first")
	}
	path, err := opencodeExisting(candidates, name)
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

// opencodeLive parses `opencode mcp list` — the vendor's own headless
// status report (list MCP servers and their status). Measured 1.18.31:
// box-drawing lines like "●  ○ probe disabled" (off), "●  ✗ remote failed"
// (connection failure), healthy rows carry ● and no failure word. ANSI
// colors ride the output and are stripped. Names not found in the probe
// keep the honest empty Live.
func opencodeLive(p Paths) map[string]string {
	if !installed(opencodeBin) {
		return nil
	}
	key := "opencode|" + p.home() + "|" + p.Cwd
	if live, ok := cachedStatus(key); ok {
		return live
	}
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()
	out, err := runVendorCLIContext(ctx, opencodeBin, "mcp", "list")
	if err != nil {
		return nil // a hung or broken CLI is skipped this round, never invented
	}
	live := map[string]string{}
	ansi := func(s string) string {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			if s[i] == 0x1b {
				for i < len(s) && s[i] != 'm' {
					i++
				}
				continue
			}
			b.WriteByte(s[i])
		}
		return b.String()
	}
	for _, raw := range strings.Split(out, "\n") {
		line := strings.Map(func(r rune) rune {
			if strings.ContainsRune("│┌└─├┤", r) {
				return -1
			}
			return r
		}, ansi(raw))
		trimmed := strings.TrimLeft(line, " ●○✗*")
		if trimmed == "" || trimmed == line {
			continue // no status glyph on this line
		}
		name := strings.Fields(trimmed)
		if len(name) == 0 {
			continue
		}
		switch {
		case strings.Contains(line, "disabled"):
			// off is the enabled flag's own pane state, not connection health
		case strings.Contains(line, "failed"):
			live[name[0]] = "failed"
		default:
			live[name[0]] = "live"
		}
	}
	putStatus(key, live)
	return live
}
