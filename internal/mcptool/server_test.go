package mcptool

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeDaemon records the last call and answers a fixed status and body.
type fakeDaemon struct {
	status   int
	body     string
	path     string
	got      map[string]any
	auth     string
	gets     []string
	getCalls int
}

func (f *fakeDaemon) Post(_ context.Context, path string, body []byte) (int, []byte, error) {
	f.path = path
	_ = json.Unmarshal(body, &f.got)
	return f.status, []byte(f.body), nil
}

// Get answers from gets (a queue, one per call) so a poll can see an item
// change; with no queue it answers the fixed body.
func (f *fakeDaemon) Get(_ context.Context, path string) (int, []byte, error) {
	f.path = path
	f.getCalls++
	if len(f.gets) > 0 {
		next := f.gets[0]
		f.gets = f.gets[1:]
		return 200, []byte(next), nil
	}
	return f.status, []byte(f.body), nil
}

func rpc(t *testing.T, s *Server, line string) map[string]any {
	t.Helper()
	out := s.Handle(context.Background(), []byte(line))
	if out == nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("bad response %s: %v", out, err)
	}
	return m
}

func TestServerSpeaksTheProtocol(t *testing.T) {
	d := &fakeDaemon{status: 200, body: `{"action":"windows","output":{}}`}
	s := NewServer("picode-computer", "0.3.1", []Family{computerFamily, browserFamily}, &Caller{Daemon: d, Identity: Identity{Term: "t1"}})

	init := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"claude-code","version":"2.0"}}}`)
	res := init["result"].(map[string]any)
	if res["protocolVersion"] != "2025-03-26" {
		t.Fatalf("a known revision is echoed: %v", res)
	}
	if res["serverInfo"].(map[string]any)["name"] != "picode-computer" || !strings.Contains(res["instructions"].(string), "computer tool:") {
		t.Fatalf("serverInfo/instructions = %v", res)
	}
	if got := rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2099-01-01"}}`)["result"].(map[string]any)["protocolVersion"]; got != protocolVersions[0] {
		t.Fatalf("an unknown revision gets the newest we know, got %v", got)
	}
	if out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)); out != nil {
		t.Fatalf("a notification gets no answer, got %s", out)
	}
	if got := rpc(t, s, `{"jsonrpc":"2.0","id":3,"method":"ping"}`); got["result"] == nil || got["error"] != nil {
		t.Fatalf("ping = %v", got)
	}
	list := rpc(t, s, `{"jsonrpc":"2.0","id":4,"method":"tools/list"}`)["result"].(map[string]any)["tools"].([]any)
	if len(list) != 2 || list[0].(map[string]any)["name"] != "browser" || list[1].(map[string]any)["name"] != "computer" {
		t.Fatalf("tools = %v", list)
	}
	schema := list[1].(map[string]any)["inputSchema"].(map[string]any)
	if schema["type"] != "object" || len(schema["properties"].(map[string]any)["action"].(map[string]any)["enum"].([]any)) != 23 {
		t.Fatalf("computer schema = %v", schema)
	}
	if got := rpc(t, s, `{"jsonrpc":"2.0","id":5,"method":"resources/list"}`); got["error"].(map[string]any)["code"].(float64) != codeMethodMissing {
		t.Fatalf("unknown method = %v", got)
	}
	if got := rpc(t, s, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"nope","arguments":{}}}`); got["error"].(map[string]any)["code"].(float64) != codeInvalidParams {
		t.Fatalf("unknown tool = %v", got)
	}
	if got := rpc(t, s, `{"jsonrpc":"2.0","id":7,"method":"tools/call"}`); got["error"] == nil {
		t.Fatalf("call without a name = %v", got)
	}
	if got := rpc(t, s, `not json`); got["error"].(map[string]any)["code"].(float64) != codeParse {
		t.Fatalf("parse error = %v", got)
	}
	if out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","id":9,"result":{}}`)); out != nil {
		t.Fatalf("a response from the client is ignored, got %s", out)
	}
}

