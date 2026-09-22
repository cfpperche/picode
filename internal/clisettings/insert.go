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
	// Find the deepest existing ancestor that is an object. A parent that
	// exists as a string, an array or null is not one to add a member to:
	// falling back to the root appended a duplicate key that shadowed the
	// user's value (adversarial review, 2026-09-20).
	depth := len(path) - 1
	for depth > 0 {
		if _, _, ok := jsonObjectSpan(text, path[:depth]); ok {
			break
		}
		if _, _, exists := jsonValueSpan(text, path[:depth]); exists {
			return nil, fmt.Errorf("%s is not an object in this file, so %s cannot be added under it", strings.Join(path[:depth], "."), path[len(path)-1])
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
	at := jsonLastMemberEnd(text, open, closeAt)
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
// file has none. Every scan skips multi-line string bodies: a `[table]`-shaped
// line inside one used to retarget the write.
func tomlInsert(text []byte, path []string, lit string) ([]byte, error) {
	table := strings.Join(path[:len(path)-1], ".")
	key := path[len(path)-1]
	line := key + " = " + lit + "\n"
	if table == "" {
		// A root key must precede the first table header.
		offset := 0
		for _, l := range tomlLines(text) {
			if !l.inString && l.header != "" {
				break
			}
			offset = l.offset + len(l.text)
		}
		return spliceNewline(text, offset, line), nil
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
	return spliceNewline(text, end, line), nil
}

// spliceNewline inserts at an offset, terminating the previous line first when
// the file did not end with a newline. Splicing straight onto an unterminated
// last line produced `use_memories = truegenerate_memories = false`, which the
// post-write check then refused — a file with no final newline could never
// take a new key (adversarial review, 2026-09-20).
func spliceNewline(text []byte, at int, insert string) []byte {
	if at > 0 && at <= len(text) && text[at-1] != '\n' {
		insert = "\n" + insert
	}
	return spliceAt(text, at, insert)
}

// tomlTableBody reports the offsets just after a table's header line and at
// the end of its last non-blank line.
func tomlTableBody(text []byte, table string) (start, end int, ok bool) {
	inTable := false
	for _, l := range tomlLines(text) {
		if l.inString {
			if inTable {
				end = l.offset + len(l.text)
			}
			continue
		}
		if l.header != "" {
			if inTable {
				return start, end, true
			}
			if l.header == table && !l.isArray {
				inTable = true
				start = l.offset + len(l.text)
				end = start
			}
			continue
		}
		if inTable && strings.TrimSpace(l.text) != "" {
			end = l.offset + len(l.text)
		}
	}
	return start, end, inTable
}

// removeTOMLTable drops `[name]` and every line under it, stopping at the last
// real key so a comment that follows the table keeps its place. Consuming the
// blank and `#` lines after it deleted notes that belonged to whatever came
// next (adversarial review, 2026-09-20).
func removeTOMLTable(text []byte, name string) ([]byte, bool) {
	start, end := -1, -1
	for _, l := range tomlLines(text) {
		if !l.inString && l.header != "" {
			if start >= 0 {
				break
			}
			if l.header == name && !l.isArray {
				start = l.offset
				end = l.offset + len(l.text)
			}
			continue
		}
		if start >= 0 && (l.inString || strings.TrimSpace(l.text) != "") && !strings.HasPrefix(strings.TrimSpace(l.text), "#") {
			end = l.offset + len(l.text)
		}
	}
	if start < 0 {
		return nil, false
	}
	// Take one blank line before the header, the exact inverse of the blank
	// line tomlInsert adds when it creates a table in a non-empty file.
	if start >= 2 && text[start-1] == '\n' && text[start-2] == '\n' {
		start--
	}
	return spliceOut(text, start, end), true
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
	// The deepest prefix of the parent that the document already has is where
	// the missing tail is created. Creating the whole chain from the root when
	// only the *middle* was missing appended a second `modelTags:` block beside
	// the one the file had — a duplicate key the parser refuses, so the save
	// was refused with the file untouched (found writing a three-level key,
	// 2026-09-22). A two-level key never hit it: its parent either exists or
	// does not.
	for depth := len(parent); depth >= 1; depth-- {
		end, indent, found := yamlBlockEnd(text, parent[:depth])
		if !found {
			continue
		}
		block := ""
		ind := indent
		for _, part := range parent[depth:] {
			block += ind + part + ":\n"
			ind += "  "
		}
		block += ind + key + ": " + lit + "\n"
		return spliceNewline(text, end, block), nil
	}
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

// yamlBlockEnd reports the offset after the last line of the block the path
// names, and the indentation its members use. It walks with yamlWalk, so a
// same-named block nested elsewhere is never the one an insert lands in.
func yamlBlockEnd(text []byte, path []string) (end int, indent string, ok bool) {
	lineStart, line, parentIndent, found := yamlWalk(text, path)
	if !found {
		return 0, "", false
	}
	end = lineStart + len(line)
	offset := end
	for _, l := range splitLines(string(text[end:])) {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			offset += len(l)
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " \t"))
		if ind <= parentIndent {
			break
		}
		if indent == "" {
			indent = strings.Repeat(" ", ind)
		}
		offset += len(l)
		end = offset
	}
	if indent == "" {
		indent = strings.Repeat(" ", parentIndent+2)
	}
	return end, indent, true
}

// removeYAMLBlock drops `name:` and every line indented under it.
func removeYAMLBlock(text []byte, path []string) ([]byte, bool) {
	lineStart, line, headIndent, found := yamlWalk(text, path)
	if !found {
		return nil, false
	}
	end := lineStart + len(line)
	offset := end
	for _, l := range splitLines(string(text[end:])) {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			// A blank line or a comment may belong to whatever comes next, so
			// it is not swallowed: consuming them emptied a whole file whose
			// block was followed by commented-out config (adversarial review).
			offset += len(l)
			continue
		}
		if ind := len(l) - len(strings.TrimLeft(l, " \t")); ind <= headIndent {
			break
		}
		offset += len(l)
		end = offset
	}
	return spliceOut(text, lineStart, end), true
}

func spliceAt(text []byte, at int, insert string) []byte {
	out := make([]byte, 0, len(text)+len(insert))
	out = append(out, text[:at]...)
	out = append(out, insert...)
	out = append(out, text[at:]...)
	return out
}

func spliceOut(text []byte, start, end int) []byte {
	return append(append([]byte{}, text[:start]...), text[end:]...)
}

// jsonLastMemberEnd reports the offset just after the last member's value in
// the object that spans open..closeAt. Anchoring on the last non-space byte
// put the comma inside a trailing `// comment`, so a JSONC file with a note
// before its closing brace could never take a new key (adversarial review,
// 2026-09-20).
func jsonLastMemberEnd(text []byte, open, closeAt int) int {
	scan := stripJSONC(text)
	dec := json.NewDecoder(bytes.NewReader(scan[open : closeAt+1]))
	depth, expectKey, last := 0, false, 0
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				if depth == 1 && !expectKey {
					if end := matchBrace(scan, open+int(dec.InputOffset())-1); end >= 0 {
						last = end + 1
					}
				}
				depth++
				if d == '{' {
					expectKey = true
				}
				continue
			default:
				depth--
				if depth == 1 {
					last = open + int(dec.InputOffset())
					expectKey = true
				}
				continue
			}
		}
		if depth == 1 {
			if expectKey {
				expectKey = false
			} else {
				last = open + int(dec.InputOffset())
				expectKey = true
			}
		}
	}
	if last <= open {
		last = closeAt
		for last > open+1 && isSpace(text[last-1]) {
			last--
		}
	}
	return last
}
