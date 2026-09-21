package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/pipkg"
)

// The unified read answers for any CLI with a driver, and refuses one without
// by naming what exists (ADR-0176).
func TestPackageReportAnswersEveryDriver(t *testing.T) {
	userDir := t.TempDir()
	old := pipkg.UserDir
	pipkg.UserDir = func() string { return userDir }
	t.Cleanup(func() { pipkg.UserDir = old })
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "settings.json"), []byte(`{"packages":["npm:pi-web-search"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, AgentCmd: "cat"}).Handler)
	t.Cleanup(ts.Close)

	res, err := http.Get(ts.URL + "/api/packages/report")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d %s", res.StatusCode, body)
	}
	for _, want := range []string{`"cli":"pi"`, `"catalog":"gallery"`, `"install":true`, `"config":true`, `"id":"agent"`, `pi-web-search`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("report %s lacks %s", body, want)
		}
	}

	res, err = http.Get(ts.URL + "/api/packages/report?cli=nope")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ = io.ReadAll(res.Body)
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "omp") {
		t.Fatalf("unknown CLI = %d %s, want 400 naming the drivers", res.StatusCode, body)
	}
}
