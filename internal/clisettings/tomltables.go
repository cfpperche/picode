package clisettings

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Arrays of tables are user data the settings schema never addresses (see
// tomlArrayTable), with one exception: an element PiCode writes whole and
// finds again by its own fields. Codex keeps its per-skill switch that way —
// `[[skills.config]] path = "…/SKILL.md" enabled = false` (ADR-0196 slice 3,
// measured on codex 0.156.1) — so the toggle appends one element and removes
// the elements it matches, and never edits one in place.

// ArrayTables returns the decoded elements of `[[name]]`, in file order.
func (d Doc) ArrayTables(name string) []map[string]any {
	if d.format != FormatTOML {
		return nil
	}
	v, ok := lookup(d.doc, strings.Split(name, "."))
	if !ok {
		return nil
	}
	var out []map[string]any
	switch list := v.(type) {
	case []any:
		for _, e := range list {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
	case []map[string]any:
		out = list
	}
	return out
}

// AppendArrayTable adds one `[[name]]` element at the end of the document.
// Fields are written in the order given; values are scalars.
func (d *Doc) AppendArrayTable(name string, fields [][2]any) error {
	if d.format != FormatTOML {
		return errors.New("arrays of tables are TOML")
	}
	var b strings.Builder
	text := string(d.text)
	if text != "" && !strings.HasSuffix(text, "\n") {
		b.WriteString("\n")
	}
	if strings.TrimSpace(text) != "" {
		b.WriteString("\n")
	}
	b.WriteString("[[" + name + "]]\n")
	for _, f := range fields {
		key, _ := f[0].(string)
		lit, err := literal(FormatTOML, f[1])
		if err != nil {
			return err
		}
		b.WriteString(key + " = " + lit + "\n")
	}
	d.text = append(append([]byte{}, d.text...), b.String()...)
	return d.reparse()
}

// RemoveArrayTables drops every `[[name]]` element match accepts and reports
// how many went. An element runs from its header to the next header; blank
// and comment lines just above that next header stay with it, since they
// introduce it.
func (d *Doc) RemoveArrayTables(name string, match func(map[string]any) bool) (int, error) {
	if d.format != FormatTOML {
		return 0, errors.New("arrays of tables are TOML")
	}
	lines := tomlLines(d.text)
	type span struct{ start, end int }
	var drop []span
	for i := 0; i < len(lines); i++ {
		if !lines[i].isArray || lines[i].header != name {
			continue
		}
		j := i + 1
		for j < len(lines) && lines[j].header == "" {
			j++
		}
		// Keep the comments and blank lines that sit above the next header.
		k := j
		for k > i+1 {
			t := strings.TrimSpace(lines[k-1].text)
			if lines[k-1].inString || (t != "" && !strings.HasPrefix(t, "#")) {
				break
			}
			k--
		}
		end := len(d.text)
		if k < len(lines) {
			end = lines[k].offset
		}
		body := string(d.text[lines[i].offset+len(lines[i].text) : end])
		elem := map[string]any{}
		if err := toml.Unmarshal([]byte(body), &elem); err != nil {
			return 0, fmt.Errorf("[[%s]] element on line %d: %w", name, i+1, err)
		}
		// An element PiCode matches holds scalars only. A table or an array
		// of tables in the span means a header was not recognised and the
		// span ran past it; nothing is removed rather than too much.
		for k, v := range elem {
			switch v.(type) {
			case map[string]any, []map[string]any:
				return 0, fmt.Errorf("[[%s]] element on line %d holds %s, a table PiCode does not remove", name, i+1, k)
			}
		}
		if match(elem) {
			start := lines[i].offset
			// The blank line AppendArrayTable put before the element goes with it.
			if start >= 2 && d.text[start-1] == '\n' && d.text[start-2] == '\n' {
				start--
			}
			// At the top of the file there is nothing for a blank line to
			// separate: it goes with the element (a comment stays).
			if start == 0 {
				for end < len(d.text) && (d.text[end] == '\n' || d.text[end] == '\r') {
					end++
				}
			}
			drop = append(drop, span{start, end})
		}
		i = j - 1
	}
	if len(drop) == 0 {
		return 0, nil
	}
	out := make([]byte, 0, len(d.text))
	last := 0
	for _, s := range drop {
		out = append(out, d.text[last:s.start]...)
		last = s.end
	}
	out = append(out, d.text[last:]...)
	if strings.TrimSpace(string(out)) == "" {
		out = nil
	}
	d.text = out
	return len(drop), d.reparse()
}
