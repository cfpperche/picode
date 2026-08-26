// Package mcp reads and writes pi-mcp-adapter config files.
// Shape is the adapter's: { "mcpServers": { name: { command|url, ... } } }.
// We never invent a PiCode-only file. Layers match the adapter (v2.28).
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const AdapterSource = "npm:pi-mcp-adapter"

// Layer is one adapter config file.
type Layer struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Writable bool   `json:"writable"`
	Scope    string `json:"scope"` // user | project | agent | import
}

// Server is one named MCP server after merge (highest layer wins).
type Server struct {
	Name      string            `json:"name"`
	Layer     string            `json:"layer"`
	Path      string            `json:"path"`
	Scope     string            `json:"scope"`
	Transport string            `json:"transport"` // stdio | url | socket
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Disabled  bool              `json:"disabled"`
	Owned     bool              `json:"owned"`
}

// Preset is a known server from the adapter.
type Preset struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
	Entry   Entry  `json:"entry"`
}

// Entry is the adapter ServerEntry subset we write.
type Entry struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Auth    string            `json:"auth,omitempty"`
	Cwd     string            `json:"cwd,omitempty"`
}

// Report is GET /api/mcp.
type Report struct {
	Adapter  Adapter    `json:"adapter"`
	Layers   []Layer    `json:"layers"`
	Servers  []Server   `json:"servers"`
	Presets  []Preset   `json:"presets"`
	Imports  []string   `json:"imports"`
	Found    []HostInfo `json:"found"`
	WriteDir string     `json:"writeDir,omitempty"`
}

// Adapter reports whether pi-mcp-adapter is a configured package.
type Adapter struct {
	Installed bool   `json:"installed"`
	Source    string `json:"source"`
}

// Paths locates adapter files. Home empty → os.UserHomeDir.
type Paths struct {
	Home     string
	Cwd      string // workspace folder; empty skips project layers
	AgentCwd string // unique agent work path; empty skips agent layer
}

var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Presets copied from pi-mcp-adapter 2.28 KNOWN_SERVER_PRESETS.
func Presets() []Preset {
	return []Preset{
		{ID: "deepwiki", Name: "DeepWiki", Summary: "Ask questions about public GitHub repositories.", Entry: Entry{URL: "https://mcp.deepwiki.com/mcp"}},
		{ID: "context7", Name: "Context7", Summary: "Look up current library documentation and examples.", Entry: Entry{URL: "https://mcp.context7.com/mcp"}},
		{ID: "notion", Name: "Notion", Summary: "Search and work with your Notion workspace.", Entry: Entry{URL: "https://mcp.notion.com/mcp", Auth: "oauth"}},
		{ID: "github", Name: "GitHub", Summary: "Work with GitHub through your Copilot account.", Entry: Entry{URL: "https://api.githubcopilot.com/mcp", Auth: "oauth"}},
		{ID: "chrome-devtools", Name: "Chrome DevTools", Summary: "Inspect and automate a local Chrome browser.", Entry: Entry{Command: "npx", Args: []string{"-y", "chrome-devtools-mcp@1.6.0"}}},
	}
}

func (p Paths) home() string {
	if p.Home != "" {
		return p.Home
	}
	h, _ := os.UserHomeDir()
	return h
}

func (p Paths) PiGlobal() string {
	return filepath.Join(p.home(), ".pi", "agent", "mcp.json")
}

func (p Paths) SharedGlobal() string {
	return filepath.Join(p.home(), ".config", "mcp", "mcp.json")
}

func (p Paths) AgentsGlobal() string {
	return filepath.Join(p.home(), ".agents", "mcp.json")
}

func (p Paths) AgentsNested() string {
	return filepath.Join(p.home(), ".agents", "mcp", "mcp.json")
}

func (p Paths) SharedProject() string {
	if p.Cwd == "" {
		return ""
	}
	return filepath.Join(p.Cwd, ".mcp.json")
}

func (p Paths) PiProject() string {
	if p.Cwd == "" {
		return ""
	}
	return filepath.Join(p.Cwd, ".pi", "mcp.json")
}

func (p Paths) AgentProject() string {
	if p.AgentCwd == "" {
		return ""
	}
	return filepath.Join(p.AgentCwd, ".pi", "mcp.json")
}

// Layers lists adapter files in merge order (last wins).
// Host imports (from mcp.json imports) are lowest precedence.
func (p Paths) Layers() []Layer {
	home := p.home()
	if home == "" {
		return nil
	}
	out := hostLayers(p)
	out = append(out, []Layer{
		{ID: "shared-global", Label: "Shared (~/.config/mcp)", Path: p.SharedGlobal(), Scope: "import", Writable: false},
		{ID: "agents-global", Label: "Shared (~/.agents)", Path: p.AgentsGlobal(), Scope: "import", Writable: false},
		{ID: "agents-nested", Label: "Shared (~/.agents/mcp)", Path: p.AgentsNested(), Scope: "import", Writable: false},
		{ID: "pi-global", Label: "This machine", Path: p.PiGlobal(), Scope: "user", Writable: true},
	}...)
	if path := p.SharedProject(); path != "" {
		out = append(out, Layer{ID: "shared-project", Label: "This folder", Path: path, Scope: "project", Writable: true})
	}
	if path := p.PiProject(); path != "" && path != p.SharedProject() {
		out = append(out, Layer{ID: "pi-project", Label: "This folder (Pi)", Path: path, Scope: "project", Writable: true})
	}
	if path := p.AgentProject(); path != "" && path != p.PiProject() && path != p.SharedProject() {
		out = append(out, Layer{ID: "agent", Label: "This agent", Path: path, Scope: "agent", Writable: true})
	}
	for i := range out {
		if st, err := os.Stat(out[i].Path); err == nil && !st.IsDir() {
			out[i].Exists = true
		}
	}
	return out
}