func TestServerCallReachesTheDaemonWithTheIdentity(t *testing.T) {
	d := &fakeDaemon{status: 200, body: `{"action":"screenshot","output":{"ok":true,"image":"iVBORw0KGgo=","mime":"image/png","meta":{"display":1,"width":10,"height":5,"scale":1,"cursor":[1,2]}}}`}
	s := NewServer("picode", "x", []Family{computerFamily}, &Caller{Daemon: d, Identity: Identity{Term: "desktop-1"}})
	got := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"computer","arguments":{"action":"screenshot","display":1}}}`)
	if d.path != "/api/computer/tool" || d.got["term"] != "desktop-1" || d.got["agent"] != "" || d.got["action"] != "screenshot" {
		t.Fatalf("wire = %s %v", d.path, d.got)
	}
	if params := d.got["params"].(map[string]any); params["display"] != float64(1) || params["action"] != nil {
		t.Fatalf("params = %v", params)
	}
	res := got["result"].(map[string]any)
	content := res["content"].([]any)
	if res["isError"] != nil || len(content) != 2 || content[1].(map[string]any)["type"] != "image" || content[1].(map[string]any)["mimeType"] != "image/png" {
		t.Fatalf("result = %v", res)
	}
	if text := content[0].(map[string]any)["text"].(string); text != "Screenshot of display 1 (10×5 image, scale 1) — coordinates you send are pixels of this image; cursor at [1, 2]." {
		t.Fatalf("text = %s", text)
	}

	// A refusal is a result the model reads, never a JSON-RPC error.
	d.status, d.body = 403, `{"error":"this agent may not use the computer — turn it on in Settings ▸ Computer"}`
	got = rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"computer","arguments":{"action":"left_click","coordinate":[1,1]}}}`)
	res = got["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), "Settings ▸ Computer") {
		t.Fatalf("refusal = %v", got)
	}
	// An unknown action never reaches the wire.
	d.path = ""
	got = rpc(t, s, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"computer","arguments":{"action":"fly"}}}`)
	if d.path != "" || got["result"].(map[string]any)["isError"] != true {
		t.Fatalf("unknown action = %v (path %q)", got, d.path)
	}
}

func TestServerWithoutIdentityOrDaemonAnswersInWords(t *testing.T) {
	d := &fakeDaemon{status: 200, body: `{}`}
	s := NewServer("picode", "x", []Family{browserFamily}, &Caller{Daemon: d})
	got := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"browser","arguments":{"verb":"snapshot"}}}`)
	res := got["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), NoIdentity) || d.path != "" {
		t.Fatalf("no identity = %v (path %q)", got, d.path)
	}
	s = NewServer("picode", "x", []Family{browserFamily}, &Caller{Daemon: d, Identity: Identity{Agent: "a1"}, Unreachable: "no server.json"})
	got = rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"browser","arguments":{"verb":"snapshot"}}}`)
	if text := got["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"].(string); !strings.Contains(text, "not reachable (no server.json)") {
		t.Fatalf("unreachable = %s", text)
	}
}

func TestServerBrowserCallCarriesTheVerbParams(t *testing.T) {
	d := &fakeDaemon{status: 200, body: `{"verb":"cdp","output":{"cookies":[]}}`}
	s := NewServer("picode", "x", []Family{browserFamily}, &Caller{Daemon: d, Identity: Identity{Agent: "a1"}})
	got := rpc(t, s, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"browser","arguments":{"verb":"cdp","method":"Network.getAllCookies","params":"{\"urls\":true}"}}}`)
	if d.path != "/api/browser/tool" || d.got["agent"] != "a1" || d.got["verb"] != "cdp" {
		t.Fatalf("wire = %v", d.got)
	}
	params := d.got["params"].(map[string]any)
	if params["method"] != "Network.getAllCookies" || params["params"].(map[string]any)["urls"] != true {
		t.Fatalf("params = %v", params)
	}
	if text := got["result"].(map[string]any)["content"].([]any)[0].(map[string]any)["text"]; text != `{"cookies":[]}` {
		t.Fatalf("cdp text = %v", text)
	}
	got = rpc(t, s, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"browser","arguments":{"verb":"cdp","params":"{"}}}`)
	if got["result"].(map[string]any)["isError"] != true {
		t.Fatal("bad cdp params must be refused before the wire")
	}
}

func TestServeIsLineOrientedAndStopsAtEOF(t *testing.T) {
	s := NewServer("picode", "x", []Family{computerFamily}, &Caller{Daemon: &fakeDaemon{}, Identity: Identity{Term: "t"}})
	in := strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\n\n{\"jsonrpc\":\"2.0\",\"method\":\"notifications/initialized\"}\n{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/list\"}")
	var out bytes.Buffer
	if err := s.Serve(context.Background(), in, &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"id":1`) || !strings.Contains(lines[1], `"tools"`) {
		t.Fatalf("out =\n%s", out.String())
	}
}

