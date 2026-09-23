package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/store"
)

const fakeGrokAuth = `{"https://auth.x.ai::client":{"key":"grok-access","refresh_token":"grok-refresh","expires_at":"2099-01-01T00:00:00Z","email":"me@x.ai","principal_id":"p1"}}`

// fakeGrok plays `grok login` and `grok logout` as Grok 1.0.41 does
// (measured): the page, the device code, a wait, then its own auth.json.
func fakeGrok(args []string) int {
	dir := os.Getenv("GROK_HOME")
	if dir == "" {
		dir = filepath.Join(os.Getenv("HOME"), ".grok")
	}
	_ = os.MkdirAll(dir, 0o700)
	auth := filepath.Join(dir, "auth.json")
	if len(args) == 0 {
		return 2
	}
	switch args[0] {
	case "logout":
		// A session Grok does not recognise stays in the file (measured).
		if os.Getenv("PICODE_FAKE_GROK_STICKY") != "1" {
			_ = os.Remove(auth)
		}
		return 0
	case "login":
		if os.Getenv("PICODE_FAKE_GROK_FAIL") == "1" {
			fmt.Println("error: the sign-in was denied")
			return 1
		}
		if len(args) > 1 && args[1] == "--device-auth" {
			fmt.Println("\nTo sign in, open this URL in your browser:\n\n  https://accounts.x.ai/oauth2/device?user_code=RDSJ-6FJ3\n\nConfirm this code in your browser:\n\n  \x1b[1mRDSJ-6FJ3\x1b[0m\n\nWaiting for authorization...")
			time.Sleep(time.Second)
		} else {
			fmt.Println("\nSigning in with Grok...\n\nOpen this URL to sign in:\n  https://auth.x.ai/oauth2/authorize?response_type=code&fake=1")
			time.Sleep(200 * time.Millisecond)
		}
		_ = os.WriteFile(auth, []byte(fakeGrokAuth), 0o600)
		return 0
	}
	return 2
}

func installFakeGrok(t *testing.T, home string) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_GROK=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "grok"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GROK_HOME", "")
	_ = home
}

