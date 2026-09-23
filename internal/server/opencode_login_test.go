package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeOpencode plays `opencode serve` as OpenCode 1.18.32 does (measured):
// the catalog, the plugin login methods, a key written by PUT /auth, a
// "code" OAuth that takes a pasted code and an "auto" one that finishes on
// its own — each writing OpenCode's own auth.json.
func fakeOpencode(args []string) int {
	port := ""
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			port = args[i+1]
		}
	}
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}
	path := filepath.Join(data, "opencode", "auth.json")
	write := func(id string, entry map[string]any) {
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		doc := map[string]any{}
		if raw, err := os.ReadFile(path); err == nil {
			_ = json.Unmarshal(raw, &doc)
		}
		doc[id] = entry
		raw, _ := json.Marshal(doc)
		_ = os.WriteFile(path, raw, 0o600)
	}
	mux := http.NewServeMux()
	// The real server accepts a request that arrives while it boots and never
	// answers it (measured): the first probe hangs, the next one answers.
	var hung atomic.Bool
	mux.HandleFunc("GET /provider/auth", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("PICODE_FAKE_OPENCODE_HANG_FIRST") == "1" && hung.CompareAndSwap(false, true) {
			time.Sleep(time.Minute)
			return
		}
		w.Write([]byte(`{"openai":[{"type":"oauth","label":"ChatGPT Pro/Plus (browser)"},{"type":"oauth","label":"ChatGPT Pro/Plus (headless)"},{"type":"api","label":"Manually enter API Key"}],"cloudflare-workers-ai":[{"type":"api","label":"API key","prompts":[{"type":"text","key":"accountId","message":"Enter your Cloudflare Account ID"}]}]}`))
	})
	mux.HandleFunc("GET /provider", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"all":[{"id":"deepseek","name":"DeepSeek","env":["DEEPSEEK_API_KEY"]},{"id":"openai","name":"OpenAI","env":["OPENAI_API_KEY"]},{"id":"cloudflare-workers-ai","name":"Cloudflare Workers AI","env":["CLOUDFLARE_API_KEY"]}],"connected":[]}`))
	})
	mux.HandleFunc("PUT /auth/{id}", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		write(r.PathValue("id"), body)
		w.Write([]byte("true"))
	})
	mux.HandleFunc("POST /provider/{id}/oauth/authorize", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method int `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Method == 0 {
			w.Write([]byte(`{"url":"https://auth.openai.com/oauth/authorize?fake=1","method":"code","instructions":"Paste the authorization code"}`))
			return
		}
		w.Write([]byte(`{"url":"https://auth.openai.com/codex/device","method":"auto","instructions":"Enter code: TXR3-KWOSA"}`))
	})
	mux.HandleFunc("POST /provider/{id}/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Method int    `json:"method"`
			Code   string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Method == 0 && body.Code != "good" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message":"invalid authorization code"}`))
			return
		}
		if body.Method == 1 {
			time.Sleep(800 * time.Millisecond)
		}
		write(r.PathValue("id"), map[string]any{"type": "oauth", "access": "oc-access", "refresh": "oc-refresh", "expires": 4102444800000})
		w.Write([]byte("true"))
	})
	_ = http.ListenAndServe("127.0.0.1:"+port, mux)
	return 0
}

func installFakeOpencode(t *testing.T) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_OPENCODE=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "opencode"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Cleanup(ocServe.stop)
}

