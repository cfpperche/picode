package connectors

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/mcp"
	"gopkg.in/yaml.v3"
)

// Hermes manages Hermes Agent's MCP servers in its native config file: the
// single YAML document at ~/.hermes/config.yaml. It is a plain file, so
// PiCode edits it directly and never shells out; there is no per-workspace
// variant, so project scope refuses with a pointer to the one file.
//
// YAML has comments and key order worth keeping, so this codec edits the
// document as a yaml.Node tree instead of a generic map: only the bytes of
// the touched entry change shape, every other key keeps its position and
// its comments, and entries' own fields (`auth`, `tools.include/exclude`,
// …) survive an update through merge-by-key on the node tree. The block is
// `mcp_servers: {name: {command, args, env | url, headers, enabled?,
// auth?, tools?}}`; the entry's `enabled` field is the toggle (driver
// toggle: "entry"), and `${VAR}` placeholders travel as plain scalars —
// they are never interpreted (ADR-0150).
type Hermes struct{}

const hermesBin = "hermes"

func (Hermes) ID() string { return "hermes" }

func (Hermes) Bin() string { return hermesBin }

// AuthHint is Hermes's own sign-in command for one server (the hermes CLI
// dispatches `mcp` and `login`, so `hermes mcp login <name>` is its path).
func (Hermes) AuthHint(name string) AuthHint {
	cmd := hermesBin + " mcp login " + name
	return AuthHint{Text: "sign in from a terminal instead: " + cmd, Command: cmd}
}

type hermesPaths struct{ Paths }

// user is the one Hermes config file.
func (c hermesPaths) user() string {
	return filepath.Join(c.home(), ".hermes", "config.yaml")
}

// scopePath resolves the one file or refuses: Hermes keeps a single config
// file — there is no per-workspace variant to create.
func (c hermesPaths) scopePath(scope string) (string, error) {
	switch scope {
	case "user", "":
		return c.user(), nil
	default:
		return "", fmt.Errorf("Hermes keeps one config file (%s); there is no per-workspace file", c.user())
	}
}

func (Hermes) Layers(p Paths) []mcp.Layer {
	pv := hermesPaths{p}
	out := []mcp.Layer{}
	user := mcp.Layer{ID: "hermes-user", Label: "Hermes user", Path: pv.user(), Scope: "user", Writable: true}
	if st, err := os.Stat(user.Path); err == nil && !st.IsDir() {
		user.Exists = true
	}
	return append(out, user)
}

// List reports the servers Hermes would load. Live stays empty — the
// driver never invents state it cannot see (ADR-0150 decision 4).
func (d Hermes) List(p Paths) (mcp.Report, error) {
	rep := mcp.Report{
		Adapter:           mcp.Adapter{Source: "native:hermes", Installed: installed(hermesBin)},
		Presets:           mcp.Presets(),
		Layers:            d.Layers(p),
		Servers:           []mcp.Server{},
		ConnectorPackages: []mcp.ConnectorPackage{},
	}
	root, err := readYAMLRoot(rep.Layers[0].Path)
	if err != nil {
		// The one file exists but does not parse: the layer reports the
		// reason and the pane shows a blocked line for it (ADR-0150) — the
		// report still answers 200.
		blockLayer(&rep.Layers[0], err)
		root = nil
	}
	servers := yamlChild(root, "mcp_servers")
	if servers == nil || servers.Kind != yaml.MappingNode {
		return rep, nil
	}
	for i := 0; i+1 < len(servers.Content); i += 2 {
		name, entry := servers.Content[i].Value, servers.Content[i+1]
		if entry.Kind != yaml.MappingNode {
			continue
		}
		s := mcp.Server{
			Name: name, Layer: rep.Layers[0].ID, Path: rep.Layers[0].Path, Scope: "user", Owned: true,
			Command: yamlScalar(yamlChild(entry, "command")),
			URL:     yamlScalar(yamlChild(entry, "url")),
			Args:    yamlScalars(yamlChild(entry, "args")),
			Env:     redact(yamlStringMap(yamlChild(entry, "env"))),
			Headers: redact(yamlStringMap(yamlChild(entry, "headers"))),
		}
		if enabled := yamlChild(entry, "enabled"); enabled != nil && enabled.Tag == "!!bool" {
			s.Disabled = enabled.Value != "true"
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

// Add upserts one server in the config file, creating it on request.
func (d Hermes) Add(p Paths, scope, name string, entry mcp.Entry) error {
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
	path, err := hermesPaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	root, err := readYAMLRoot(path)
	if err != nil {
		return err
	}
	servers := yamlChild(root, "mcp_servers")
	if servers == nil || servers.Kind != yaml.MappingNode {
		servers = &yaml.Node{Kind: yaml.MappingNode}
		if root == nil {
			root = &yaml.Node{Kind: yaml.MappingNode}
		}
		setYAMLPair(root, "mcp_servers", servers)
	}
	setYAMLPair(servers, name, hermesEntryNode(entry, yamlChild(servers, name)))
	return writeYAMLFile(path, root)
}

// Toggle flips the entry's `enabled` field — Hermes has a real per-server
// switch (driver toggle: "entry"). The edit touches one scalar; comments
// and siblings keep their bytes.
func (d Hermes) Toggle(p Paths, scope, name string, disabled bool) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := hermesPaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	root, _, entry, err := hermesEntryForWrite(path, name)
	if err != nil {
		return err
	}
	if entry.Kind != yaml.MappingNode {
		return fmt.Errorf("server %q in %s is not a mapping", name, path)
	}
	setYAMLPair(entry, "enabled", yamlBool(!disabled))
	return writeYAMLFile(path, root)
}

// Remove deletes the entry pair and nothing else; the `mcp_servers` block
// (and its comments) stays even when it empties.
func (d Hermes) Remove(p Paths, scope, name string) error {
	if err := mcp.ValidName(name); err != nil {
		return err
	}
	sc, err := guestScope(scope)
	if err != nil {
		return err
	}
	path, err := hermesPaths{p}.scopePath(sc)
	if err != nil {
		return err
	}
	root, servers, _, err := hermesEntryForWrite(path, name)
	if err != nil {
		return err
	}
	deleteYAMLPair(servers, name)
	return writeYAMLFile(path, root)
}

// hermesEntryForWrite loads the document and walks to the mcp_servers
// entry a mutation targets. A missing file or block answers "not in", the
// same observable result the JSON drivers give.
func hermesEntryForWrite(path, name string) (*yaml.Node, *yaml.Node, *yaml.Node, error) {
	root, err := readYAMLRoot(path)
	if err != nil {
		return nil, nil, nil, err
	}
	if root == nil {
		return nil, nil, nil, fmt.Errorf("server %q is not in %s", name, path)
	}
	servers := yamlChild(root, "mcp_servers")
	if servers == nil || servers.Kind != yaml.MappingNode {
		return nil, nil, nil, fmt.Errorf("server %q is not in %s", name, path)
	}
	entry := yamlChild(servers, name)
	if entry == nil {
		return nil, nil, nil, fmt.Errorf("server %q is not in %s", name, path)
	}
	return root, servers, entry, nil
}

// hermesEntryNode merges the new definition into the previous entry node:
// unknown keys and Hermes's own fields (`auth`, `tools`, …) survive an
// update because the previous pairs are kept; switching transport drops the
// other side's fields (same merge rule as internal/mcp). An empty entry
// (Toggle's shape) is a pure passthrough — it must not clear the
// definition. A new entry starts as an enabled mapping, matching Hermes's
// own default of loading every configured server.
func hermesEntryNode(e mcp.Entry, prev *yaml.Node) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode}
	if prev != nil && prev.Kind == yaml.MappingNode {
		out.Content = append(out.Content, prev.Content...)
	}
	url := strings.TrimSpace(e.URL)
	cmd := strings.TrimSpace(e.Command)
	if url == "" && cmd == "" {
		return out
	}
	if url != "" {
		deleteYAMLPair(out, "command")
		deleteYAMLPair(out, "args")
		deleteYAMLPair(out, "env")
		setYAMLPair(out, "url", yamlStr(url))
		if hs := headersOf(e); len(hs) > 0 {
			setYAMLPair(out, "headers", yamlStringMapNode(hs))
		} else {
			deleteYAMLPair(out, "headers")
		}
		return out
	}
	deleteYAMLPair(out, "url")
	deleteYAMLPair(out, "headers")
	setYAMLPair(out, "command", yamlStr(cmd))
	if len(e.Args) > 0 {
		setYAMLPair(out, "args", yamlStringsNode(e.Args))
	} else {
		deleteYAMLPair(out, "args")
	}
	if len(e.Env) > 0 {
		setYAMLPair(out, "env", yamlStringMapNode(e.Env))
	} else {
		deleteYAMLPair(out, "env")
	}
	return out
}

