package toolpreview

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"
)

func fixture(t *testing.T, format string, w, h int) string {
	t.Helper()
	var b bytes.Buffer
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	var err error
	if format == "jpeg" {
		err = jpeg.Encode(&b, im, nil)
	} else {
		err = png.Encode(&b, im)
	}
	if err != nil {
		t.Fatal(err)
	}
	return "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}

func TestCaptureDecisionTable(t *testing.T) {
	pngURI := fixture(t, "png", 20, 10)
	for _, tc := range []struct {
		name  string
		raw   any
		valid bool
	}{
		{"PNG", map[string]any{"image": pngURI}, true},
		{"JPEG", map[string]any{"image": fixture(t, "jpeg", 20, 10)}, true},
		{"remote", map[string]any{"image": "https://example.com/x.png"}, false},
		{"relative", map[string]any{"image": "/api/file"}, false},
		{"svg", map[string]any{"image": "data:image/svg+xml;base64,PHN2Zy8+"}, false},
		{"blob", map[string]any{"image": "blob:x"}, false},
		{"type", map[string]any{"image": 42}, false},
		{"malformed", map[string]any{"image": "data:image/png;base64,AAAA"}, false},
		{"metadata-type", "hostile", false},
		{"bytes", map[string]any{"image": "data:image/png;base64," + strings.Repeat("A", MaxBytes*2)}, false},
		{"width", map[string]any{"image": fixture(t, "png", 1601, 1)}, false},
		{"pixels", map[string]any{"image": fixture(t, "png", 1600, 1600)}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			details := map[string]any{"preview": tc.raw, "verified": false}
			out := Details(details)
			p := out["preview"].(map[string]any)
			if (p["image"] != nil) != tc.valid {
				t.Fatalf("valid=%v preview=%v", tc.valid, p)
			}
			if out["verified"] != false {
				t.Fatal("producer metadata changed")
			}
			if details["preview"] == nil {
				t.Fatal("mutated input")
			}
		})
	}
	for _, d := range []map[string]any{nil, {}, {"preview": nil}} {
		if out := Details(d); out["preview"] != nil {
			t.Fatal("invented unavailable state")
		}
	}
}

func TestMetadataPrivacy(t *testing.T) {
	d := Details(map[string]any{"preview": map[string]any{
		"image": fixture(t, "png", 1, 1), "url": "https://user:secret@example.com/page?token=secret#private",
		"title": strings.Repeat("x", 1000), "ts": float64(123), "source": "session/tab",
	}})["preview"].(map[string]any)
	if d["url"] != "https://example.com/page" || len(d["title"].(string)) != 160 || d["ts"] != float64(123) || d["source"] != "session/tab" {
		t.Fatalf("metadata: %v", d)
	}
}

func TestResultEnvelopes(t *testing.T) {
	result := map[string]any{"details": map[string]any{"preview": map[string]any{"image": "https://example.com/private"}, "artifactVerification": "kept"}}
	for _, tc := range []struct {
		kind, field string
		list        bool
	}{
		{"tool_execution_update", "partialResult", false}, {"tool_execution_end", "result", false},
		{"message_start", "message", false}, {"message_end", "message", false},
		{"turn_end", "toolResults", true}, {"agent_end", "messages", true},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			e := map[string]any{"type": tc.kind}
			if tc.list {
				e[tc.field] = []any{result}
			} else {
				e[tc.field] = result
			}
			raw, _ := json.Marshal(e)
			out := Event(raw)
			if bytes.Contains(out, []byte("https://")) || !bytes.Contains(out, []byte("unavailable")) || !bytes.Contains(out, []byte("artifactVerification")) {
				t.Fatalf("unsafe result: %s", out)
			}
		})
	}
	escaped := Event([]byte(`{"type":"tool_execution_update","partialResult":{"details":{"pre\u0076iew":{"image":"https://example.com"}}}}`))
	if bytes.Contains(escaped, []byte("https://")) || !bytes.Contains(escaped, []byte("unavailable")) {
		t.Fatalf("escaped key bypass: %s", escaped)
	}
	ordinary := []byte(`{"type":"agent_settled"}`)
	if !bytes.Equal(Event(ordinary), ordinary) {
		t.Fatal("changed ordinary event")
	}
}
