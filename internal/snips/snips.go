// Package snips parses and expands PiCode Snippets (ADR-0130).
//
// This is not internal/snippet, which runs a conversation source fence.
// Expand is pure one-pass substitution: no shell, no eval, no rescan of values.
package snips

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxName         = 40
	MaxPlaceholders = 32
	MaxEnum         = 32
	MaxEnumLen      = 80
	MaxDefault      = 500
	MaxSlug         = 64
)

// Reserved names are filled from Expand's ctx, never from values.
var Reserved = map[string]bool{
	"cwd":       true,
	"workspace": true,
	"branch":    true,
	"date":      true,
	"agent":     true,
	"cli":       true,
}

// Placeholder is one unique {{name}} in a body. First occurrence wins
// the default. Enum is stored by the editor, not parsed from the body.
type Placeholder struct {
	Name     string   `json:"name"`
	Default  string   `json:"default,omitempty"`
	Optional bool     `json:"optional,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

// ParseError is a malformed body. Offset is a byte index.
type ParseError struct {
	Offset int
	Token  string
	Msg    string
}

func (e *ParseError) Error() string {
	if e == nil {
		return "snips: parse error"
	}
	if e.Offset > 0 || e.Token != "" {
		return fmt.Sprintf("snips: %s at %d", e.Msg, e.Offset)
	}
	return "snips: " + e.Msg
}

func parseErr(off int, token, msg string) error {
	return &ParseError{Offset: off, Token: token, Msg: msg}
}

func hasAt(s string, i int, pre string) bool {
	return i+len(pre) <= len(s) && s[i:i+len(pre)] == pre
}

func skipWS(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

func isNameStart(r rune) bool {
	return r == '_' || unicode.IsLetter(r) && r <= unicode.MaxASCII && (r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
}

func isNameCont(r rune) bool {
	return isNameStart(r) || r >= '0' && r <= '9'
}

// readPlaceholder parses a {{ ... }} starting at i. It unescapes {{{{ / }}}}
// only inside the default so the stored default is the literal value.
func readPlaceholder(body string, i int) (Placeholder, int, error) {
	start := i
	if !hasAt(body, i, "{{") {
		return Placeholder{}, i, parseErr(i, "", "not a placeholder")
	}
	i += 2
	i = skipWS(body, i)
	if i >= len(body) {
		return Placeholder{}, start, parseErr(start, "{{", "unclosed placeholder")
	}
	name, next, err := readName(body, i)
	if err != nil {
		return Placeholder{}, start, err
	}
	i = skipWS(body, next)
	ph := Placeholder{Name: name}
	if i < len(body) && body[i] == '=' {
		ph.Optional = true
		i++
		def, end, err := readDefault(body, i)
		if err != nil {
			return Placeholder{}, start, err
		}
		if utf8.RuneCountInString(def) > MaxDefault {
			return Placeholder{}, start, parseErr(start, name, "default too long")
		}
		ph.Default = def
		i = end
	}
	i = skipWS(body, i)
	if !hasAt(body, i, "}}") {
		tok := "{{" + name
		return Placeholder{}, start, parseErr(start, tok, "unclosed placeholder")
	}
	return ph, i + 2, nil
}

func readName(body string, i int) (string, int, error) {
	if i >= len(body) {
		return "", i, parseErr(i, "", "missing placeholder name")
	}
	r, w := utf8.DecodeRuneInString(body[i:])
	if !isNameStart(r) {
		return "", i, parseErr(i, body[i:min(i+8, len(body))], "invalid placeholder name")
	}
	j := i + w
	for j < len(body) {
		r, w = utf8.DecodeRuneInString(body[j:])
		if !isNameCont(r) {
			break
		}
		j += w
	}
	name := body[i:j]
	if len(name) > MaxName {
		return "", i, parseErr(i, name[:MaxName], "placeholder name too long")
	}
	return name, j, nil
}

func readDefault(body string, i int) (string, int, error) {
	var b strings.Builder
	start := i
	for i < len(body) {
		if hasAt(body, i, "}}}}") {
			b.WriteString("}}")
			i += 4
			continue
		}
		if hasAt(body, i, "{{{{") {
			b.WriteString("{{")
			i += 4
			continue
		}
		if hasAt(body, i, "}}") {
			return strings.TrimSpace(b.String()), i, nil
		}
		b.WriteByte(body[i])
		i++
	}
	return "", start, parseErr(start, "", "unclosed placeholder")
}

func lookup(ph Placeholder, values, ctx map[string]string) (string, bool) {
	if Reserved[ph.Name] {
		if ph.Name == "date" {
			if ctx != nil {
				if v, ok := ctx["date"]; ok {
					return v, true
				}
			}
			return time.Now().UTC().Format("2006-01-02"), true
		}
		if ctx != nil {
			return ctx[ph.Name], true
		}
		return "", true
	}
	if values != nil {
		if v, ok := values[ph.Name]; ok && v != "" {
			return v, true
		}
	}
	if ph.Optional {
		return ph.Default, true
	}
	return "", false
}
