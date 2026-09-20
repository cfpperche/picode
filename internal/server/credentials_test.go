package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func TestCredentialsRosterAndWrites(t *testing.T) {
	ts, _, home := cleanupServer(t)

	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	if roster["cli"] != "codex" || roster["cliName"] != "Codex" {
		t.Fatalf("roster = %v %v", roster["cli"], roster["cliName"])
	}
	if vault, _ := roster["vault"].(map[string]any); vault["readable"] != true {
		t.Fatalf("vault = %v, want a readable empty vault", roster["vault"])
	}
	providers, _ := roster["providers"].([]any)
	if len(providers) == 0 {
		t.Fatal("codex declares no providers")
	}

	// An unknown provider cannot be stored: the roster is a closed vocabulary.
	if bad := cliRequestFull(t, ts, "POST", "/api/credentials", map[string]any{"provider": "nope", "key": "x"}); bad["status"] != "400" {
		t.Fatalf("unknown provider = %v, want 400", bad)
	}

	added := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "openai-codex", "label": "Work", "key": "sk-test-1234567890abcd",
	}, 201)
	id, _ := added["id"].(string)
	if id == "" || added["label"] != "Work" || added["type"] != "api_key" {
		t.Fatalf("added = %v", added)
	}
	if added["hint"] != "sk-test…abcd" {
		t.Fatalf("hint = %v, want the masked key", added["hint"])
	}

	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	raw, _ := json.Marshal(roster)
	if strings.Contains(string(raw), "sk-test-1234567890abcd") {
		t.Fatal("the roster returned the key itself")
	}
	if !strings.Contains(string(raw), "sk-test…abcd") {
		t.Fatal("the roster lost the masked hint")
	}

	// Rename, then pause — refused while it is the only live row, because that
	// is Sign out with extra steps (ADR-0013's rule, kept).
	cliRequest(t, ts, "PATCH", "/api/credentials/openai-codex/"+id, map[string]any{"label": "Renamed"}, 200)
	if bad := cliRequestFull(t, ts, "POST", "/api/credentials/openai-codex/"+id+"/pause", map[string]any{"paused": true}); bad["status"] != "400" {
		t.Fatalf("pausing the only live row = %v, want 400", bad)
	}

	// A second row makes pausing the first one fine.
	second := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "openai-codex", "key": "sk-test-secondkey-9876",
	}, 201)
	cliRequest(t, ts, "POST", "/api/credentials/openai-codex/"+id+"/pause", map[string]any{"paused": true}, 200)
	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	paused := false
	label := ""
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] != "openai-codex" {
			continue
		}
		for _, a := range row["accounts"].([]any) {
			account := a.(map[string]any)
			if account["id"] == id {
				paused, _ = account["paused"].(bool)
				label, _ = account["label"].(string)
			}
		}
	}
	if !paused || label != "Renamed" {
		t.Fatalf("row after pause = paused:%v label:%q", paused, label)
	}

	cliRequest(t, ts, "DELETE", "/api/credentials/openai-codex/"+id, nil, 204)
	cliRequest(t, ts, "DELETE", "/api/credentials/openai-codex/"+second["id"].(string), nil, 204)
	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	for _, p := range roster["providers"].([]any) {
		if row := p.(map[string]any); row["id"] == "openai-codex" && len(row["accounts"].([]any)) != 0 {
			t.Fatalf("accounts survived the delete: %v", row["accounts"])
		}
	}

	if bad := cliRequestFull(t, ts, "GET", "/api/credentials?cli=nope", nil); bad["status"] != "404" {
		t.Fatalf("unknown CLI = %v, want 404", bad)
	}
	_ = home
}

func TestCredentialImportReadsTheCLIsOwnLogin(t *testing.T) {
	ts, _, home := cleanupServer(t)
	dir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	creds := `{"claudeAiOauth":{"accessToken":"oat01-access","refreshToken":"oat01-refresh","expiresAt":1767225600000,"subscriptionType":"max"}}`
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte(creds), 0o600); err != nil {
		t.Fatal(err)
	}

	// Detection appears on the roster before anything is imported.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	found := false
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] != "anthropic" {
			continue
		}
		native, _ := row["native"].(map[string]any)
		if native == nil || native["detected"] != true || native["imported"] == true {
			t.Fatalf("anthropic native = %v, want a detected, unimported login", row["native"])
		}
		found = true
	}
	if !found {
		t.Fatal("claude-code does not declare anthropic")
	}

	imported := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code"}, 201)
	if imported["origin"] != "imported:claude-code" || imported["type"] != "oauth" {
		t.Fatalf("imported = %v", imported)
	}
	body, _ := json.Marshal(imported)
	if strings.Contains(string(body), "oat01-access") {
		t.Fatal("the import answered with the token")
	}

	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	importedAgain := false
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] != "anthropic" {
			continue
		}
		if native, _ := row["native"].(map[string]any); native != nil && native["imported"] == true {
			importedAgain = true
		}
		if len(row["accounts"].([]any)) != 1 {
			t.Fatalf("accounts = %v, want the imported row", row["accounts"])
		}
	}
	if !importedAgain {
		t.Fatal("the roster still offers Import for a credential already in the vault")
	}

	// A CLI with no detectable login says so instead of inventing a row.
	if bad := cliRequestFull(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "grok"}); bad["status"] != "404" {
		t.Fatalf("import with no login = %v, want 404", bad)
	}
}

