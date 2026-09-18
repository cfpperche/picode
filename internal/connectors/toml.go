package connectors

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// The TOML codecs (Codex, Grok) share one surgical-splice contract
// (ADR-0150): the native config keeps MCP servers in `[mcp_servers.<name>]`
// tables, and every write replaces or deletes only that table's byte span —
// comments, unrelated tables and ${VAR} placeholders survive byte-for-byte.
// The spliced text is re-parsed before it is written and the write is
// atomic, so corruption fails loudly instead of landing on disk.

// tomlHeader reports whether a header line opens this server's table
// (bare, double-quoted or literal-quoted key form).
func tomlHeader(line, name string) bool {
	switch strings.TrimRight(line, " \t\r\n") {
	case "[mcp_servers." + name + "]",
		`[mcp_servers."` + name + `"]`,
		"[mcp_servers.'" + name + "']":
		return true
	}
	return false
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
		if !tomlHeader(texts[i], name) {
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

// tomlBlock renders one `[mcp_servers.<name>]` table from the merged entry.
// The entry is marshaled inside a wrapper document so nested maps (env,
// unknown tables) come out as proper [mcp_servers.<name>.…] sub-tables —
// marshaling the bare entry would emit them at the root level — then the
// wrapper's preamble is cut and everything from this table's header on is
// kept. Vendor names the CLI in the error ("codex", "grok").
func tomlBlock(vendor, name string, entry map[string]any) (string, error) {
	wrapper := map[string]any{"mcp_servers": map[string]any{name: entry}}
	b, err := toml.Marshal(wrapper)
	if err != nil {
		return "", fmt.Errorf("cannot encode %s for the %s config: %w", name, vendor, err)
	}
	lines := strings.SplitAfter(string(b), "\n")
	for i, line := range lines {
		if tomlHeader(line, name) {
			return strings.Join(lines[i:], ""), nil
		}
	}
	return "", fmt.Errorf("cannot encode %s for the %s config", name, vendor)
}

// spliceTOMLTable replaces (block == "") or rewrites the table's byte span,
// validates the whole result, and lands it atomically.
func spliceTOMLTable(path, text, name, block string) error {
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

// readTOMLFile parses a config into a generic map. A missing file is an
// empty config; malformed TOML is an error — refuse loudly, never write on
// top of a file we could not read (ADR-0150).
func readTOMLFile(path string) (map[string]any, error) {
	text, err := readTOMLText(path)
	if err != nil || strings.TrimSpace(text) == "" {
		return map[string]any{}, err
	}
	var raw map[string]any
	if err := toml.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("%s is not valid TOML", path)
	}
	return raw, nil
}

func readTOMLText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(b), nil
}

// parseTOMLServers returns the mcp_servers entries of a config text.
// Malformed TOML refuses the whole operation before any splice is attempted.
func parseTOMLServers(text, path string) (map[string]map[string]any, error) {
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
