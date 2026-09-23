package server

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeCodexAppServer is a minimal `codex app-server`: it answers the
// handshake and login/start the way Codex 0.156.0 does (measured), writing
// auth.json — and config.toml for Bedrock — into $CODEX_HOME.
func fakeCodexAppServer() {
	home := os.Getenv("CODEX_HOME")
	out := bufio.NewWriter(os.Stdout)
	send := func(v any) {
		raw, _ := json.Marshal(v)
		out.Write(append(raw, '\n'))
		out.Flush()
	}
	write := func(name, body string) { _ = os.WriteFile(filepath.Join(home, name), []byte(body), 0o600) }
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		var msg struct {
			ID     any            `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
		}
		if json.Unmarshal(sc.Bytes(), &msg) != nil {
			continue
		}
		switch msg.Method {
		case "initialize":
			send(map[string]any{"id": msg.ID, "result": map[string]any{"codexHome": home}})
		case "account/logout":
			_ = os.Remove(filepath.Join(home, "auth.json"))
			send(map[string]any{"id": msg.ID, "result": map[string]any{}})
		case "account/login/start":
			p := msg.Params
			switch p["type"] {
			case "apiKey":
				write("auth.json", `{"auth_mode":"apikey","OPENAI_API_KEY":"`+p["apiKey"].(string)+`"}`)
				send(map[string]any{"id": msg.ID, "result": map[string]any{"type": "apiKey"}})
			case "amazonBedrock":
				if p["apiKey"] == "reject" {
					send(map[string]any{"id": msg.ID, "error": map[string]any{"code": -32600, "message": "invalid Bedrock key"}})
					continue
				}
				write("auth.json", `{"auth_mode":"bedrockApiKey","OPENAI_API_KEY":null,"bedrock_api_key":{"api_key":"`+p["apiKey"].(string)+`","region":"`+p["region"].(string)+`"}}`)
				write("config.toml", "# mine\nmodel = \"gpt-5.5\"\nmodel_provider = \"amazon-bedrock\"\n")
				send(map[string]any{"id": msg.ID, "result": map[string]any{"type": "amazonBedrock"}})
			case "chatgpt", "chatgptDeviceCode":
				res := map[string]any{"type": p["type"], "loginId": "L1"}
				if p["type"] == "chatgpt" {
					res["authUrl"] = "https://auth.openai.com/oauth/authorize?fake=1"
				} else {
					res["verificationUrl"], res["userCode"] = "https://auth.openai.com/codex/device", "ABCD-1234"
				}
				send(map[string]any{"id": msg.ID, "result": res})
				// The device code waits longer, so a second sign-in meets it running.
				if p["type"] == "chatgptDeviceCode" {
					time.Sleep(time.Second)
				} else {
					time.Sleep(150 * time.Millisecond)
				}
				write("auth.json", `{"auth_mode":"chatgpt","tokens":{"id_token":"x","access_token":"a","refresh_token":"r","account_id":"acc"}}`)
				send(map[string]any{"method": "account/login/completed", "params": map[string]any{"loginId": "L1", "success": true}})
			}
		}
	}
}

// installFakeCodex puts a `codex` on PATH that runs the fake above.
func installFakeCodex(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_CODEX=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	home := filepath.Join(t.TempDir(), "codex-home")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", home)
	return home
}

func waitCodexLogin(t *testing.T) map[string]any {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		codexLogin.mu.Lock()
		running, done, errText := codexLogin.running, codexLogin.done, codexLogin.err
		codexLogin.mu.Unlock()
		if !running && done {
			return map[string]any{"error": errText}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the Codex sign-in never finished")
	return nil
}

// Codex's GUI sign-in (ADR-0191), one test per row:
//
//	API key                   → Codex writes auth.json; answered at once
//	Bedrock                   → auth.json + model_provider; the roster shows the platform, no row in use
//	Codex refuses             → its message comes back, nothing marked done-ok
//	ChatGPT / device code     → URL (and code) back at once; done on Codex's notification
//	a login after Bedrock     → model_provider = "amazon-bedrock" removed, the rest of config.toml kept
//	Stop using Bedrock        → Codex's logout, and the line removed
//	a second sign-in at once  → 409
func TestCodexGUILogin(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)
	home := installFakeCodex(t)
	read := func(name string) string {
		raw, _ := os.ReadFile(filepath.Join(home, name))
		return string(raw)
	}

	// Row: API key.
	res := cliRequest(t, ts, "POST", "/api/codex/login", map[string]any{"type": "apiKey", "apiKey": "sk-proj-test-1234567890"}, 200)
	if res["done"] != true || !strings.Contains(read("auth.json"), `"apikey"`) {
		t.Fatalf("apiKey login = %v, auth.json = %s", res, read("auth.json"))
	}

	// Row: Bedrock.
	cliRequest(t, ts, "POST", "/api/codex/login", map[string]any{"type": "amazonBedrock", "apiKey": "ABSK-test", "region": "us-east-1"}, 200)
	if !strings.Contains(read("config.toml"), `model_provider = "amazon-bedrock"`) {
		t.Fatalf("config.toml = %s", read("config.toml"))
	}
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	if p, _ := roster["platform"].(map[string]any); p["name"] != "Amazon Bedrock" || p["region"] != "us-east-1" {
		t.Fatalf("roster platform = %v", roster["platform"])
	}
	if raw, _ := json.Marshal(roster); strings.Contains(string(raw), "ABSK-test") {
		t.Fatal("the roster carries the Bedrock key")
	}
	if add, _ := roster["add"].(map[string]any); add["kind"] != "codex" {
		t.Fatalf("add = %v", roster["add"])
	}

	// Row: Codex refuses.
	bad := cliRequestFull(t, ts, "POST", "/api/codex/login", map[string]any{"type": "amazonBedrock", "apiKey": "reject", "region": "us-east-1"})
	if bad["status"] != "400" || !strings.Contains(string(codexJSON(bad["body"])), "invalid Bedrock key") {
		t.Fatalf("refused login = %v", bad)
	}

	// Row: ChatGPT in the browser — URL now, done on the notification — and
	// the Bedrock line goes with it.
	res = cliRequest(t, ts, "POST", "/api/codex/login", map[string]any{"type": "chatgpt"}, 200)
	if res["done"] != false || !strings.HasPrefix(res["authUrl"].(string), "https://auth.openai.com/") {
		t.Fatalf("chatgpt login = %v", res)
	}
	if st := waitCodexLogin(t); st["error"] != "" {
		t.Fatalf("chatgpt finished with %v", st)
	}
	if cfg := read("config.toml"); strings.Contains(cfg, "model_provider") || !strings.Contains(cfg, `model = "gpt-5.5"`) || !strings.Contains(cfg, "# mine") {
		t.Fatalf("config.toml after a ChatGPT login = %q", cfg)
	}

	// Row: device code.
	res = cliRequest(t, ts, "POST", "/api/codex/login", map[string]any{"type": "chatgptDeviceCode"}, 200)
	if res["userCode"] != "ABCD-1234" || res["verificationUrl"] == nil {
		t.Fatalf("device code = %v", res)
	}
	// Row: a second sign-in while one waits.
	if second := cliRequestFull(t, ts, "POST", "/api/codex/login", map[string]any{"type": "chatgpt"}); second["status"] != "409" {
		t.Fatalf("second sign-in = %v, want 409", second)
	}
	waitCodexLogin(t)

	// Row: Stop using Bedrock.
	cliRequest(t, ts, "POST", "/api/codex/login", map[string]any{"type": "amazonBedrock", "apiKey": "ABSK-2", "region": "eu-west-1"}, 200)
	cliRequest(t, ts, "DELETE", "/api/codex/platform", nil, 200)
	if read("auth.json") != "" || strings.Contains(read("config.toml"), "model_provider") {
		t.Fatalf("after Stop using: auth.json = %q, config.toml = %q", read("auth.json"), read("config.toml"))
	}

	// Refusals before Codex starts.
	for _, body := range []map[string]any{{"type": "mars"}, {"type": "apiKey"}, {"type": "amazonBedrock", "apiKey": "k", "region": "Bad Region"}} {
		if st := cliRequestFull(t, ts, "POST", "/api/codex/login", body); st["status"] != "400" {
			t.Fatalf("POST %v = %v, want 400", body, st)
		}
	}
}

func codexJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}