// List merges servers. Missing files are empty.
func List(p Paths) (Report, error) {
	rep := Report{Adapter: Adapter{Source: AdapterSource}, Presets: Presets(), Layers: p.Layers()}
	if raw, err := readFile(p.PiGlobal()); err == nil && raw != nil {
		rep.Imports = importKindsOf(raw)
	}
	rep.Found = FoundHosts(p)
	seen := map[string]Server{}
	order := []string{}
	for _, layer := range rep.Layers {
		raw, err := readFile(layer.Path)
		if (err != nil || raw == nil) && strings.HasSuffix(layer.Path, ".toml") {
			if srv := serversFromTomlFile(layer.Path); len(srv) > 0 {
				raw = map[string]any{"mcpServers": toAnyServers(srv)}
				err = nil
			}
		}
		if err != nil {
			if layer.Scope == "import" {
				continue
			}
			return rep, err
		}
		if raw == nil {
			continue
		}
		entries := serversOf(raw)
		if layer.Scope == "import" {
			entries = serversOfHost(raw)
		}
		for name, entry := range entries {
			if _, ok := seen[name]; !ok {
				order = append(order, name)
			}
			if prev, ok := seen[name]; ok && overlayOnly(entry) {
				prev.Disabled = boolOf(entry["disabled"])
				prev.Layer = layer.ID
				prev.Path = layer.Path
				prev.Owned = layer.Writable
				prev.Scope = layer.Scope
				seen[name] = prev
				continue
			}
			seen[name] = serverFrom(name, entry, layer)
		}
	}
	rep.Servers = make([]Server, 0, len(order))
	for _, name := range order {
		rep.Servers = append(rep.Servers, seen[name])
	}
	return rep, nil
}

// WriteScope is user | project | agent.
func WritePath(p Paths, scope string) (string, error) {
	switch strings.TrimSpace(scope) {
	case "", "user":
		return p.PiGlobal(), nil
	case "project":
		path := p.SharedProject()
		if path == "" {
			return "", fmt.Errorf("select a workspace first")
		}
		return path, nil
	case "agent":
		path := p.AgentProject()
		if path == "" {
			return "", fmt.Errorf("this agent shares its folder — use This workspace, or give it its own folder")
		}
		return path, nil
	default:
		return "", fmt.Errorf("scope must be user, project, or agent")
	}
}

// Add upserts a server into the scope's owned file.
func Add(p Paths, scope, name string, entry Entry) error {
	if err := ValidName(name); err != nil {
		return err
	}
	if err := validEntry(entry); err != nil {
		return err
	}
	path, err := WritePath(p, scope)
	if err != nil {
		return err
	}
	raw, err := readFileOrEmpty(path)
	if err != nil {
		return err
	}
	servers := serversOf(raw)
	servers[name] = entryToMap(entry, servers[name])
	setServers(raw, servers)
	return writeFile(path, raw)
}

// Toggle sets or clears disabled on the owned file for this server.
// If the winning definition lives in an import we don't own, write a
// disabled-only stub into the matching owned overlay (adapter pattern).
func Toggle(p Paths, scope, name string, disabled bool) error {
	if err := ValidName(name); err != nil {
		return err
	}
	path, err := WritePath(p, scope)
	if err != nil {
		return err
	}
	raw, err := readFileOrEmpty(path)
	if err != nil {
		return err
	}
	servers := serversOf(raw)
	cur, ok := servers[name]
	if !ok {
		if !disabled {
			return fmt.Errorf("server %q is not in %s", name, path)
		}
		// Overlay stub — do not copy command/url/credentials.
		servers[name] = map[string]any{"disabled": true}
		setServers(raw, servers)
		return writeFile(path, raw)
	}
	if disabled {
		cur["disabled"] = true
	} else {
		delete(cur, "disabled")
		if len(cur) == 0 {
			delete(servers, name)
		}
	}
	if len(cur) > 0 {
		servers[name] = cur
	}
	setServers(raw, servers)
	return writeFile(path, raw)
}

// Remove deletes a server from an owned file. Import layers are refused.
func Remove(p Paths, scope, name string) error {
	if err := ValidName(name); err != nil {
		return err
	}
	path, err := WritePath(p, scope)
	if err != nil {
		return err
	}
	raw, err := readFile(path)
	if err != nil {
		return err
	}
	if raw == nil {
		return fmt.Errorf("no MCP config at %s", path)
	}
	servers := serversOf(raw)
	if _, ok := servers[name]; !ok {
		return fmt.Errorf("server %q is not in this file — disable it instead", name)
	}
	delete(servers, name)
	setServers(raw, servers)
	return writeFile(path, raw)
}

