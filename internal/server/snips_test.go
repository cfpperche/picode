package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func snipJSON(t *testing.T, tsURL, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, tsURL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res := do(t, http.DefaultClient, req)
	defer res.Body.Close()
	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestSnipsHTTP(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, out := snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{
		"title": "Review PR", "body": "Look at {{pr}}",
	})
	if code != http.StatusOK {
		t.Fatalf("create = %d %v", code, out)
	}
	id := out["id"].(string)
	if out["slug"] != "review-pr" {
		t.Fatalf("slug = %v", out["slug"])
	}

	code, list := snipJSON(t, ts.URL, http.MethodGet, "/api/snips", nil)
	if code != http.StatusOK {
		t.Fatalf("list = %d", code)
	}
	raw, _ := json.Marshal(list)
	if strings.Contains(string(raw), `"body":`) {
		t.Fatalf("list body: %s", raw)
	}

	code, pick := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatalf("picker = %d %v", code, pick)
	}
	snips, _ := pick["snips"].([]any)
	if len(snips) != 1 {
		t.Fatalf("picker rows = %v", pick)
	}

	code, got := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/"+id, nil)
	if code != http.StatusOK || got["body"] != "Look at {{pr}}" {
		t.Fatalf("get = %d %v", code, got)
	}

	stale := out["updatedAt"].(string)
	code, _ = snipJSON(t, ts.URL, http.MethodPatch, "/api/snips/"+id, map[string]any{
		"title": "Review PR", "body": "v2", "ifUpdatedAt": stale,
	})
	if code != http.StatusOK {
		t.Fatalf("patch = %d", code)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodPatch, "/api/snips/"+id, map[string]any{
		"title": "Review PR", "body": "v3", "ifUpdatedAt": stale,
	})
	if code != http.StatusConflict {
		t.Fatalf("stale patch = %d", code)
	}

	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips", map[string]any{"title": "Review PR", "body": "x"})
	if code != http.StatusBadRequest {
		t.Fatalf("dup slug = %d", code)
	}

	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/starred", map[string]any{"starred": true})
	if code != http.StatusOK {
		t.Fatalf("star = %d", code)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodPost, "/api/snips/"+id+"/archived", map[string]any{"archived": true})
	if code != http.StatusOK {
		t.Fatalf("archive = %d", code)
	}
	code, pick = snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatal(code)
	}
	if rows, _ := pick["snips"].([]any); len(rows) != 0 {
		t.Fatalf("archived still in picker: %v", pick)
	}

	req, _ := http.NewRequest(http.MethodDelete, ts.URL+"/api/snips/"+id, nil)
	res := do(t, http.DefaultClient, req)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d", res.StatusCode)
	}
	code, _ = snipJSON(t, ts.URL, http.MethodGet, "/api/snips/"+id, nil)
	if code != http.StatusNotFound {
		t.Fatalf("get deleted = %d", code)
	}
}

func TestSnipsPickerIsNotAnID(t *testing.T) {
	ts, _, _ := cleanupServer(t)
	code, _ := snipJSON(t, ts.URL, http.MethodGet, "/api/snips/picker", nil)
	if code != http.StatusOK {
		t.Fatalf("picker treated as id: %d", code)
	}
}
