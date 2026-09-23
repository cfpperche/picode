package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeHermes plays `hermes auth add` as Hermes 0.21.4 does (measured): a key
// read from stdin goes last in the provider's pool (access_token); an OAuth
// sign-in prints a portal line, the page and a code, waits, then stores the
// provider's tokens.
func fakeHermes(args []string) int {
	home := os.Getenv("HERMES_HOME")
	if home == "" {
		home = filepath.Join(os.Getenv("HOME"), ".hermes")
	}
	_ = os.MkdirAll(home, 0o700)
	path := filepath.Join(home, "auth.json")
	doc := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &doc)
	}
	save := func() { raw, _ := json.Marshal(doc); _ = os.WriteFile(path, raw, 0o600) }
	if len(args) < 4 || args[0] != "auth" || args[1] != "add" {
		return 2
	}
	provider, kind := args[2], args[4]
	switch kind {
	case "api-key":
		if os.Getenv("PICODE_FAKE_HERMES_ARGV") != "" {
			for _, a := range args {
				if strings.HasPrefix(a, "sk-") {
					fmt.Println("key leaked into argv")
					return 3
				}
			}
		}
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		key := strings.TrimSpace(line)
		if key == "reject" {
			fmt.Println("Error: invalid API key")
			return 1
		}
		pool, _ := doc["credential_pool"].(map[string]any)
		if pool == nil {
			pool = map[string]any{}
		}
		list, _ := pool[provider].([]any)
		list = append(list, map[string]any{"auth_type": "api_key", "label": fmt.Sprintf("api-key-%d", len(list)+1), "priority": len(list), "access_token": key})
		pool[provider] = list
		doc["credential_pool"] = pool
		save()
		fmt.Printf("Added %s credential #%d\n", provider, len(list))
		return 0
	case "oauth":
		fmt.Println("Starting Hermes login via Nous Portal...\nPortal: https://portal.nousresearch.com\nTo continue:\n  1. Open: https://portal.nousresearch.com/manage-subscription?user_code=Q48J-EVXF\n  2. If prompted, enter code: Q48J-EVXF\nWaiting for approval (polling every 1s)...")
		time.Sleep(time.Second)
		providers, _ := doc["providers"].(map[string]any)
		if providers == nil {
			providers = map[string]any{}
		}
		providers[provider] = map[string]any{"tokens": map[string]any{"access_token": "nous-access", "refresh_token": "nous-refresh", "account_id": "acct-1"}}
		doc["providers"] = providers
		save()
		return 0
	}
	return 2
}

func installFakeHermes(t *testing.T) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_HERMES=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "hermes"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HERMES_HOME", "")
}

const hermesServerCatalog = `[
 {"id":"nous","name":"Nous Portal","auth_type":"oauth_device_code","env":[]},
 {"id":"deepseek","name":"DeepSeek","auth_type":"api_key","env":["DEEPSEEK_API_KEY"]}
]`

// Hermes's GUI credentials (ADR-0193), one test per row:
//
//	roster                → Hermes's catalog appended; pi's picker door; OAuth as a device code
//	key                   → through stdin (never argv), last in the pool, filed in the vault
//	a second key          → after the first
//	Hermes refuses a key  → its last line comes back
//	OAuth                 → the page and the code (not the portal line); done on exit 0; filed in the vault
//	a second sign-in      → 409 while one waits
//	unknown provider/kind → refused before Hermes runs
func TestHermesGUICredentials(t *testing.T) {
	cat := filepath.Join(t.TempDir(), "hermes.json")
	if err := os.WriteFile(cat, []byte(hermesServerCatalog), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PICODE_HERMES_CATALOG", cat)
	t.Setenv("PICODE_FAKE_HERMES_ARGV", "1")
	ts, _, home, _ := credentialsServer(t)
	installFakeHermes(t)
	pool := func() map[string]any {
		raw, _ := os.ReadFile(filepath.Join(home, ".hermes", "auth.json"))
		var d map[string]any
		_ = json.Unmarshal(raw, &d)
		return d
	}

	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=hermes", nil, 200)
	if add, _ := roster["add"].(map[string]any); add["kind"] != "provider" {
		t.Fatalf("add = %v", roster["add"])
	}
	if p := rosterFor(t, ts, "hermes", "nous"); p["signin"] != "device" || p["name"] != "Nous Portal" {
		t.Fatalf("nous = %v", p)
	}
	if p := rosterFor(t, ts, "hermes", "anthropic"); p["signin"] != nil {
		t.Fatalf("anthropic offers a Hermes sign-in Hermes does not have: %v", p["signin"])
	}

	// Rows: key, second key, refusal.
	cliRequest(t, ts, "POST", "/api/hermes/credential", map[string]any{"provider": "deepseek", "type": "api_key", "key": "sk-first-0123456789"}, 200)
	cliRequest(t, ts, "POST", "/api/hermes/credential", map[string]any{"provider": "deepseek", "type": "api_key", "key": "sk-second-0123456789"}, 200)
	list := pool()["credential_pool"].(map[string]any)["deepseek"].([]any)
	if len(list) != 2 || list[1].(map[string]any)["access_token"] != "sk-second-0123456789" {
		t.Fatalf("pool = %v", list)
	}
	if rows := rosterFor(t, ts, "hermes", "deepseek")["accounts"].([]any); len(rows) != 2 {
		t.Fatalf("vault rows for deepseek = %d, want 2", len(rows))
	}
	if bad := cliRequestFull(t, ts, "POST", "/api/hermes/credential", map[string]any{"provider": "deepseek", "type": "api_key", "key": "reject"}); bad["status"] != "502" || !strings.Contains(fmt.Sprint(bad["body"]), "invalid API key") {
		t.Fatalf("refused key = %v", bad)
	}

	// Rows: OAuth, and a second sign-in while it waits.
	res := cliRequest(t, ts, "POST", "/api/hermes/credential", map[string]any{"provider": "nous", "type": "oauth"}, 200)
	if res["userCode"] != "Q48J-EVXF" || !strings.Contains(res["url"].(string), "user_code=") {
		t.Fatalf("oauth = %v", res)
	}
	if second := cliRequestFull(t, ts, "POST", "/api/hermes/credential", map[string]any{"provider": "nous", "type": "oauth"}); second["status"] != "409" {
		t.Fatalf("second = %v", second)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := cliRequest(t, ts, "GET", "/api/hermes/credential", nil, 200)
		if st["pending"] == false && st["done"] == true {
			if st["error"] != "" {
				t.Fatalf("oauth finished with %v", st)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if rows := rosterFor(t, ts, "hermes", "nous")["accounts"].([]any); len(rows) != 1 {
		t.Fatalf("the Nous sign-in was not filed in the vault: %v", rows)
	}

	// Row: refusals before Hermes runs.
	for _, body := range []map[string]any{
		{"provider": "mars", "type": "api_key", "key": "k"},
		{"provider": "nous", "type": "api_key", "key": "k"},
		{"provider": "deepseek", "type": "api_key", "key": ""},
	} {
		if st := cliRequestFull(t, ts, "POST", "/api/hermes/credential", body); st["status"] != "400" {
			t.Fatalf("POST %v = %v, want 400", body, st)
		}
	}
}