func TestHTTPDaemonSendsTheBearerAndAcceptsLoopbackTLS(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"error":"nope"}`))
	}))
	defer srv.Close()
	d := NewHTTPDaemon(srv.URL, func() string { return "abc" })
	status, body, err := d.Post(context.Background(), "/api/computer/tool", []byte(`{}`))
	if err != nil || status != 403 || string(body) != `{"error":"nope"}` || gotAuth != "Bearer abc" || gotPath != "/api/computer/tool" {
		t.Fatalf("post = %d %s %v (auth %q path %q)", status, body, err, gotAuth, gotPath)
	}
	if _, err := ParseAnswer(status, body, "action"); err == nil || err.Error() != "nope" {
		t.Fatalf("refusal must carry the daemon's words, got %v", err)
	}
}

func TestDiscoveryMirrorsThePiPackages(t *testing.T) {
	env := MapEnv(map[string]string{"PICODE_URL": "https://box:8445/"})
	if u, err := ResolveURL(env, t.TempDir()); err != nil || u != "https://box:8445" {
		t.Fatalf("PICODE_URL = %q %v", u, err)
	}
	if _, err := ResolveURL(MapEnv(map[string]string{"PICODE_URL": "https://user:pass@box/"}), ""); err == nil {
		t.Fatal("credentials in the origin are refused")
	}
	if u, err := ResolveURL(MapEnv(map[string]string{"PICODE_TERM_URL": "https://127.0.0.1:8445"}), ""); err != nil || u != "https://127.0.0.1:8445" {
		t.Fatalf("PICODE_TERM_URL = %q %v", u, err)
	}
	dir := t.TempDir()
	if _, err := ResolveURL(MapEnv(nil), dir); err == nil || err.Error() != "no server.json" {
		t.Fatalf("missing server.json = %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "server.json"), []byte(`{"url":"https://localhost:8445/"}`), 0o600)
	if u, err := ResolveURL(MapEnv(nil), dir); err != nil || u != "https://localhost:8445" {
		t.Fatalf("server.json = %q %v", u, err)
	}
	if _, err := ParseServerJSON([]byte(`{"url":"ftp://x"}`)); err == nil {
		t.Fatal("ftp is not a usable url")
	}
	if _, err := ParseServerJSON([]byte(`{`)); err == nil {
		t.Fatal("bad json")
	}
	if got := ReadToken(MapEnv(map[string]string{"PICODE_TOKEN": strings.Repeat("a", 32)}), dir); got != strings.Repeat("a", 32) {
		t.Fatalf("env token = %q", got)
	}
	_ = os.WriteFile(filepath.Join(dir, "token"), []byte(strings.Repeat("b", 64)+"\n"), 0o600)
	if got := ReadToken(MapEnv(map[string]string{"PICODE_TOKEN": "short"}), dir); got != strings.Repeat("b", 64) {
		t.Fatalf("file token = %q", got)
	}
	if got := ReadToken(MapEnv(nil), t.TempDir()); got != "" {
		t.Fatalf("no token = %q", got)
	}
	if DataDir(MapEnv(map[string]string{"PICODE_DATA": "/d"}), "/home/x") != "/d" || DataDir(MapEnv(nil), "/home/x/") != "/home/x/.picode" {
		t.Fatal("data dir")
	}
	for url, want := range map[string]bool{"https://localhost:8445": false, "https://127.0.0.1:8445": false, "https://[::1]:8445": false, "https://box:8445": true, "not a url": true} {
		if RejectUnauthorizedFor(url) != want {
			t.Errorf("RejectUnauthorizedFor(%s) != %v", url, want)
		}
	}
	id := IdentityFrom(MapEnv(map[string]string{"PICODE_TERM_ID": " t1 "}))
	if id.Principal() != "term:t1" || IdentityFrom(MapEnv(map[string]string{"PICODE_AGENT_ID": "a", "PICODE_TERM_ID": "t"})).Principal() != "a" || !IdentityFrom(MapEnv(nil)).Empty() {
		t.Fatal("identity rule")
	}
}