func TestCredentialVerifySpendsOneListingCall(t *testing.T) {
	ts, _, _ := cleanupServer(t)

	calls := 0
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("x-api-key") == "" {
			t.Errorf("verify sent no key header")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	old := probeTable["anthropic"]
	probeTable["anthropic"] = &probe{url: srv.URL, header: xAPIKey}
	t.Cleanup(func() { probeTable["anthropic"] = old })

	row := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "anthropic", "key": "sk-ant-test-abcdefghijkl",
	}, 201)
	id, _ := row["id"].(string)

	ok := cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+id+"/verify", map[string]any{}, 200)
	if ok["state"] != "ok" {
		t.Fatalf("verify = %v, want ok", ok)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want exactly one listing request", calls)
	}

	// The provider refusing the key is a state on the row, not an error to the
	// browser: the pane renders it as the row's health.
	status = http.StatusUnauthorized
	bad := cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+id+"/verify", map[string]any{}, 200)
	if bad["state"] != "invalid" {
		t.Fatalf("verify = %v, want invalid", bad)
	}
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	health := ""
	for _, p := range roster["providers"].([]any) {
		entry := p.(map[string]any)
		if entry["id"] != "anthropic" {
			continue
		}
		for _, a := range entry["accounts"].([]any) {
			account := a.(map[string]any)
			if h, ok := account["health"].(map[string]any); ok && account["id"] == id {
				health, _ = h["state"].(string)
			}
		}
	}
	if health != "invalid" {
		t.Fatalf("health on the roster = %q, want the cached verdict", health)
	}

	// A provider with no probe hides the control and refuses the call.
	plain := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "radius", "key": "r-test-key-123456",
	}, 201)
	if bad := cliRequestFull(t, ts, "POST", "/api/credentials/radius/"+plain["id"].(string)+"/verify", map[string]any{}); bad["status"] != "400" {
		t.Fatalf("verify without a probe = %v, want 400", bad)
	}
}

// credentialsServer is cleanupServer plus the terminal-runtime registry, so a
// test can prove the activation refusal while a CLI actually runs.
func credentialsServer(t *testing.T) (*httptest.Server, string, string, *TermRuntimes) {
	t.Helper()
	ts, dataDir, home := cleanupServer(t)
	runtimes := NewTermRuntimes()
	// cleanupServer already built its server; rebuild one around the same
	// store and data dir with the runtimes wired in.
	ts.Close()
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	next := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime("cat", st, nil), AgentCmd: "cat",
		DataDir: dataDir, TermRuntimes: runtimes,
	}).Handler)
	t.Cleanup(next.Close)
	return next, dataDir, home, runtimes
}