func ValidName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !nameRe.MatchString(name) {
		return fmt.Errorf("name must be letters, digits, dot, dash, or underscore")
	}
	return nil
}

func validEntry(e Entry) error {
	cmd := strings.TrimSpace(e.Command)
	url := strings.TrimSpace(e.URL)
	if cmd == "" && url == "" {
		return fmt.Errorf("command or url is required")
	}
	if cmd != "" && url != "" {
		return fmt.Errorf("use command or url, not both")
	}
	if cmd != "" {
		if hasBad(cmd) {
			return fmt.Errorf("command has invalid characters")
		}
		for _, a := range e.Args {
			if strings.ContainsAny(a, "\n\r") {
				return fmt.Errorf("args must be a single line")
			}
		}
		return nil
	}
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}

func hasBad(s string) bool {
	for _, r := range s {
		if r == ';' || r == '|' || r == '&' || r == '`' || r == '\n' || r == '\r' || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func overlayOnly(entry map[string]any) bool {
	_, cmd := entry["command"]
	_, url := entry["url"]
	_, sock := entry["socket"]
	return !cmd && !url && !sock
}

func serverFrom(name string, entry map[string]any, layer Layer) Server {
	s := Server{
		Name:     name,
		Layer:    layer.ID,
		Path:     layer.Path,
		Scope:    layer.Scope,
		Owned:    layer.Writable,
		Disabled: boolOf(entry["disabled"]),
		Command:  strOf(entry["command"]),
		URL:      strOf(entry["url"]),
		Args:     stringsOf(entry["args"]),
		Env:      stringMap(entry["env"]),
		Headers:  stringMap(entry["headers"]),
	}
	switch {
	case s.URL != "":
		s.Transport = "url"
	case strOf(entry["socket"]) != "":
		s.Transport = "socket"
	default:
		s.Transport = "stdio"
	}
	return s
}

func serversOf(raw map[string]any) map[string]map[string]any {
	out := map[string]map[string]any{}
	if raw == nil {
		return out
	}
	var obj any
	if v, ok := raw["mcpServers"]; ok {
		obj = v
	} else {
		obj = raw["mcp-servers"]
	}
	m, ok := obj.(map[string]any)
	if !ok {
		return out
	}
	for name, v := range m {
		if inner, ok := v.(map[string]any); ok {
			out[name] = inner
		}
	}
	return out
}

func setServers(raw map[string]any, servers map[string]map[string]any) {
	delete(raw, "mcp-servers")
	if len(servers) == 0 {
		delete(raw, "mcpServers")
		return
	}
	next := map[string]any{}
	for k, v := range servers {
		next[k] = v
	}
	raw["mcpServers"] = next
}

func entryToMap(e Entry, prev map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range prev {
		out[k] = v
	}
	// Switching transport drops the other side's fields (adapter merge rule).
	if strings.TrimSpace(e.Command) != "" {
		delete(out, "url")
		delete(out, "headers")
		delete(out, "auth")
		delete(out, "socket")
		out["command"] = strings.TrimSpace(e.Command)
		if len(e.Args) > 0 {
			arr := make([]any, len(e.Args))
			for i, a := range e.Args {
				arr[i] = a
			}
			out["args"] = arr
		} else {
			delete(out, "args")
		}
		if len(e.Env) > 0 {
			out["env"] = toAnyMap(e.Env)
		}
	}
	if strings.TrimSpace(e.URL) != "" {
		delete(out, "command")
		delete(out, "args")
		delete(out, "env")
		delete(out, "cwd")
		delete(out, "socket")
		out["url"] = strings.TrimSpace(e.URL)
		if len(e.Headers) > 0 {
			out["headers"] = toAnyMap(e.Headers)
		}
		if e.Auth != "" {
			out["auth"] = e.Auth
		}
	}
	return out
}

func toAnyMap(m map[string]string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

func readFile(path string) (map[string]any, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseConfig(b)
}

func readFileOrEmpty(path string) (map[string]any, error) {
	raw, err := readFile(path)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw, nil
}

func parseConfig(b []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err == nil {
		return raw, nil
	}
	if err := json.Unmarshal([]byte(stripJSONC(string(b))), &raw); err != nil {
		return nil, fmt.Errorf("mcp.json is not JSON")
	}
	return raw, nil
}

func writeFile(path string, raw map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmp, path); err2 != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

func strOf(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolOf(v any) bool {
	b, _ := v.(bool)
	return b
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

// stripJSONC drops // and /* */ outside strings so we can read Cursor-style files.
func stripJSONC(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inStr := false
	esc := false
	i := 0
	for i < len(s) {
		c := s[i]
		if inStr {
			b.WriteByte(c)
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
			i++
			continue
		}
		if c == '"' {
			inStr = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			if i+1 < len(s) {
				i += 2
			}
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}
