package clisettings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// insertValue adds a key the document does not have yet, in the place the
// format expects it, and returns the new text. Everything already in the file
// keeps its bytes: an insert only ever adds.
func insertValue(text []byte, f Format, path []string, lit string) ([]byte, error) {
	switch f {
	case FormatJSON, FormatJSONC:
		return jsonInsert(text, path, lit)
	case FormatTOML:
		return tomlInsert(text, path, lit)
	case FormatYAML:
		return yamlInsert(text, path, lit)
	}
	return nil, fmt.Errorf("unknown config format %q", f)
}

// removeValue drops a key the document sets, so the CLI falls back to its own
// default, and takes the container with it when that key was the last one in
// it: a `[memories]` header or a `memory:` block left behind with nothing
// under it is not what the file said before PiCode touched it (the ADR-0099
// rule that an emptied object goes with its last key).
func removeValue(text []byte, f Format, path []string) ([]byte, error) {
	out, err := removeOne(text, f, path)
	if err != nil {
		return nil, err
	}
	for depth := len(path) - 1; depth > 0; depth-- {
		parent := path[:depth]
		doc, err := decode(out, f, "")
		if err != nil {
			return out, nil
		}
		v, ok := lookup(doc, parent)
		if !ok {
			continue
		}
		// A YAML block whose last key left decodes as nil, a TOML table as an
		// empty map. Both mean the container has nothing left to hold.
		if v != nil {
			m, isMap := v.(map[string]any)
			if !isMap || len(m) > 0 {
				break
			}
		}
		if out, err = removeOne(out, f, parent); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// removeOne drops exactly one key or table.
func removeOne(text []byte, f Format, path []string) ([]byte, error) {
	start, end, ok := valueSpan(text, f, path)
	if !ok {
		// A table or a block is a header plus its body, not a `key = value`
		// span. That is also how an emptied container is taken away.
		switch f {
		case FormatTOML:
			if out, done := removeTOMLTable(text, strings.Join(path, ".")); done {
				return out, nil
			}
		case FormatYAML:
			if out, done := removeYAMLBlock(text, path); done {
				return out, nil
			}
		}
		return text, nil
	}
	switch f {
	case FormatTOML, FormatYAML:
		// Line formats: drop the whole line, including its leading indent.
		lineStart := bytes.LastIndexByte(text[:start], '\n') + 1
		lineEnd := end
		if nl := bytes.IndexByte(text[end:], '\n'); nl >= 0 {
			lineEnd = end + nl + 1
		} else {
			lineEnd = len(text)
		}
		return append(append([]byte{}, text[:lineStart]...), text[lineEnd:]...), nil
	default:
		// JSON: drop `"key": value` and the one comma that joined it.
		keyStart := jsonKeyStart(text, path)
		if keyStart < 0 {
			return text, nil
		}
		cut := end
		rest := text[end:]
		trimmed := 0
		for trimmed < len(rest) && isSpace(rest[trimmed]) {
			trimmed++
		}
		trailing := trimmed < len(rest) && rest[trimmed] == ','
		if trailing {
			cut = end + trimmed + 1
		} else {
			// Last member: take the comma that precedes it instead, and stop
			// there. Eating the newline after it as well glued the closing
			// brace onto the member above (live QA, 2026-09-20).
			before := keyStart
			for before > 0 && isSpace(text[before-1]) {
				before--
			}
			if before > 0 && text[before-1] == ',' {
				keyStart = before - 1
			}
		}
		lineStart := bytes.LastIndexByte(text[:keyStart], '\n') + 1
		if strings.TrimSpace(string(text[lineStart:keyStart])) == "" {
			keyStart = lineStart
		}
		if trailing {
			if nl := bytes.IndexByte(text[cut:], '\n'); nl >= 0 && strings.TrimSpace(string(text[cut:cut+nl])) == "" {
				cut += nl + 1
			}
		}
		return append(append([]byte{}, text[:keyStart]...), text[cut:]...), nil
	}
}

// jsonKeyStart reports the offset of the opening quote of the key the path
// names, or -1.
func jsonKeyStart(text []byte, path []string) int {
	start, _, ok := jsonValueSpan(text, path)
	if !ok {
		return -1
	}
	// Walk back over the colon and the quoted key.
	i := start - 1
	for i >= 0 && isSpace(text[i]) {
		i--
	}
	if i < 0 || text[i] != ':' {
		return -1
	}
	i--
	for i >= 0 && isSpace(text[i]) {
		i--
	}
	if i < 0 || text[i] != '"' {
		return -1
	}
	i--
	for i >= 0 {
		if text[i] == '"' && (i == 0 || text[i-1] != '\\') {
			return i
		}
		i--
	}
	return -1
}

// jsonInsert adds the key to its parent object, creating missing parents.
func jsonInsert(text []byte, path []string, lit string) ([]byte, error) {
	if len(bytes.TrimSpace(text)) == 0 {
		text = []byte("{}\n")
	}
	// Find the deepest existing ancestor.
	depth := len(path) - 1
	for depth > 0 {
		if _, _, ok := jsonObjectSpan(text, path[:depth]); ok {
			break
		}
		depth--
	}
	// Build the member text for everything below that ancestor.
	member := quoteJSON(path[depth]) + ": "
	closing := ""
	for i := depth + 1; i < len(path); i++ {
		member += "{" + quoteJSON(path[i]) + ": "
		closing += "}"
	}
	member += lit + closing
	open, closeAt, ok := jsonObjectSpan(text, path[:depth])
	if !ok {
		return nil, fmt.Errorf("cannot place %s: its parent object is missing", strings.Join(path, "."))
	}
	empty := strings.TrimSpace(string(text[open+1:closeAt])) == ""
	outer, inner := jsonIndent(text, open, closeAt, empty)
	if empty {
		return spliceAt(text, open+1, "\n"+inner+member+"\n"+outer), nil
	}
	// The comma belongs to the member before it, not to the line above the
	// closing brace: inserting at closeAt put it alone on its own line
	// (live QA, 2026-09-20). Anchor on the last non-space byte instead.
	at := closeAt
	for at > open+1 && isSpace(text[at-1]) {
		at--
	}
	return spliceAt(text, at, ",\n"+inner+member), nil
}

// jsonObjectSpan reports the offsets of the `{` and matching `}` of the object
// at path (the document root for an empty path).
func jsonObjectSpan(text []byte, path []string) (open, closeAt int, ok bool) {
	scan := stripJSONC(text)
	if len(path) == 0 {
		o := bytes.IndexByte(scan, '{')
		if o < 0 {
			return 0, 0, false
		}
		c := matchBrace(scan, o)
		if c < 0 {
			return 0, 0, false
		}
		return o, c, true
	}
	start, end, found := jsonValueSpan(text, path)
	if !found {
		return 0, 0, false
	}
	if start >= len(scan) || scan[start] != '{' {
		return 0, 0, false
	}
	_ = end
	c := matchBrace(scan, start)
	if c < 0 {
		return 0, 0, false
	}
	return start, c, true
}

func matchBrace(scan []byte, open int) int {
	level, inString, escape := 0, false, false
	for i := open; i < len(scan); i++ {
		c := scan[i]
		if inString {
			switch {
			case escape:
				escape = false
			case c == '\\':
				escape = true
			case c == '"':
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{', '[':
			level++
		case '}', ']':
			level--
			if level == 0 {
				return i
			}
		}
	}
	return -1
}

// jsonIndent reads the file's own indentation so an inserted member lines up
// with its siblings instead of imposing a style.
func jsonIndent(text []byte, open, closeAt int, empty bool) (outer, inner string) {
	lineStart := bytes.LastIndexByte(text[:closeAt], '\n') + 1
	outer = leadingSpace(string(text[lineStart:closeAt]))
	if !empty {
		body := text[open+1 : closeAt]
		for _, line := range splitLines(string(body)) {
			if strings.TrimSpace(line) == "" {
				continue
			}
			inner = leadingSpace(line)
			break
		}
	}
	if inner == "" {
		inner = outer + "  "
	}
	return outer, inner
}

func leadingSpace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// tomlInsert appends `key = value` to its table, creating the table when the
// file has none.
func tomlInsert(text []byte, path []string, lit string) ([]byte, error) {
	table := strings.Join(path[:len(path)-1], ".")
	key := path[len(path)-1]
	line := key + " = " + lit + "\n"
	if table == "" {
		// A root key must precede the first table header.
		lines := splitLines(string(text))
		offset := 0
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "[") {
				break
			}
			offset += len(l)
		}
		return spliceAt(text, offset, line), nil
	}
	start, end, found := tomlTableBody(text, table)
	if !found {
		body := string(text)
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		if strings.TrimSpace(body) != "" {
			body += "\n"
		}
		return []byte(body + "[" + table + "]\n" + line), nil
	}
	_ = start
	return spliceAt(text, end, line), nil
}

// tomlTableBody reports the offsets just after a table's header line and at
// the end of its last non-blank line.
func tomlTableBody(text []byte, table string) (start, end int, ok bool) {
	lines := splitLines(string(text))
	offset, inTable := 0, false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		header := strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
		if header {
			name := strings.TrimSpace(strings.Trim(trimmed, "[]"))
			if inTable {
				return start, end, true
			}
			if name == table {
				inTable = true
				start = offset + len(line)
				end = start
			}
			offset += len(line)
			continue
		}
		offset += len(line)
		if inTable && trimmed != "" {
			end = offset
		}
	}
	return start, end, inTable
}

// yamlInsert appends `key: value` under its parent block, creating the block
// when the document has none.
func yamlInsert(text []byte, path []string, lit string) ([]byte, error) {
	if len(path) == 1 {
		body := string(text)
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		return []byte(body + path[0] + ": " + lit + "\n"), nil
	}
	parent := path[:len(path)-1]
	key := path[len(path)-1]
	end, indent, found := yamlBlockEnd(text, parent)
	if !found {
		body := string(text)
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		block := ""
		for i, part := range parent {
			block += strings.Repeat("  ", i) + part + ":\n"
		}
		block += strings.Repeat("  ", len(parent)) + key + ": " + lit + "\n"
		return []byte(body + block), nil
	}
	return spliceAt(text, end, indent+key+": "+lit+"\n"), nil
}

// yamlBlockEnd reports the offset after the last line of the block the path
// names, and the indentation its members use.
func yamlBlockEnd(text []byte, path []string) (end int, indent string, ok bool) {
	lines := splitLines(string(text))
	offset, depth, parentIndent := 0, 0, -1
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			offset += len(line)
			continue
		}
		lineIndent := len(line) - len(strings.TrimLeft(line, " "))
		if inBlock {
			if lineIndent <= parentIndent {
				return end, indent, true
			}
			if indent == "" {
				indent = strings.Repeat(" ", lineIndent)
			}
			offset += len(line)
			end = offset
			continue
		}
		name, _, found := strings.Cut(trimmed, ":")
		if found && name == path[depth] && lineIndent > parentIndent {
			depth++
			parentIndent = lineIndent
			offset += len(line)
			end = offset
			if depth == len(path) {
				inBlock = true
				indent = ""
			}
			continue
		}
		offset += len(line)
	}
	if inBlock {
		if indent == "" {
			indent = strings.Repeat(" ", parentIndent+2)
		}
		return end, indent, true
	}
	return 0, "", false
}