func writeCodexLogin(t *testing.T, home, access, refresh, accountID string) string {
	t.Helper()
	dir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"auth_mode":"chatgpt","tokens":{"access_token":%q,"refresh_token":%q,"account_id":%q},"unknown_key":{"kept":true}}`, access, refresh, accountID)
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func rosterFor(t *testing.T, ts *httptest.Server, cli, provider string) map[string]any {
	t.Helper()
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli="+cli, nil, 200)
	for _, p := range roster["providers"].([]any) {
		row := p.(map[string]any)
		if row["id"] == provider {
			return row
		}
	}
	t.Fatalf("%s does not declare %s", cli, provider)
	return nil
}

func TestCredentialActivateWritesTheCLIsOwnFile(t *testing.T) {
	ts, dataDir, home, _ := credentialsServer(t)
	firstPath := writeCodexLogin(t, home, "access-one", "refresh-one", "acct-one")

	// Import the login the CLI has, then let the CLI log into another account
	// by itself: the file, not a click, is what says which one is in use.
	cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "codex"}, 201)
	writeCodexLogin(t, home, "access-two", "refresh-two", "acct-two")
	cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "codex"}, 201)

	provider := rosterFor(t, ts, "codex", "openai-codex")
	accounts := provider["accounts"].([]any)
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want the two Codex logins kept", len(accounts))
	}
	inUse := ""
	for _, a := range accounts {
		row := a.(map[string]any)
		if row["active"] == true {
			inUse, _ = row["label"].(string)
		}
		if row["activatable"] != true {
			t.Fatalf("row %v is not activatable", row["id"])
		}
	}
	if inUse == "" {
		t.Fatalf("no row reported in use: %v", accounts)
	}

	// Activate the older one: the file it reads changes, the unknown key in
	// that file survives, and the original is kept once.
	var target map[string]any
	for _, a := range accounts {
		row := a.(map[string]any)
		if row["active"] != true {
			target = row
		}
	}
	if target == nil {
		t.Fatal("no inactive row to activate")
	}
	res := cliRequest(t, ts, "POST", "/api/credentials/openai-codex/"+target["id"].(string)+"/activate", map[string]any{"cli": "codex"}, 200)
	if res["ok"] != true {
		t.Fatalf("activate = %v", res)
	}
	backup, _ := res["backup"].(string)
	if backup == "" {
		t.Fatal("the replaced file was not kept")
	}
	raw, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "access-one") {
		t.Fatalf("the CLI's file was not rewritten: %s", raw)
	}
	if !strings.Contains(string(raw), "unknown_key") {
		t.Fatalf("a key PiCode does not own was dropped: %s", raw)
	}

	// The file now answers "in use", and a second activation of the other
	// account does not keep a second backup.
	provider = rosterFor(t, ts, "codex", "openai-codex")
	for _, a := range provider["accounts"].([]any) {
		row := a.(map[string]any)
		if row["id"] == target["id"] && row["active"] != true {
			t.Fatalf("activated row is not marked in use: %v", row)
		}
	}
	other := ""
	for _, a := range provider["accounts"].([]any) {
		row := a.(map[string]any)
		if row["id"] != target["id"] {
			other = row["id"].(string)
		}
	}
	res2 := cliRequest(t, ts, "POST", "/api/credentials/openai-codex/"+other+"/activate", map[string]any{"cli": "codex"}, 200)
	if res2["backup"] != backup {
		t.Fatalf("second activation kept another backup: %v vs %v", res2["backup"], backup)
	}
	if got, _ := filepath.Glob(filepath.Join(dataDir, "credfiles", "codex-*.bak")); len(got) != 1 {
		t.Fatalf("backups = %v, want exactly one", got)
	}
}

func TestCredentialActivateRefusesWhileTheCLIRuns(t *testing.T) {
	ts, _, home, runtimes := credentialsServer(t)
	writeCodexLogin(t, home, "access-one", "refresh-one", "acct-one")
	imported := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "codex"}, 201)
	id, _ := imported["id"].(string)

	if _, ok := runtimes.Start("terminal-fixture", TermRuntime{CLI: "codex", RunID: "run-1", PID: os.Getpid()}); !ok {
		t.Fatal("fixture runtime did not register")
	}
	res := cliRequestFull(t, ts, "POST", "/api/credentials/openai-codex/"+id+"/activate", map[string]any{"cli": "codex"})
	if res["status"] != "409" {
		t.Fatalf("activate with a live terminal = %v, want 409", res)
	}
	if body, _ := res["body"].(map[string]any); body == nil || !strings.Contains(fmt.Sprint(body["error"]), "Close the 1 running Codex terminal") {
		t.Fatalf("refusal does not name the running terminal: %v", res)
	}

	runtimes.Drop("terminal-fixture")
	cliRequest(t, ts, "POST", "/api/credentials/openai-codex/"+id+"/activate", map[string]any{"cli": "codex"}, 200)
}

func TestCredentialActivateRefusesWhatItCannotWrite(t *testing.T) {
	ts, _, _, _ := credentialsServer(t)

	// Omp has no credential file at all.
	added := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{"provider": "openai-codex", "key": "sk-omp-1234567890"}, 201)
	res := cliRequestFull(t, ts, "POST", "/api/credentials/openai-codex/"+added["id"].(string)+"/activate", map[string]any{"cli": "omp"})
	if res["status"] != "400" {
		t.Fatalf("activate for omp = %v, want 400", res)
	}

	// A provider the CLI does not use.
	res = cliRequestFull(t, ts, "POST", "/api/credentials/openai-codex/"+added["id"].(string)+"/activate", map[string]any{"cli": "muse"})
	if res["status"] != "400" {
		t.Fatalf("activate for an undeclared provider = %v, want 400", res)
	}
	if body, _ := res["body"].(map[string]any); body == nil || !strings.Contains(fmt.Sprint(body["error"]), "does not use that provider") {
		t.Fatalf("refusal does not say why: %v", res)
	}
}
