package clisettings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Format names one config document shape. A CLI declares the format of each
// file it owns; the reader uses the real parser and the writer splices the
// exact bytes of one scalar, so comments, key order and every untouched key
// survive a save (ADR-0163).
type Format string

const (
	FormatJSON  Format = "json"
	FormatJSONC Format = "jsonc"
	FormatTOML  Format = "toml"
	FormatYAML  Format = "yaml"
)

// parseError carries the file a parser rejected. A caller turns it into the
// pane's "this file is not readable" state; a write on top of it is refused
// rather than replacing a document PiCode did not understand (ADR-0099 §4).
type parseError struct {
	path   string
	reason string
}

func (e *parseError) Error() string { return e.path + ": " + e.reason }

// IsParseError reports whether err came from a rejected config document.
func IsParseError(err error) bool {
	var pe *parseError
	return errors.As(err, &pe)
}

// decode parses a whole document into a generic tree. Reading uses the real
// parser for the format; only writing is textual.
func decode(text []byte, f Format, path string) (map[string]any, error) {
	if len(bytes.TrimSpace(text)) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	var err error
	switch f {
	case FormatJSON, FormatJSONC:
		err = json.Unmarshal(stripJSONC(text), &out)
	case FormatTOML:
		err = toml.Unmarshal(text, &out)
	case FormatYAML:
		var node any
		if err = yaml.Unmarshal(text, &node); err == nil {
			m, ok := node.(map[string]any)
			if !ok {
				if node == nil {
					return map[string]any{}, nil
				}
				return nil, &parseError{path, "does not hold a settings mapping"}
			}
			out = m
		}
	default:
		return nil, fmt.Errorf("unknown config format %q", f)
	}
	if err != nil {
		return nil, &parseError{path, err.Error()}
	}
	return out, nil
}

// lookup walks a dotted path through a decoded document.
func lookup(doc map[string]any, path []string) (any, bool) {
	var cur any = doc
	for _, part := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// stripJSONC blanks comments in place so byte offsets stay aligned with the
// original text: the JSON scanner that locates a value span runs over this
// buffer and its offsets index the real file.
func stripJSONC(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	inString, escape := false, false
	for i := 0; i < len(out); i++ {
		c := out[i]
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
		switch {
		case c == '"':
			inString = true
		case c == '/' && i+1 < len(out) && out[i+1] == '/':
			for i < len(out) && out[i] != '\n' {
				out[i] = ' '
				i++
			}
		case c == '/' && i+1 < len(out) && out[i+1] == '*':
			for i < len(out) && !(out[i] == '*' && i+1 < len(out) && out[i+1] == '/') {
				if out[i] != '\n' {
					out[i] = ' '
				}
				i++
			}
			if i < len(out) {
				out[i] = ' '
			}
			if i+1 < len(out) {
				out[i+1] = ' '
			}
			i++
		}
	}
	return out
}

// literal renders one scalar in the document's own syntax. Only scalars are
// written: a field whose value is a list or an object is not in any CLI's
// declared schema (ADR-0163 keeps the schema to scalars precisely so a write
// is always one token).
func literal(f Format, v any) (string, error) {
	switch value := v.(type) {
	case bool:
		return strconv.FormatBool(value), nil
	case float64:
		if value == float64(int64(value)) {
			return strconv.FormatInt(int64(value), 10), nil
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	case int:
		return strconv.Itoa(value), nil
	case string:
		switch f {
		case FormatYAML:
			// yaml.Marshal of a lone string yields "value\n"; trim it and keep
			// the quoting it chose (a string needing quotes keeps them).
			b, err := yaml.Marshal(value)
			if err != nil {
				return "", err
			}
			return strings.TrimRight(string(b), "\n"), nil
		default:
			b, err := json.Marshal(value)
			if err != nil {
				return "", err
			}
			return string(b), nil
		}
	default:
		return "", fmt.Errorf("only scalar settings are written, got %T", v)
	}
}

// valueSpan locates the byte span of the value assigned to a dotted path, so
// a write replaces exactly those bytes and nothing else.
func valueSpan(text []byte, f Format, path []string) (start, end int, ok bool) {
	switch f {
	case FormatJSON, FormatJSONC:
		return jsonValueSpan(text, path)
	case FormatTOML:
		return tomlValueSpan(text, path)
	case FormatYAML:
		return yamlValueSpan(text, path)
	}
	return 0, 0, false
}

// jsonValueSpan walks the token stream and reports the offsets of the value
// under path. Offsets come from the decoder itself, so they are exact whatever
// the file's whitespace, and comments were blanked to keep them aligned.
//
// frames mirrors the open objects and arrays; keys carries the current key of
// each frame. A path matches only when every open frame is an object, so a
// value inside an array is never addressed by a dotted key.
func jsonValueSpan(text []byte, path []string) (int, int, bool) {
	scan := stripJSONC(text)
	dec := json.NewDecoder(bytes.NewReader(scan))
	type frame struct {
		object    bool
		expectKey bool
	}
	var frames []frame
	var keys []string
	matches := func() bool {
		if len(frames) != len(path) {
			return false
		}
		for i := range frames {
			if !frames[i].object || keys[i] != path[i] {
				return false
			}
		}
		return true
	}
	// valueSeen records that a value was consumed at the current level, so an
	// object frame expects a key again.
	valueSeen := func() {
		if n := len(frames); n > 0 && frames[n-1].object {
			frames[n-1].expectKey = true
		}
	}
	for {
		before := dec.InputOffset()
		tok, err := dec.Token()
		if err != nil {
			return 0, 0, false
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				// A composite in value position: report it before descending,
				// because its span is the whole brace pair.
				if matches() {
					open := valueStartAt(scan, int(before))
					if closeAt := matchBrace(scan, open); closeAt >= 0 {
						return open, closeAt + 1, true
					}
					return 0, 0, false
				}
				frames = append(frames, frame{object: d == '{', expectKey: d == '{'})
				keys = append(keys, "")
				continue
			default:
				if len(frames) > 0 {
					frames = frames[:len(frames)-1]
					keys = keys[:len(keys)-1]
				}
				valueSeen()
				continue
			}
		}
		if n := len(frames); n > 0 && frames[n-1].object && frames[n-1].expectKey {
			key, isString := tok.(string)
			if !isString {
				return 0, 0, false
			}
			keys[n-1] = key
			frames[n-1].expectKey = false
			continue
		}
		if matches() {
			return valueStartAt(scan, int(before)), int(dec.InputOffset()), true
		}
		valueSeen()
	}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// valueStartAt moves from where the decoder left the previous token to the
// first byte of the value itself, stepping over the separator the grammar puts
// between them (`:` after a key, `,` between array elements) and any spacing.
func valueStartAt(scan []byte, at int) int {
	for at < len(scan) {
		c := scan[at]
		if isSpace(c) || c == ':' || c == ',' {
			at++
			continue
		}
		break
	}
	return at
}

// tomlValueSpan finds `key = value` inside the table named by the path's
// prefix. TOML is line-oriented, so the span ends before a trailing comment:
// a user's `# why` note beside a value survives the write.
func tomlValueSpan(text []byte, path []string) (int, int, bool) {
	if len(path) == 0 {
		return 0, 0, false
	}
	table := strings.Join(path[:len(path)-1], ".")
	key := path[len(path)-1]
	lines := splitLines(string(text))
	current := ""
	offset := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			current = strings.TrimSpace(strings.Trim(trimmed, "[]"))
			offset += len(line)
			continue
		}
		if current == table && !strings.HasPrefix(trimmed, "#") {
			if name, rest, found := strings.Cut(line, "="); found && strings.TrimSpace(name) == key {
				valueStart := offset + len(name) + 1
				lead := len(rest) - len(strings.TrimLeft(rest, " \t"))
				valueStart += lead
				value := rest[lead:]
				value = strings.TrimRight(value, "\r\n")
				value = trimTrailingComment(value)
				return valueStart, valueStart + len(strings.TrimRight(value, " \t")), true
			}
		}
		offset += len(line)
	}
	return 0, 0, false
}

