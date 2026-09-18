package connectors

import (
	"bytes"
	"encoding/json"
	"errors"

	"github.com/cfpperche/picode/internal/mcp"
)

// parseError marks a config file that exists but does not parse (any
// codec: JSON, TOML, YAML). List degrades the owning layer to a blocked
// state instead of failing the whole pane; every write path still refuses
// on it — corruption over silence.
type parseError struct {
	path   string
	reason string // "is not valid JSON" and friends, shown as-is
}

func (e *parseError) Error() string { return e.path + " " + e.reason }

// blockLayer degrades one layer whose file could not be read: the layer
// reports exists plus a short reason and contributes no servers. A
// malformed user file must never blank the pane (ADR-0150).
func blockLayer(layer *mcp.Layer, err error) {
	layer.Exists = true
	var pe *parseError
	if errors.As(err, &pe) {
		layer.Error = pe.reason
	} else {
		layer.Error = "could not be read"
	}
}

// lenientJSON is the read-side second chance: strict JSON failed, so try
// again after stripping the JSONC escapes vendors hand-write (OpenCode
// leaves trailing commas in opencode.json). Anything the strip cannot make
// parseable is still refused as parseError by the caller.
func lenientJSON(b []byte) (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal(stripJSONC(b), &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// stripJSONC removes // and /* */ comments and a comma directly before } or
// ]. String literals are copied verbatim — a URL keeps its // — and
// everything else passes through untouched, so a document the vendor would
// also refuse stays refused.
func stripJSONC(b []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(b))
	inStr, esc := false, false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if inStr {
			out.WriteByte(c)
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
			out.WriteByte(c)
		case '/':
			if i+1 < len(b) && b[i+1] == '/' {
				for i < len(b) && b[i] != '\n' {
					i++
				}
				out.WriteByte('\n')
			} else if i+1 < len(b) && b[i+1] == '*' {
				i += 2
				for i+1 < len(b) && !(b[i] == '*' && b[i+1] == '/') {
					i++
				}
				i++ // land on the closer; the loop's i++ moves past it
				out.WriteByte(' ')
			} else {
				out.WriteByte(c)
			}
		case ',':
			j := i + 1
			for j < len(b) && (b[j] == ' ' || b[j] == '\t' || b[j] == '\r' || b[j] == '\n') {
				j++
			}
			if j < len(b) && b[j] != '}' && b[j] != ']' {
				out.WriteByte(c)
			}
		default:
			out.WriteByte(c)
		}
	}
	return out.Bytes()
}
