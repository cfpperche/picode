package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestCheckPromptImages(t *testing.T) {
	ok := promptImage{MimeType: "image/png", Data: "aaa"}
	rows := []struct {
		name string
		in   []promptImage
		err  string
	}{
		{"none", nil, ""},
		{"one", []promptImage{ok}, ""},
		{"five", []promptImage{ok, ok, ok, ok, ok}, "at most 4"},
		{"bad mime", []promptImage{{MimeType: "application/pdf", Data: "x"}}, "unsupported"},
		{"empty data", []promptImage{{MimeType: "image/png"}}, "required"},
		{"too big", []promptImage{{MimeType: "image/png", Data: strings.Repeat("a", maxImageB64+1)}}, "4 MB"},
	}
	for _, r := range rows {
		err := checkPromptImages(r.in)
		if r.err == "" {
			if err != nil {
				t.Fatalf("%s: %v", r.name, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), r.err) {
			t.Fatalf("%s: %v want %s", r.name, err, r.err)
		}
	}
}

func TestAgentPromptHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	proj := t.TempDir()
	if err := os.WriteFile(proj+"/a.txt", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	res := postJSON(t, ts, "/api/workspaces", map[string]string{"name": "App", "path": proj})
	var wk workspaceView
	_ = json.NewDecoder(res.Body).Decode(&wk)
	id := wk.Agent.ID

	body, _ := json.Marshal(map[string]any{
		"kind":    "prompt",
		"message": "see",
		"images":  []map[string]string{{"mimeType": "image/png", "data": "aaa"}},
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+id+"/prompt", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got := do(t, ts.Client(), req)
	if got.StatusCode != http.StatusConflict {
		t.Fatalf("stopped = %d want 409", got.StatusCode)
	}

	five := make([]map[string]string, 5)
	for i := range five {
		five[i] = map[string]string{"mimeType": "image/png", "data": "aaa"}
	}
	body, _ = json.Marshal(map[string]any{"message": "x", "images": five})
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/agents/"+id+"/prompt", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	got = do(t, ts.Client(), req)
	if got.StatusCode != http.StatusBadRequest {
		t.Fatalf("five = %d", got.StatusCode)
	}

	none := do(t, ts.Client(), mustGet(t, ts.URL+"/api/agents/ag_missing/prompt"))
	// GET not registered — method not allowed or 404. POST missing agent:
	req, _ = http.NewRequest(http.MethodPost, ts.URL+"/api/agents/ag_missing/prompt", bytes.NewReader([]byte(`{"message":"hi"}`)))
	req.Header.Set("Content-Type", "application/json")
	none = do(t, ts.Client(), req)
	if none.StatusCode != http.StatusNotFound {
		t.Fatalf("missing = %d", none.StatusCode)
	}
}