// trimTrailingComment drops a ` # …` tail that sits outside a quoted string.
func trimTrailingComment(value string) string {
	inString := false
	var quote byte
	for i := 0; i < len(value); i++ {
		c := value[i]
		switch {
		case inString && c == '\\':
			i++
		case inString && c == quote:
			inString = false
		case !inString && (c == '"' || c == '\''):
			inString, quote = true, c
		case !inString && c == '#':
			return value[:i]
		}
	}
	return value
}

// yamlValueSpan walks indented blocks to the key the path names. Only inline
// scalars are located; a key whose value is a block is reported as absent so
// the writer refuses instead of mangling it.
func yamlValueSpan(text []byte, path []string) (int, int, bool) {
	lines := splitLines(string(text))
	offset := 0
	depth := 0
	parentIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			offset += len(line)
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent <= parentIndent && depth > 0 {
			// Left the block we were descending into without a match.
			return 0, 0, false
		}
		name, rest, found := strings.Cut(trimmed, ":")
		if !found {
			offset += len(line)
			continue
		}
		if name != path[depth] {
			offset += len(line)
			continue
		}
		if depth == len(path)-1 {
			value := strings.TrimRight(rest, "\r\n")
			value = trimTrailingComment(value)
			lead := len(value) - len(strings.TrimLeft(value, " \t"))
			if strings.TrimSpace(value) == "" {
				return 0, 0, false
			}
			valueStart := offset + (len(line) - len(strings.TrimLeft(line, " "))) + len(name) + 1 + lead
			return valueStart, valueStart + len(strings.TrimRight(value[lead:], " \t")), true
		}
		depth++
		parentIndent = indent
		offset += len(line)
	}
	return 0, 0, false
}

// splitLines keeps the line terminators so offsets stay exact.
func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i+1])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
