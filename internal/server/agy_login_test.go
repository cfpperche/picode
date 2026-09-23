package server

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeAgy plays `agy -p` with no login as Antigravity 1.2.9 does (measured):
// Google's page, then a code read from the terminal. A good code writes the
// token file and then "runs the prompt" for a long while — PiCode must stop
// it the moment the token lands.
func fakeAgy(args []string) int {
	home := os.Getenv("HOME")
	_ = os.WriteFile(filepath.Join(home, "agy.pid"), []byte(strconv.Itoa(os.Getpid())), 0o600)
	if p, err := exec.LookPath("xdg-open"); err == nil {
		_ = os.WriteFile(filepath.Join(home, "agy-xdg-open"), []byte(p), 0o600)
	}
	fmt.Println("Authentication required. Please visit the URL to log in:\n  https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=fake&state=s1\n\nWaiting for authentication (timeout 60s)...\nOr, paste the authorization code here and press Enter:")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "good-code" {
		// Antigravity 1.2.9's own words (measured): the real error, then a
		// last line that says "timed out" for every failure.
		fmt.Println("Error: authentication failed: token exchange failed: oauth2: \"invalid_grant\" \"Bad Request\"")
		fmt.Println("error: authentication failed or timed out")
		return 1
	}
	dir := filepath.Join(os.Getenv("HOME"), ".gemini", "antigravity-cli")
	_ = os.MkdirAll(dir, 0o700)
	tok := `{"token":{"access_token":"agy-access","token_type":"Bearer","refresh_token":"agy-refresh","expiry":"2099-01-01T00:00:00Z"},"auth_method":"consumer"}`
	_ = os.WriteFile(filepath.Join(dir, "antigravity-oauth-token"), []byte(tok), 0o600)
	_ = os.WriteFile(filepath.Join(os.Getenv("HOME"), "agy-prompt-started"), []byte("x"), 0o600)
	time.Sleep(20 * time.Second)
	_ = os.WriteFile(filepath.Join(os.Getenv("HOME"), "agy-prompt-ran"), []byte("x"), 0o600)
	return 0
}

func installFakeAgy(t *testing.T) {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	script := "#!/bin/sh\nPICODE_FAKE_AGY=1 exec '" + self + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "agy"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitAgy(t *testing.T) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		agyLogin.mu.Lock()
		running, done, e := agyLogin.running, agyLogin.done, agyLogin.err
		agyLogin.mu.Unlock()
		if !running && done {
			return e
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the Antigravity sign-in never finished")
	return ""
}

// Antigravity's GUI sign-in (ADR-0197), one test per row:
//
//	start                    → Google's page back; agy waits under a pseudo terminal
//	a bad code               → Antigravity's own error; a login set aside is put back
//	a good code              → token written, agy stopped before the prompt runs, filed in the vault
//	a login already present  → filed in the vault and set aside, so agy asks instead of running the prompt
func TestAgyGUILogin(t *testing.T) {
	if _, err := os.Stat("/usr/bin/script"); err != nil {
		t.Skip("no script(1) on this machine")
	}
	ts, _, home, _ := credentialsServer(t)
	installFakeAgy(t)
	token := filepath.Join(home, ".gemini", "antigravity-cli", "antigravity-oauth-token")

	// Row: a login already present, then a bad code → it is put back.
	if err := os.MkdirAll(filepath.Dir(token), 0o700); err != nil {
		t.Fatal(err)
	}
	old := `{"token":{"access_token":"old-access","token_type":"Bearer","refresh_token":"old-refresh","expiry":"2099-01-01T00:00:00Z"},"auth_method":"consumer"}`
	if err := os.WriteFile(token, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	res := cliRequest(t, ts, "POST", "/api/agy/login", map[string]any{}, 200)
	if !strings.HasPrefix(res["url"].(string), "https://accounts.google.com/o/oauth2/auth?") {
		t.Fatalf("start = %v", res)
	}
	if _, err := os.Stat(token); !os.IsNotExist(err) {
		t.Fatal("the present login was not set aside: agy would have run the prompt")
	}
	if raw, _ := os.ReadFile(filepath.Join(home, "agy-xdg-open")); !strings.Contains(string(raw), "agy-noopen") {
		t.Fatalf("agy would open a browser on the PiCode machine through %q", raw)
	}
	cliRequest(t, ts, "POST", "/api/agy/login/code", map[string]any{"code": "wrong"}, 200)
	if e := waitAgy(t); !strings.Contains(e, "did not accept that code") {
		t.Fatalf("bad code finished with %q, want the code refusal, not the window closing", e)
	}

	// Row: cancel stops agy itself, not only script (script puts agy in a
	// session of its own).
	cliRequest(t, ts, "POST", "/api/agy/login", map[string]any{}, 200)
	time.Sleep(300 * time.Millisecond)
	pidRaw, _ := os.ReadFile(filepath.Join(home, "agy.pid"))
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidRaw)))
	cliRequest(t, ts, "DELETE", "/api/agy/login", nil, 200)
	waitAgy(t)
	gone := false
	for i := 0; i < 40; i++ {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
			gone = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !gone {
		t.Fatalf("agy (pid %d) outlived the cancel", pid)
	}
	if raw, _ := os.ReadFile(token); !strings.Contains(string(raw), "old-access") {
		t.Fatal("the login set aside was not put back after a cancel")
	}
	if raw, _ := os.ReadFile(token); !strings.Contains(string(raw), "old-access") {
		t.Fatal("the login set aside was not put back after a failed sign-in")
	}
	if rows := rosterFor(t, ts, "agy", "google")["accounts"].([]any); len(rows) != 1 {
		t.Fatalf("the present login was not filed in the vault first: %v", rows)
	}

	// Row: a good code.
	cliRequest(t, ts, "POST", "/api/agy/login", map[string]any{}, 200)
	start := time.Now()
	cliRequest(t, ts, "POST", "/api/agy/login/code", map[string]any{"code": "good-code"}, 200)
	if e := waitAgy(t); e != "" {
		t.Fatalf("good code finished with %q", e)
	}
	if took := time.Since(start); took > 8*time.Second {
		t.Fatalf("agy was not stopped at the token: %v", took)
	}
	if _, err := os.Stat(filepath.Join(home, "agy-prompt-ran")); err == nil {
		t.Fatal("the prompt ran to completion: agy was not stopped at the token")
	}
	if raw, _ := os.ReadFile(token); !strings.Contains(string(raw), "agy-access") {
		t.Fatalf("token = %s", raw)
	}
	// Antigravity's login carries no account name, so the vault keeps one
	// subscription row per provider (ADR-0013's rule): the new login is that row.
	if rows := rosterFor(t, ts, "agy", "google")["accounts"].([]any); len(rows) != 1 {
		t.Fatalf("vault rows after the new login = %d, want the one subscription row", len(rows))
	}

	// Refusals.
	if st := cliRequestFull(t, ts, "POST", "/api/agy/login/code", map[string]any{"code": "late"}); st["status"] != "409" {
		t.Fatalf("a code with nothing waiting = %v, want 409", st)
	}
	if roster := cliRequest(t, ts, "GET", "/api/credentials?cli=agy", nil, 200); roster["add"].(map[string]any)["kind"] != "agy" {
		t.Fatalf("add = %v", roster["add"])
	}
}
