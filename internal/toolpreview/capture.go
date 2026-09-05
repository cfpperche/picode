// Package toolpreview bounds optional capture metadata before web delivery.
// It does not write Pi's session files or capture any pixels (ADR-0076).
package toolpreview

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"net/url"
	"strings"
	"unicode"
)

const MaxBytes = 200 * 1024
const MaxSide = 1600
const MaxPixels = 1600000

// Details returns a copy only when a preview was present. All unrelated fields
// survive, including error/verification metadata owned by the producing tool.
func Details(details map[string]any) map[string]any {
	raw, exists := details["preview"]
	if !exists || raw == nil {
		return details
	}
	out := make(map[string]any, len(details))
	for k, v := range details {
		out[k] = v
	}
	p, ok := raw.(map[string]any)
	if !ok {
		out["preview"] = map[string]any{"unavailable": true}
		return out
	}
	normalized, ok := capture(p)
	if !ok {
		normalized = map[string]any{"unavailable": true}
	}
	out["preview"] = normalized
	return out
}

func capture(p map[string]any) (map[string]any, bool) {
	s, _ := p["image"].(string)
	if len(s) > base64.StdEncoding.EncodedLen(MaxBytes)+32 {
		return nil, false
	}
	var decode func(*bytes.Reader) (image.Config, error)
	var encoded string
	switch {
	case strings.HasPrefix(s, "data:image/png;base64,"):
		encoded = strings.TrimPrefix(s, "data:image/png;base64,")
		decode = func(r *bytes.Reader) (image.Config, error) { return png.DecodeConfig(r) }
	case strings.HasPrefix(s, "data:image/jpeg;base64,"):
		encoded = strings.TrimPrefix(s, "data:image/jpeg;base64,")
		decode = func(r *bytes.Reader) (image.Config, error) { return jpeg.DecodeConfig(r) }
	default:
		return nil, false
	}
	if strings.ContainsAny(encoded, "\r\n") {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(b) == 0 || len(b) > MaxBytes {
		return nil, false
	}
	cfg, err := decode(bytes.NewReader(b))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxSide || cfg.Height > MaxSide || cfg.Width*cfg.Height > MaxPixels {
		return nil, false
	}
	out := map[string]any{"image": s, "title": label(p["title"], 160), "url": captionURL(p["url"])}
	if source := label(p["source"], 120); source != "" {
		out["source"] = source
	}
	if ts, ok := p["ts"].(float64); ok && ts > 0 && ts <= 8640000000000000 && ts == float64(int64(ts)) {
		out["ts"] = ts
	}
	return out, true
}

func label(raw any, limit int) string {
	s, _ := raw.(string)
	r := []rune(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s))
	if len(r) > limit {
		r = r[:limit]
	}
	return string(r)
}

func captionURL(raw any) string {
	s, _ := raw.(string)
	if len(s) > 4096 {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	u.User, u.RawQuery, u.Fragment, u.RawFragment, u.ForceQuery = nil, "", "", "", false
	return label(u.String(), 512)
}

// Event bounds all known result-bearing RPC envelopes, including duplicated
// final tool messages. Decode keys before testing them: escaped JSON keys
// must not bypass the projection (for example pre\\u0076iew).
func Event(raw []byte) []byte {
	var event map[string]any
	if json.Unmarshal(raw, &event) != nil {
		return raw
	}
	normalizeResult := func(raw any) {
		if result, ok := raw.(map[string]any); ok {
			if d, ok := result["details"].(map[string]any); ok {
				result["details"] = Details(d)
			}
		}
	}
	switch event["type"] {
	case "tool_execution_update":
		normalizeResult(event["partialResult"])
	case "tool_execution_end":
		normalizeResult(event["result"])
	case "message_start", "message_end", "turn_end":
		normalizeResult(event["message"])
		if results, ok := event["toolResults"].([]any); ok {
			for _, r := range results {
				normalizeResult(r)
			}
		}
	case "agent_end":
		if messages, ok := event["messages"].([]any); ok {
			for _, m := range messages {
				normalizeResult(m)
			}
		}
	default:
		return raw
	}
	b, err := json.Marshal(event)
	if err != nil {
		return raw
	}
	return b
}