// readYAMLRoot parses one YAML document into an editable node tree. An
// empty or missing file decodes to nil (no root) without an error — Add
// may create it; Toggle/Remove answer "not in". Malformed YAML refuses
// with a clear error and nothing is written.
func readYAMLRoot(path string) (*yaml.Node, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		// Typed so List degrades just this layer (ADR-0150); writes on top
		// still refuse.
		return nil, &parseError{path, "is not valid YAML"}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, &parseError{path, "does not hold a config mapping"}
	}
	return root, nil
}

// writeYAMLFile re-encodes the whole document at 2-space indent (Hermes's
// own shape) and lands it atomically. Order and comments inside the tree
// are preserved by construction.
func writeYAMLFile(path string, root *yaml.Node) error {
	if root == nil {
		return fmt.Errorf("%s holds no config mapping; PiCode will not touch it", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		_ = enc.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return atomicWrite(path, buf.Bytes())
}

// yamlChild returns the value node of key in a mapping node, or nil.
func yamlChild(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// setYAMLPair replaces or appends the key/value pair, keeping the key's
// original position (and comments) on replace.
func setYAMLPair(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content, yamlStr(key), value)
}

// deleteYAMLPair removes the key's pair; a missing key is fine.
func deleteYAMLPair(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return
		}
	}
}

func yamlStr(s string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: s}
}

func yamlBool(b bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: fmt.Sprintf("%t", b)}
}

// yamlScalar reads a scalar node as the string it would render as, nil
// safe and non-scalar safe.
func yamlScalar(n *yaml.Node) string {
	if n == nil || n.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(n.Value)
}

func yamlScalars(n *yaml.Node) []string {
	if n == nil || n.Kind != yaml.SequenceNode {
		return nil
	}
	out := make([]string, 0, len(n.Content))
	for _, item := range n.Content {
		out = append(out, item.Value)
	}
	return out
}

// yamlStringMap reads a mapping of scalars for display (values only; the
// report redacts them).
func yamlStringMap(n *yaml.Node) map[string]string {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	out := map[string]string{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		if n.Content[i+1].Kind == yaml.ScalarNode {
			out[n.Content[i].Value] = n.Content[i+1].Value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// yamlStringsNode renders a []string as a block sequence node.
func yamlStringsNode(s []string) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range s {
		seq.Content = append(seq.Content, yamlStr(v))
	}
	return seq
}

// yamlStringMapNode renders a map as a mapping node with sorted keys —
// deterministic output for the golden tests and the diff.
func yamlStringMapNode(m map[string]string) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range sortKeys(m) {
		out.Content = append(out.Content, yamlStr(k), yamlStr(m[k]))
	}
	return out
}
