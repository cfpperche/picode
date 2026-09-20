package clisettings

import "strings"

// TOML is scanned line by line, which is only safe once the scanner knows
// where strings are. A `"""` block can contain anything — including lines that
// look exactly like `key = value` or `[table]` — and a naive line scan wrote
// into the middle of one, or retargeted a write because a `[table]`-shaped
// line inside a string flipped the current table (adversarial review,
// 2026-09-20).
//
// tomlLine classifies one physical line, with its byte offset, so every TOML
// operation in this package agrees on what is config and what is prose.
type tomlLine struct {
	text     string
	offset   int
	inString bool   // the line begins inside a multi-line string
	header   string // the table this line opens, when it is a header
	isArray  bool   // the header is `[[an array of tables]]`
}

func tomlLines(text []byte) []tomlLine {
	lines := splitLines(string(text))
	out := make([]tomlLine, 0, len(lines))
	offset := 0
	delim := "" // the open multi-line delimiter, "" when outside one
	for _, l := range lines {
		tl := tomlLine{text: l, offset: offset, inString: delim != ""}
		delim = scanTOMLLine(l, delim)
		if !tl.inString {
			trimmed := strings.TrimSpace(l)
			switch {
			case strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]"):
				tl.header = strings.TrimSpace(trimmed[2 : len(trimmed)-2])
				tl.isArray = true
			case strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]"):
				tl.header = strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			}
		}
		out = append(out, tl)
		offset += len(l)
	}
	return out
}

// scanTOMLLine advances the multi-line string state across one line and
// returns the delimiter still open at its end.
func scanTOMLLine(line, delim string) string {
	for i := 0; i < len(line); i++ {
		if delim != "" {
			if strings.HasPrefix(line[i:], delim) {
				i += 2
				delim = ""
			}
			continue
		}
		switch {
		case strings.HasPrefix(line[i:], `"""`):
			delim = `"""`
			i += 2
		case strings.HasPrefix(line[i:], `'''`):
			delim = `'''`
			i += 2
		case line[i] == '"':
			i = skipTOMLBasic(line, i)
		case line[i] == '\'':
			i = skipTOMLLiteral(line, i)
		case line[i] == '#':
			return delim
		}
	}
	return delim
}

// skipTOMLBasic returns the index of the closing quote of a basic string that
// opens at i, or the end of the line when it never closes.
func skipTOMLBasic(line string, i int) int {
	for j := i + 1; j < len(line); j++ {
		if line[j] == '\\' {
			j++
			continue
		}
		if line[j] == '"' {
			return j
		}
	}
	return len(line)
}

// skipTOMLLiteral does the same for a single-quoted literal string, where a
// backslash is an ordinary character. Treating it as an escape swallowed the
// trailing comment of `model = 'gpt\'  # why` (adversarial review).
func skipTOMLLiteral(line string, i int) int {
	if j := strings.IndexByte(line[i+1:], '\''); j >= 0 {
		return i + 1 + j
	}
	return len(line)
}

// tomlArrayTable reports whether the document declares name as an array of
// tables. PiCode does not write into one: its elements are user data the
// schema has no way to address, and the reader reports the key unset, so a
// toggle would edit an arbitrary element forever.
func tomlArrayTable(text []byte, name string) bool {
	for _, l := range tomlLines(text) {
		if l.header == name && l.isArray {
			return true
		}
	}
	return false
}