func spliceAt(text []byte, at int, insert string) []byte {
	out := make([]byte, 0, len(text)+len(insert))
	out = append(out, text[:at]...)
	out = append(out, insert...)
	out = append(out, text[at:]...)
	return out
}

// removeTOMLTable drops `[name]` and every line under it, plus one blank line
// that only separated it from the next table.
func removeTOMLTable(text []byte, name string) ([]byte, bool) {
	lines := splitLines(string(text))
	offset, start, end := 0, -1, -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		header := strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
		if header {
			if start >= 0 {
				end = offset
				break
			}
			if strings.TrimSpace(strings.Trim(trimmed, "[]")) == name {
				start = offset
			}
		}
		offset += len(line)
	}
	if start < 0 {
		return nil, false
	}
	if end < 0 {
		end = offset
	}
	// Take the blank line that preceded the header with it.
	for start > 1 && text[start-1] == '\n' && (start < 2 || text[start-2] == '\n') {
		start--
	}
	return append(append([]byte{}, text[:start]...), text[end:]...), true
}

// removeYAMLBlock drops `name:` and every line indented under it, plus one
// blank line that only separated it from the next key.
func removeYAMLBlock(text []byte, path []string) ([]byte, bool) {
	lines := splitLines(string(text))
	offset, depth, parentIndent := 0, 0, -1
	start, headIndent := -1, 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			offset += len(line)
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if start >= 0 {
			if indent > headIndent {
				offset += len(line)
				continue
			}
			return spliceOut(text, start, offset), true
		}
		name, _, found := strings.Cut(trimmed, ":")
		if found && name == path[depth] && indent > parentIndent {
			if depth == len(path)-1 {
				start, headIndent = offset, indent
				offset += len(line)
				continue
			}
			depth++
			parentIndent = indent
		}
		offset += len(line)
	}
	if start >= 0 {
		return spliceOut(text, start, offset), true
	}
	return nil, false
}

func spliceOut(text []byte, start, end int) []byte {
	return append(append([]byte{}, text[:start]...), text[end:]...)
}