// OpenCode's GUI credentials (ADR-0201), one test per row:
//
//	catalog          → OpenCode's providers with their methods; a provider with no plugin gets the key door
//	key              → PUT /auth through OpenCode (prompts as metadata), filed in the vault
//	"code" OAuth     → the page; a wrong code refused with OpenCode's message; the right one filed
//	"auto" OAuth     → the page and instructions; done when OpenCode finishes; filed
//	a second sign-in → 409 while one waits
func TestOpencodeGUICredentials(t *testing.T) {
	ts, _, home, _ := credentialsServer(t)
	installFakeOpencode(t)
	auth := filepath.Join(home, ".local", "share", "opencode", "auth.json")
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	// Row: a cold start whose first probe hangs still answers in seconds.
	t.Setenv("PICODE_FAKE_OPENCODE_HANG_FIRST", "1")
	start := time.Now()
	cat := cliRequest(t, ts, "GET", "/api/opencode/catalog", nil, 200)
	if took := time.Since(start); took > 15*time.Second {
		t.Fatalf("the catalog took %v: a hung boot probe held it", took)
	}
	var deepseek, openai map[string]any
	for _, p := range cat["providers"].([]any) {
		row := p.(map[string]any)
		switch row["id"] {
		case "deepseek":
			deepseek = row
		case "openai":
			openai = row
		}
	}
	if deepseek == nil || deepseek["methods"].([]any)[0].(map[string]any)["type"] != "api" {
		t.Fatalf("deepseek = %v", deepseek)
	}
	if openai == nil || len(openai["methods"].([]any)) != 3 {
		t.Fatalf("openai = %v", openai)
	}

	// Row: key, with a prompt's answer as metadata.
	cliRequest(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "cloudflare-workers-ai", "method": 0, "type": "api", "key": "cf-key-0123456789", "inputs": map[string]any{"accountId": "acct-42"}}, 200)
	raw, _ := os.ReadFile(auth)
	if !strings.Contains(string(raw), "cf-key-0123456789") || !strings.Contains(string(raw), "acct-42") {
		t.Fatalf("auth.json = %s", raw)
	}
	cliRequest(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "deepseek", "method": -1, "type": "api", "key": "ds-key-0123456789"}, 200)
	if rows := rosterFor(t, ts, "opencode", "deepseek")["accounts"].([]any); len(rows) != 1 {
		t.Fatalf("the DeepSeek key was not filed in the vault: %v", rows)
	}

	// Row: "code" OAuth.
	res := cliRequest(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "openai", "method": 0, "type": "oauth"}, 200)
	if res["mode"] != "code" || !strings.HasPrefix(res["url"].(string), "https://auth.openai.com/") {
		t.Fatalf("code authorize = %v", res)
	}
	if second := cliRequestFull(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "openai", "method": 1, "type": "oauth"}); second["status"] != "409" {
		t.Fatalf("second = %v", second)
	}
	if bad := cliRequestFull(t, ts, "POST", "/api/opencode/credential/code", map[string]any{"code": "wrong"}); bad["status"] != "400" || !strings.Contains(string(codexJSONFor(bad["body"])), "invalid authorization code") {
		t.Fatalf("wrong code = %v", bad)
	}
	res = cliRequest(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "openai", "method": 0, "type": "oauth"}, 200)
	cliRequest(t, ts, "POST", "/api/opencode/credential/code", map[string]any{"code": "good"}, 200)
	if rows := rosterFor(t, ts, "opencode", "openai-codex")["accounts"].([]any); len(rows) == 0 {
		t.Fatal("the ChatGPT sign-in was not filed in the vault under openai-codex")
	}

	// Row: "auto" OAuth.
	res = cliRequest(t, ts, "POST", "/api/opencode/credential", map[string]any{"provider": "openai", "method": 1, "type": "oauth"}, 200)
	if res["mode"] != "auto" || !strings.Contains(res["instructions"].(string), "TXR3-KWOSA") {
		t.Fatalf("auto authorize = %v", res)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := cliRequest(t, ts, "GET", "/api/opencode/credential", nil, 200)
		if st["pending"] == false && st["done"] == true {
			if st["error"] != "" {
				t.Fatalf("auto finished with %v", st)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if roster := cliRequest(t, ts, "GET", "/api/credentials?cli=opencode", nil, 200); roster["add"].(map[string]any)["kind"] != "opencode" {
		t.Fatalf("add = %v", roster["add"])
	}
}

func codexJSONFor(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}
