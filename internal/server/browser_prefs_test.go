package server

// The destinations side of the prefs: the PUT round-trips both keys and
// refuses a value outside app/external.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/store"
)

func prefsServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
}

func TestPrefsDestinationsRoundTrip(t *testing.T) {
	ts := prefsServer(t)
	put, _ := json.Marshal(map[string]any{"webOpenDest": "external", "localOpenDest": "app"})
	resp, err := http.NewRequest(http.MethodPut, ts.URL+"/api/browser/prefs", bytes.NewReader(put))
	if err != nil {
		t.Fatal(err)
	}
	resp.Header.Set("Content-Type", "application/json")
	do, err := http.DefaultClient.Do(resp)
	if err != nil {
		t.Fatal(err)
	}
	defer do.Body.Close()
	if do.StatusCode != http.StatusOK {
		t.Fatalf("PUT status %d", do.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(do.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["webOpenDest"] != "external" || body["localOpenDest"] != "app" {
		t.Fatalf("destinations did not round trip: %v", body)
	}
	bad, _ := json.Marshal(map[string]any{"webOpenDest": "printer"})
	badResp, err := http.DefaultClient.Do(mustRequest(http.MethodPut, ts.URL+"/api/browser/prefs", bad))
	if err != nil {
		t.Fatal(err)
	}
	badResp.Body.Close()
	if badResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("a bogus destination must be a 400, got %d", badResp.StatusCode)
	}
}

func mustRequest(method, url string, body []byte) *http.Request {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
