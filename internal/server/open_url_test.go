package server

// Tests for the terminal browser hand-off (ADR-0180): the wrappers, the
// wiring row, and /api/terminals/{id}/open-url's client-vs-host routing.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/store"
)

func TestOpenURLWrappersDefaultOnAndOptOut(t *testing.T) {
	dataDir := t.TempDir()
	if !interceptOn(dataDir, OpenURLID) {
		t.Fatal("browser hand-off must default on")
	}
	if env := openURLEnv(dataDir); len(env) != 0 {
		t.Fatalf("no wrappers yet, env must be empty, got %v", env)
	}
	ensureOpenURLWrappers(dataDir)
	for _, name := range openURLWrappers {
		body, err := os.ReadFile(wrapperPath(dataDir, name))
		if err != nil {
			t.Fatalf("%s not written: %v", name, err)
		}
		for _, want := range []string{"ADR-0180", "name=" + name, dataDir + "/token", "/api/terminals/$PICODE_TERM_ID/open-url"} {
			if !strings.Contains(string(body), want) {
				t.Fatalf("%s wrapper missing %q:\n%s", name, want, body)
			}
		}
		if st, _ := os.Stat(wrapperPath(dataDir, name)); st.Mode()&0o111 == 0 {
			t.Fatalf("%s not executable", name)
		}
	}
	if env := openURLEnv(dataDir); len(env) != 1 || env[0] != "BROWSER="+wrapperPath(dataDir, "picode-open") {
		t.Fatalf("openURLEnv = %v", env)
	}

	if err := uninstallOpenURL(dataDir); err != nil {
		t.Fatal(err)
	}
	if interceptOn(dataDir, OpenURLID) {
		t.Fatal("explicit opt-out lost")
	}
	for _, name := range openURLWrappers {
		if _, err := os.Stat(wrapperPath(dataDir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still present after uninstall", name)
		}
	}
	// The opt-out persists: ensure must not re-arm the wrappers.
	ensureOpenURLWrappers(dataDir)
	if _, err := os.Stat(wrapperPath(dataDir, "picode-open")); !os.IsNotExist(err) {
		t.Fatal("opted-out wrapper came back")
	}
	if env := openURLEnv(dataDir); len(env) != 0 {
		t.Fatalf("openURLEnv after opt-out = %v", env)
	}

	// Re-enable through the wiring verb.
	if err := installOpenURL(dataDir); err != nil {
		t.Fatal(err)
	}
	if !interceptOn(dataDir, OpenURLID) {
		t.Fatal("re-enable lost")
	}
	if _, err := os.Stat(wrapperPath(dataDir, "xdg-open")); err != nil {
		t.Fatalf("xdg-open shadow not reinstalled: %v", err)
	}
}

// The wrapper's decision table, run against the real script with a fake
// xdg-open standing in for the system: http(s) forwards to the daemon,
// anything else falls through to the real opener.
func TestOpenURLWrapperRunsTheTable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("posix sh only")
	}
	dataDir := t.TempDir()
	ensureOpenURLWrappers(dataDir)
	realDir := t.TempDir()
	recorder := filepath.Join(realDir, "opened.txt")
	real := "#!/bin/sh\nprintf '%s\\n' \"$1\" >> '" + recorder + "'\n"
	if err := os.WriteFile(filepath.Join(realDir, "xdg-open"), []byte(real), 0o755); err != nil {
		t.Fatal(err)
	}
	env := []string{"PATH=" + filepath.Join(dataDir, "bin") + ":" + realDir + ":/usr/bin:/bin"}

	// No PICODE_TERM_URL: the shadow must pass the URL to the real xdg-open.
	cmd := exec.Command(wrapperPath(dataDir, "xdg-open"), "https://example.com/plain")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("shadow passthrough: %v: %s", err, out)
	}
	raw, err := os.ReadFile(recorder)
	if err != nil || string(raw) != "https://example.com/plain\n" {
		t.Fatalf("real xdg-open got %q (%v), want the URL", raw, err)
	}

	// A non-URL argument must neither reach the daemon nor error out.
	cmd = exec.Command(wrapperPath(dataDir, "picode-open"), "/tmp/a folder")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("non-URL run: %v: %s", err, out)
	}
}

func urlFixture(t *testing.T) (*store.Store, string) {
	t.Helper()
	st := testStore(t)
	term, err := st.CreateTerminalIn("", "QA", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return st, term.ID
}

func postOpenURL(t *testing.T, ts *httptest.Server, termID, body string) (int, map[string]any) {
	t.Helper()
	res, err := http.Post(ts.URL+"/api/terminals/"+termID+"/open-url", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestOpenURLDeliveredToAClient(t *testing.T) {
	st, termID := urlFixture(t)
	f := &feed.Feed{}
	seen := make(chan store.Event, 4)
	f.Listen(func(ev store.Event) { seen <- ev })
	_, _, unsub, err := f.Subscribe(0)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	called := ""
	old := openURLFn
	openURLFn = func(u string) error { called = u; return nil }
	defer func() { openURLFn = old }()

	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st, Feed: f}).Handler)
	defer ts.Close()
	code, out := postOpenURL(t, ts, termID, `{"url":"  https://auth.example.com/login?client_id=x&state=y  "}`)
	if code != http.StatusOK || out["delivered"] != "client" {
		t.Fatalf("status=%d out=%v", code, out)
	}
	ev := <-seen
	if ev.Type != "terminal.open_url" {
		t.Fatalf("event = %q", ev.Type)
	}
	var data struct {
		TermID string `json:"termId"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal([]byte(ev.Data), &data); err != nil {
		t.Fatal(err)
	}
	if data.TermID != termID || data.URL != "https://auth.example.com/login?client_id=x&state=y" {
		t.Fatalf("event data = %+v", data)
	}
	if called != "" {
		t.Fatalf("host opener must not run when a client took it: %q", called)
	}
}

func TestOpenURLFallsBackToTheHost(t *testing.T) {
	st, termID := urlFixture(t)
	called := ""
	old := openURLFn
	openURLFn = func(u string) error { called = u; return nil }
	defer func() { openURLFn = old }()

	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	defer ts.Close()
	code, out := postOpenURL(t, ts, termID, `{"url":"https://example.com/ok"}`)
	if code != http.StatusOK || out["delivered"] != "host" {
		t.Fatalf("status=%d out=%v", code, out)
	}
	if called != "https://example.com/ok" {
		t.Fatalf("host opener got %q", called)
	}
}

func TestOpenURLRejectsBadTargets(t *testing.T) {
	st, termID := urlFixture(t)
	called := 0
	old := openURLFn
	openURLFn = func(u string) error { called++; return nil }
	defer func() { openURLFn = old }()
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{Store: st}).Handler)
	defer ts.Close()
	for _, body := range []string{
		`{"url":""}`,
		`{"url":"file:///etc/passwd"}`,
		`{"url":"javascript:alert(1)"}`,
		`{"url":"https://has a space/x"}`,
		`{"url":123}`,
		`not json`,
	} {
		if code, _ := postOpenURL(t, ts, termID, body); code != http.StatusBadRequest {
			t.Fatalf("body %s: status = %d, want 400", body, code)
		}
	}
	if code, _ := postOpenURL(t, ts, "term_missing", `{"url":"https://example.com"}`); code != http.StatusNotFound {
		t.Fatalf("unknown terminal: status = %d, want 404", code)
	}
	if called != 0 {
		t.Fatalf("host opener ran %d times for refused targets", called)
	}
}