func waitGrokLogin(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		grokLogin.mu.Lock()
		running, done, errText := grokLogin.running, grokLogin.done, grokLogin.err
		grokLogin.mu.Unlock()
		if !running && done {
			return errText
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the Grok sign-in never finished")
	return ""
}

// Grok's GUI sign-in (ADR-0192), one test per row:
//
//	browser / device        → the page (and code) back at once; done when grok login exits 0
//	a second sign-in        → 409 while one waits
//	Grok refuses            → its last line comes back
//	Use on a key row        → the session is filed in the vault, Grok signs out, XAI_API_KEY at launch
//	the person's env        → never overwritten
//	Use on the session row  → auth.json written back, the key stops riding
func TestGrokGUILogin(t *testing.T) {
	ts, dataDir, home, _ := credentialsServer(t)
	installFakeGrok(t, home)
	st := openTestStore(t, dataDir)
	deps := Deps{Store: st, DataDir: dataDir}
	auth := filepath.Join(home, ".grok", "auth.json")

	// Row: device code, and a second sign-in while it waits.
	res := cliRequest(t, ts, "POST", "/api/grok/login", map[string]any{"type": "device"}, 200)
	if res["userCode"] != "RDSJ-6FJ3" || !strings.HasPrefix(res["url"].(string), "https://accounts.x.ai/") {
		t.Fatalf("device = %v", res)
	}
	if second := cliRequestFull(t, ts, "POST", "/api/grok/login", map[string]any{"type": "oauth"}); second["status"] != "409" {
		t.Fatalf("second sign-in = %v, want 409", second)
	}
	if e := waitGrokLogin(t); e != "" {
		t.Fatalf("device finished with %q", e)
	}
	if _, err := os.Stat(auth); err != nil {
		t.Fatal("no auth.json after the device sign-in")
	}

	// Row: browser.
	res = cliRequest(t, ts, "POST", "/api/grok/login", map[string]any{"type": "oauth"}, 200)
	if !strings.HasPrefix(res["url"].(string), "https://auth.x.ai/") || res["userCode"] != nil {
		t.Fatalf("oauth = %v", res)
	}
	waitGrokLogin(t)

	// Row: Grok refuses.
	t.Setenv("PICODE_FAKE_GROK_FAIL", "1")
	if bad := cliRequestFull(t, ts, "POST", "/api/grok/login", map[string]any{"type": "oauth"}); bad["status"] != "502" || !strings.Contains(fmt.Sprint(bad["body"]), "denied") {
		t.Fatalf("refused = %v", bad)
	}
	t.Setenv("PICODE_FAKE_GROK_FAIL", "")

	// Row: a logout that leaves the session in place is refused.
	t.Setenv("PICODE_FAKE_GROK_STICKY", "1")
	sticky := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "xai", "key": "xai-sticky-key-0123456789abcdef"}, 201)
	if st := cliRequestFull(t, ts, "POST", "/api/credentials/xai/"+sticky["id"].(string)+"/activate", map[string]any{"cli": "grok"}); st["status"] != "502" {
		t.Fatalf("Use key with a sticky session = %v, want 502", st)
	}
	if env := grokCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("a refused switch still marked the key in use: %v", env)
	}
	if _, err := os.Stat(grokSessionCopy(deps)); !os.IsNotExist(err) {
		t.Fatal("a refused switch kept a copy of a session Grok would not let go of")
	}
	t.Setenv("PICODE_FAKE_GROK_STICKY", "")

	// Row: Use on a key row.
	key := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "xai", "key": "xai-test-key-0123456789abcdef"}, 201)
	keyID := key["id"].(string)
	cliRequest(t, ts, "POST", "/api/credentials/xai/"+keyID+"/activate", map[string]any{"cli": "grok"}, 200)
	if _, err := os.Stat(auth); !os.IsNotExist(err) {
		t.Fatal("Grok still holds its session: the key would be outranked")
	}
	roster := rosterFor(t, ts, "grok", "xai")
	sessionID := ""
	for _, a := range roster["accounts"].([]any) {
		row := a.(map[string]any)
		if row["type"] == "oauth" {
			sessionID = row["id"].(string)
		}
	}
	if sessionID == "" {
		t.Fatal("the signed-out session was not filed in the vault")
	}
	for _, a := range roster["accounts"].([]any) {
		row := a.(map[string]any)
		if row["id"] == sessionID && row["activatable"] != true {
			t.Fatal("the filed session offers no Use: the pane would promise a way back it does not show")
		}
		if row["id"] == sessionID && row["label"] != "me@x.ai" {
			t.Fatalf("the filed session is labelled %v, want Grok's own email", row["label"])
		}
	}
	if roster["note"] != nil && roster["note"] != "" {
		t.Fatalf("the outranked-key note still shows with the key in use: %v", roster["note"])
	}
	if !rosterRowActive(t, roster, keyID) {
		t.Fatal("the chosen key is not marked in use")
	}
	if env := grokCredentialEnv(deps, map[string]string{}); len(env) != 1 || env[0][0] != "XAI_API_KEY" {
		t.Fatalf("launch env = %v", env)
	}
	// Row: the person's env.
	if env := grokCredentialEnv(deps, map[string]string{"XAI_API_KEY": "mine"}); len(env) != 0 {
		t.Fatalf("a user-set XAI_API_KEY was overwritten: %v", env)
	}

	// Row: Use on the session row.
	cliRequest(t, ts, "POST", "/api/credentials/xai/"+sessionID+"/activate", map[string]any{"cli": "grok"}, 200)
	if _, err := os.Stat(auth); err != nil {
		t.Fatal("Use did not write the session back")
	}
	if env := grokCredentialEnv(deps, map[string]string{}); len(env) != 0 {
		t.Fatalf("the key still rides after the session was chosen: %v", env)
	}
	if roster := cliRequest(t, ts, "GET", "/api/credentials?cli=grok", nil, 200); roster["add"].(map[string]any)["kind"] != "grok" {
		t.Fatalf("add = %v", roster["add"])
	}
}

func openTestStore(t *testing.T, dataDir string) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
