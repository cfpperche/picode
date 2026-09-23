package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeMuse plays `muse login` as Muse 1.3.0 does (measured): the page, the
// code, a wait, then its own auth.json with the account login.
func fakeMuse(args []string) int {
	if len(args) == 0 || args[0] != "login" {
		return 2
	}
	fmt.Println("Open this page to sign in:\n  https://auth.meta.com/oauth/device/?code=SGGJ-XPZJ\nconfirm this code matches:\n  SGGJ-XPZJ\nWaiting for approval…")
	time.Sleep(time.Second)
	dir := filepath.Join(os.Getenv("HOME"), ".config", "muse")
	_ = os.MkdirAll(dir, 0o700)
	body := `{"schema_version":1,"providers":{"meta":{"mechanism":"oauth","access_token":"meta-access","refresh_token":"meta-refresh","expires_at":4102444800,"user_email":"me@meta.test"}}}`
	_ = os.WriteFile(filepath.Join(dir, "auth.json"), []byte(body), 0o600)
	return 0
}

func installFakeMuse(t *testing.T) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_MUSE=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "muse"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Muse's GUI sign-in (ADR-0195), one test per row:
//
//	account (device code)   → page and code at once; Muse's login filed in the vault on exit 0
//	a second sign-in        → 409 while one waits
//	Use on a key row        → the account login is filed first, then the key replaces it in Muse's one slot
//	Use on the account row  → the login written back
func TestMuseGUILogin(t *testing.T) {
	ts, _, home, _ := credentialsServer(t)
	installFakeMuse(t)
	authPath := filepath.Join(home, ".config", "muse", "auth.json")

	res := cliRequest(t, ts, "POST", "/api/muse/login", map[string]any{}, 200)
	if res["userCode"] != "SGGJ-XPZJ" || !strings.HasPrefix(res["url"].(string), "https://auth.meta.com/") {
		t.Fatalf("login = %v", res)
	}
	if second := cliRequestFull(t, ts, "POST", "/api/muse/login", map[string]any{}); second["status"] != "409" {
		t.Fatalf("second = %v", second)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		st := cliRequest(t, ts, "GET", "/api/muse/login", nil, 200)
		if st["pending"] == false && st["done"] == true {
			if st["error"] != "" {
				t.Fatalf("login finished with %v", st)
			}
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	accountID := ""
	for _, a := range rosterFor(t, ts, "muse", "meta-ai")["accounts"].([]any) {
		if row := a.(map[string]any); row["type"] == "oauth" {
			accountID = row["id"].(string)
		}
	}
	if accountID == "" {
		t.Fatal("the Muse login was not filed in the vault")
	}

	// Use on a key: the account is already in the vault; Muse's slot takes the key.
	key := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "meta-ai", "key": "meta-key-0123456789abcdef"}, 201)
	cliRequest(t, ts, "POST", "/api/credentials/meta-ai/"+key["id"].(string)+"/activate", map[string]any{"cli": "muse"}, 200)
	raw, _ := os.ReadFile(authPath)
	if !strings.Contains(string(raw), "meta-key-0123456789abcdef") || strings.Contains(string(raw), "meta-access") {
		t.Fatalf("auth.json after Use key = %s", raw)
	}
	// Use on the account row writes the login back.
	cliRequest(t, ts, "POST", "/api/credentials/meta-ai/"+accountID+"/activate", map[string]any{"cli": "muse"}, 200)
	raw, _ = os.ReadFile(authPath)
	if !strings.Contains(string(raw), "meta-access") {
		t.Fatalf("auth.json after Use account = %s", raw)
	}
	if roster := cliRequest(t, ts, "GET", "/api/credentials?cli=muse", nil, 200); roster["add"].(map[string]any)["kind"] != "muse" {
		t.Fatalf("add = %v", roster["add"])
	}
}

// A login made in Muse's own terminal and never imported is filed before Use
// replaces it: the one slot never loses a credential PiCode did not keep.
func TestMuseUseFilesTheLoginItReplaces(t *testing.T) {
	ts, _, home, _ := credentialsServer(t)
	dir := filepath.Join(home, ".config", "muse")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte(`{"schema_version":1,"providers":{"meta":{"mechanism":"oauth","access_token":"only-in-muse","refresh_token":"r","expires_at":4102444800,"user_email":"solo@meta.test"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	key := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "meta-ai", "key": "meta-key-solo-0123456789"}, 201)
	cliRequest(t, ts, "POST", "/api/credentials/meta-ai/"+key["id"].(string)+"/activate", map[string]any{"cli": "muse"}, 200)
	found := false
	for _, a := range rosterFor(t, ts, "muse", "meta-ai")["accounts"].([]any) {
		if row := a.(map[string]any); row["type"] == "oauth" {
			found = true
		}
	}
	if !found {
		t.Fatal("the login Muse held was lost when the key replaced it")
	}
	for _, a := range rosterFor(t, ts, "muse", "meta-ai")["accounts"].([]any) {
		if row := a.(map[string]any); row["type"] == "oauth" && row["label"] != "solo@meta.test" {
			t.Fatalf("the filed login is labelled %v, want Muse's own email", row["label"])
		}
	}
}
