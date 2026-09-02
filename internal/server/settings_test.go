package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/config"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// A server with the rebind plumbing wired, as main.go does it.
func newBindServer(t *testing.T) (*httptest.Server, *store.Store, *int) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rebinds := 0
	snap := func() PortSnapshot {
		cfg, _ := config.Resolve(st.GetSetting)
		return PortSnapshot{Current: 8445, Configured: cfg.Port.String(), Host: cfg.Host, PublicURL: cfg.PublicURL}
	}
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		Rebind: func() { rebinds++ }, PortSnapshot: snap,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, st, &rebinds
}

func put(t *testing.T, ts *httptest.Server, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPut, ts.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestPublicURLSetting(t *testing.T) {
	t.Setenv("PICODE_HOST", "")
	ts, st, rebinds := newBindServer(t)

	if res, _ := put(t, ts, "/api/server/public-url", `{"url":"http://box:8445"}`); res.StatusCode != 400 {
		t.Fatalf("plain http on a TLS server = %d", res.StatusCode)
	}
	res, out := put(t, ts, "/api/server/public-url", `{"url":"https://Box.tail.ts.net:8445/"}`)
	if res.StatusCode != 200 || out["publicUrl"] != "https://box.tail.ts.net:8445" {
		t.Fatalf("set = %d %v", res.StatusCode, out)
	}
	if v, _, _ := st.GetSetting(config.PublicURLSettingKey); v != "https://box.tail.ts.net:8445" {
		t.Fatalf("stored %q", v)
	}
	if *rebinds != 1 {
		t.Fatalf("rebinds = %d (the loop refreshes server.json)", *rebinds)
	}
	r, err := ts.Client().Get(ts.URL + "/api/server")
	if err != nil {
		t.Fatal(err)
	}
	var info serverInfo
	_ = json.NewDecoder(r.Body).Decode(&info)
	r.Body.Close()
	if info.PublicURL != "https://box.tail.ts.net:8445" || info.Host != config.DefaultHost || info.Interfaces == nil {
		t.Fatalf("info = %+v", info)
	}
	if res, _ := put(t, ts, "/api/server/public-url", `{"url":""}`); res.StatusCode != 200 {
		t.Fatalf("clear = %d", res.StatusCode)
	}
	if v, _, _ := st.GetSetting(config.PublicURLSettingKey); v != "" {
		t.Fatalf("not cleared: %q", v)
	}
}

func TestHostSetting(t *testing.T) {
	t.Setenv("PICODE_HOST", "")
	ts, st, rebinds := newBindServer(t)

	if res, _ := put(t, ts, "/api/server/host", `{"host":"box.local"}`); res.StatusCode != 400 {
		t.Fatalf("name as host = %d", res.StatusCode)
	}
	if res, _ := put(t, ts, "/api/server/host", `{"host":"203.0.113.9"}`); res.StatusCode != 400 {
		t.Fatalf("foreign address = %d", res.StatusCode)
	}
	if res, out := put(t, ts, "/api/server/host", `{"host":"0.0.0.0"}`); res.StatusCode != 200 || out["moving"] != false {
		t.Fatalf("same host = %d %v", res.StatusCode, out)
	}
	res, out := put(t, ts, "/api/server/host", `{"host":"127.0.0.1"}`)
	if res.StatusCode != 202 || out["moving"] != true {
		t.Fatalf("loopback = %d %v", res.StatusCode, out)
	}
	if v, _, _ := st.GetSetting(config.HostSettingKey); v != "127.0.0.1" || *rebinds != 1 {
		t.Fatalf("stored %q rebinds %d", v, *rebinds)
	}
}
