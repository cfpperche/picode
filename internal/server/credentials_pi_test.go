package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/credentials"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/usage"
)

// piRosterServer serves the roster for cli=pi: cleanupServer's sandbox (HOME,
// PICODE_DATA, XDG) with a fake pi, because pi's providers now come from the
// catalog. What auth.json and models.json hold is the test's to write.
func piRosterServer(t *testing.T, authJSON string) (*httptest.Server, string) {
	t.Helper()
	root := t.TempDir()
	home := filepath.Join(root, "home")
	dataDir := filepath.Join(root, "data")
	for _, dir := range []string{home, dataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PICODE_DATA", dataDir)
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	st, err := store.Open(filepath.Join(dataDir, "picode.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	pi := fakePi(t, authJSON)
	ts := httptest.NewServer(New("127.0.0.1:0", Deps{
		Store: st, Tmux: tmux.New(), Runtime: rpc.NewRuntime(pi, st, nil),
		AgentCmd: pi, DataDir: dataDir,
	}).Handler)
	t.Cleanup(ts.Close)
	return ts, home
}

func rosterProvider(t *testing.T, roster map[string]any, id string) map[string]any {
	t.Helper()
	for _, raw := range roster["providers"].([]any) {
		p := raw.(map[string]any)
		if p["id"] == id {
			return p
		}
	}
	t.Fatalf("provider %q is not in the roster: %v", id, roster["providers"])
	return nil
}

func rosterAccount(t *testing.T, provider map[string]any, id string) map[string]any {
	t.Helper()
	for _, raw := range provider["accounts"].([]any) {
		a := raw.(map[string]any)
		if a["id"] == id {
			return a
		}
	}
	t.Fatalf("account %q is not under provider %v: %v", id, provider["id"], provider["accounts"])
	return nil
}

// authEntry reads one provider's entry out of pi's own file.
func authEntry(t *testing.T, home, provider string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		t.Fatalf("pi's auth.json: %v", err)
	}
	var obj map[string]map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("pi's auth.json is not an object: %v", err)
	}
	return obj[provider]
}

// Pi's half of the roster comes from the catalog (ADR-0169): what pi lists
// plus the models.json definitions, each entry carrying the check pi can run
// and the door to the custom-endpoint page.
func TestPiRosterProvidersComeFromTheCatalog(t *testing.T) {
	ts, home := piRosterServer(t, `{"status":"ready"}`)
	if err := os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	models := `{"providers":{"acme-gateway":{"baseUrl":"https://acme.test/v1","api":"openai-completions","models":[{"id":"acme-1"}]}}}`
	if err := os.WriteFile(filepath.Join(home, ".pi", "agent", "models.json"), []byte(models), 0o600); err != nil {
		t.Fatal(err)
	}

	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=pi", nil, 200)
	if roster["cli"] != "pi" || roster["cliName"] != "Pi" {
		t.Fatalf("roster = %v %v", roster["cli"], roster["cliName"])
	}
	if vault, _ := roster["vault"].(map[string]any); vault["readable"] != true {
		t.Fatalf("vault = %v, want a readable empty vault", roster["vault"])
	}
	add, _ := roster["add"].(map[string]any)
	if add["kind"] != "provider" || add["label"] != "Add provider" {
		t.Fatalf("add = %v, want pi's own dialog", roster["add"])
	}
	custom, _ := roster["custom"].(map[string]any)
	if custom["available"] != true || custom["href"] != "#/clis/pi/providers/custom" {
		t.Fatalf("custom = %v, want the models.json page", roster["custom"])
	}
	if signin, _ := roster["signin"].(map[string]any); signin["available"] != true {
		t.Fatalf("signin = %v, want pi's own /login", roster["signin"])
	}

	cases := []struct {
		name       string
		provider   string
		wantCustom bool
		wantKinds  []string
		wantEnv    string
	}{
		{
			// A provider pi's own declaration names: the shapes and the
			// variable come from there.
			name: "declared native provider", provider: "anthropic",
			wantKinds: []string{"api_key", "oauth"}, wantEnv: "ANTHROPIC_API_KEY",
		},
		{
			// Only the catalog knows this one — it is not in pi's
			// declaration, so it is proof the list is the catalog's.
			name: "provider only the catalog names", provider: "deepseek",
			wantKinds: []string{"api_key"}, wantEnv: "DEEPSEEK_API_KEY",
		},
		{
			// A models.json definition: the pane offers Edit provider for it.
			name: "models.json definition", provider: "acme-gateway",
			wantCustom: true, wantKinds: []string{"api_key"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := rosterProvider(t, roster, tc.provider)
			if got, _ := p["custom"].(bool); got != tc.wantCustom {
				t.Fatalf("%s custom = %v, want %v", tc.provider, p["custom"], tc.wantCustom)
			}
			if p["verify"] != verifyByProvider {
				t.Fatalf("%s verify = %v, want %q", tc.provider, p["verify"], verifyByProvider)
			}
			kinds, _ := p["kinds"].([]any)
			got := make([]string, 0, len(kinds))
			for _, k := range kinds {
				got = append(got, k.(string))
			}
			if strings.Join(got, ",") != strings.Join(tc.wantKinds, ",") {
				t.Fatalf("%s kinds = %v, want %v", tc.provider, got, tc.wantKinds)
			}
			if tc.wantEnv != "" {
				env, _ := p["env"].(map[string]any)
				if env["api_key"] != tc.wantEnv {
					t.Fatalf("%s env = %v, want api_key %s", tc.provider, p["env"], tc.wantEnv)
				}
			}
			if _, ok := p["accounts"].([]any); !ok {
				t.Fatalf("%s has no accounts array", tc.provider)
			}
		})
	}
}

// A row's usage is the cached report — the same entry /api/providers/usage
// serves — and a row the cache never saw keeps no usage key at all, so the
// pane renders "unknown" with Check and no vendor is called (ADR-0031).
func TestPiRowUsageComesFromTheCacheAlone(t *testing.T) {
	ts, _ := piRosterServer(t, `{"status":"ready","provider":"anthropic","authType":"oauth"}`)

	subscription, err := credentials.Default().Import("anthropic",
		json.RawMessage(`{"type":"oauth","access":"access-1","refresh":"refresh-1","accountId":"acct-1"}`),
		"Sub", "vault", "")
	if err != nil {
		t.Fatal(err)
	}
	keyRow, err := credentials.Default().Import("anthropic",
		json.RawMessage(`{"type":"api_key","key":"sk-ant-usage-test"}`), "Key", "vault", "")
	if err != nil {
		t.Fatal(err)
	}
	percent := 42.0
	usage.Remember("anthropic", subscription.ID, usage.Report{
		Provider: "anthropic", Status: "ok", Plan: "Max",
		Windows:   []usage.Window{{ID: "5h", Label: "5-hour limit", UsedPercent: &percent, Unit: "percent"}},
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}, time.Now())
	t.Cleanup(func() { usage.Forget("anthropic", subscription.ID) })

	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=pi", nil, 200)
	anthropic := rosterProvider(t, roster, "anthropic")

	cases := []struct {
		name      string
		account   string
		wantUsage bool
	}{
		{name: "the row the cache has a report for", account: subscription.ID, wantUsage: true},
		{name: "a row the cache never saw", account: keyRow.ID, wantUsage: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := rosterAccount(t, anthropic, tc.account)
			raw, has := row["usage"]
			if has != tc.wantUsage {
				t.Fatalf("usage = %v (present %v), want present %v", raw, has, tc.wantUsage)
			}
			if !tc.wantUsage {
				return
			}
			entry, _ := raw.(map[string]any)
			if entry["status"] != "ok" || entry["plan"] != "Max" || entry["accountId"] != subscription.ID {
				t.Fatalf("usage = %v, want the cached report", entry)
			}
			windows, _ := entry["windows"].([]any)
			if len(windows) != 1 {
				t.Fatalf("windows = %v, want the vendor's one window", entry["windows"])
			}
			window, _ := windows[0].(map[string]any)
			if window["label"] != "5-hour limit" || window["usedPercent"] != percent {
				t.Fatalf("window = %v, want the vendor's window", window)
			}
		})
	}
}

// Pi's rows go through the same vault verbs as a guest's: Use writes the
// chosen account into pi's own file (ADR-0166), and Pause takes the row out of
// play — promoting the other live row into the file, which is what pausing the
// active slot means.
func TestPiRosterActivatesAndPausesARow(t *testing.T) {
	ts, home := piRosterServer(t, `{"status":"ready","provider":"anthropic","authType":"oauth"}`)

	work := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "anthropic", "label": "Work", "key": "sk-ant-first-key-0001",
	}, 201)
	id, _ := work["id"].(string)
	if id == "" {
		t.Fatalf("added = %v", work)
	}
	other := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "anthropic", "key": "sk-ant-second-key-0002",
	}, 201)

	// Nothing is in pi's file yet: the row is offered, not in use.
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=pi", nil, 200)
	anthropic := rosterProvider(t, roster, "anthropic")
	row := rosterAccount(t, anthropic, id)
	if row["activatable"] != true || row["active"] == true {
		t.Fatalf("row before Use = %v, want activatable and not in use", row)
	}

	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+id+"/activate", map[string]any{"cli": "pi"}, 200)
	entry := authEntry(t, home, "anthropic")
	if entry["type"] != "api_key" || entry["key"] != "sk-ant-first-key-0001" {
		t.Fatalf("pi's auth.json holds %v, want the chosen row's key", entry)
	}

	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=pi", nil, 200)
	anthropic = rosterProvider(t, roster, "anthropic")
	if rosterAccount(t, anthropic, id)["active"] != true {
		t.Fatalf("the used row is not marked in use: %v", rosterAccount(t, anthropic, id))
	}

	// Pause takes it out of play: the vault says so, and the other live row is
	// promoted into pi's file.
	cliRequest(t, ts, "POST", "/api/credentials/anthropic/"+id+"/pause", map[string]any{"paused": true}, 200)
	roster = cliRequest(t, ts, "GET", "/api/credentials?cli=pi", nil, 200)
	anthropic = rosterProvider(t, roster, "anthropic")
	paused := rosterAccount(t, anthropic, id)
	if paused["paused"] != true || paused["active"] == true {
		t.Fatalf("row after Pause = %v, want paused and out of use", paused)
	}
	if entry := authEntry(t, home, "anthropic"); entry["key"] != "sk-ant-second-key-0002" {
		t.Fatalf("pi's auth.json holds %v, want the promoted row's key", entry)
	}
	_ = other
}

// The guest roster keeps what it had — providers from the declaration, their
// kinds, notes and rows — and adds only the pane's fields. Verify is the row's
// listing probe where one exists, and absent where it does not, so the pane
// offers no control that could only fail.
func TestGuestRosterCarriesThePaneFieldsOnly(t *testing.T) {
	ts, _, _ := cleanupServer(t)

	cases := []struct {
		name       string
		cli        string
		provider   string
		wantKinds  []string
		wantEnv    string
		wantVerify string
		wantNote   string
	}{
		{
			name: "a provider with a listing probe", cli: "codex", provider: "openai-codex",
			wantKinds: []string{"api_key", "oauth"}, wantEnv: "OPENAI_API_KEY", wantVerify: verifyByRow,
		},
		{
			name: "a provider with a note and a probe", cli: "grok", provider: "xai",
			wantKinds: []string{"api_key", "oauth"}, wantEnv: "XAI_API_KEY", wantVerify: verifyByRow,
			wantNote: "signed-in Grok session wins",
		},
		{
			name: "a provider with no honest check", cli: "muse", provider: "meta-ai",
			wantKinds: []string{"api_key", "oauth"}, wantEnv: "META_API_KEY",
			wantNote: "works only inside Muse",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roster := cliRequest(t, ts, "GET", "/api/credentials?cli="+tc.cli, nil, 200)
			add, _ := roster["add"].(map[string]any)
			// The kind names the dialog the pane opens: the key form, or a
			// CLI's own sign-in dialog (ADR-0187, ADR-0191, ADR-0192).
			wantKind := map[string]string{"codex": "codex", "claude-code": "claude-code", "grok": "grok", "omp": "provider"}[tc.cli]
			if wantKind == "" {
				wantKind = "key"
			}
			if add["kind"] != wantKind || add["label"] != "Add provider" {
				t.Fatalf("%s add = %v, want the provider door", tc.cli, roster["add"])
			}
			if _, isPi := roster["custom"]; isPi {
				t.Fatalf("%s carries pi's custom door: %v", tc.cli, roster["custom"])
			}
			p := rosterProvider(t, roster, tc.provider)
			if got := p["verify"]; got != tc.wantVerify && !(tc.wantVerify == "" && got == nil) {
				t.Fatalf("%s verify = %v, want %q", tc.provider, got, tc.wantVerify)
			}
			if _, stale := p["verifier"]; stale {
				t.Fatalf("%s still carries the old verifier field", tc.provider)
			}
			if _, isPi := p["custom"]; isPi {
				t.Fatalf("%s carries custom: %v", tc.provider, p["custom"])
			}
			kinds, _ := p["kinds"].([]any)
			got := make([]string, 0, len(kinds))
			for _, k := range kinds {
				got = append(got, k.(string))
			}
			if strings.Join(got, ",") != strings.Join(tc.wantKinds, ",") {
				t.Fatalf("%s kinds = %v, want %v", tc.provider, got, tc.wantKinds)
			}
			if env, _ := p["env"].(map[string]any); env["api_key"] != tc.wantEnv {
				t.Fatalf("%s env = %v, want api_key %s", tc.provider, p["env"], tc.wantEnv)
			}
			if tc.wantNote != "" {
				note, _ := p["note"].(string)
				if !strings.Contains(note, tc.wantNote) {
					t.Fatalf("%s note = %q, want it to name %q", tc.provider, note, tc.wantNote)
				}
			}
			if _, ok := p["accounts"].([]any); !ok {
				t.Fatalf("%s has no accounts array", tc.provider)
			}
		})
	}

	// The rows are what the pane always got: a saved key keeps the row's
	// shape, gains no usage key until the cache holds a report, and stays
	// activatable because Codex's own file can hold it.
	added := cliRequest(t, ts, "POST", "/api/credentials", map[string]any{
		"provider": "openai-codex", "key": "sk-test-guest-key-0001",
	}, 201)
	roster := cliRequest(t, ts, "GET", "/api/credentials?cli=codex", nil, 200)
	row := rosterAccount(t, rosterProvider(t, roster, "openai-codex"), added["id"].(string))
	if row["activatable"] != true || row["origin"] != "vault" || row["type"] != "api_key" {
		t.Fatalf("guest row = %v, want the vault row Codex's file can hold", row)
	}
	if _, has := row["usage"]; has {
		t.Fatalf("guest row carries usage with no cached report: %v", row["usage"])
	}

	if bad := cliRequestFull(t, ts, "GET", "/api/credentials?cli=nope", nil); bad["status"] != "404" {
		t.Fatalf("unknown CLI = %v, want 404", bad)
	}
}

// One sign-in terminal per CLI: a second Sign in leads back to the terminal
// already waiting, and a record whose tmux session died is removed instead of
// piling sign-in ghosts into the terminals list (thirteen "Claude Code
// sign-in" sessions on one machine, 2026-09-21).
func TestCredentialSigninReusesItsTerminal(t *testing.T) {
	if !tmux.New().Available() {
		t.Skip("tmux not installed")
	}
	ts, _, _ := cleanupServer(t)
	cliRequest(t, ts, "PUT", "/api/clis/pi", clilaunch.Config{Executable: "/bin/cat", Args: []string{}}, 200)

	first := cliRequest(t, ts, "POST", "/api/credentials/signin", map[string]any{"cli": "pi"}, 201)
	id1, _ := first["terminalId"].(string)
	if id1 == "" || first["reused"] == true {
		t.Fatalf("first sign-in = %v, want a fresh terminal", first)
	}
	t.Cleanup(func() {
		_ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id1))
	})

	second := cliRequest(t, ts, "POST", "/api/credentials/signin", map[string]any{"cli": "pi"}, 200)
	if second["terminalId"] != id1 || second["reused"] != true {
		t.Fatalf("second sign-in = %v, want the same terminal reused", second)
	}

	// The session died (the person closed it after signing in): the record is
	// a husk and the next Sign in replaces it — once past the grace a
	// just-created sign-in gets (signinGrace; zero here).
	grace := signinGrace
	signinGrace = 0
	t.Cleanup(func() { signinGrace = grace })
	_ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id1))
	third := cliRequest(t, ts, "POST", "/api/credentials/signin", map[string]any{"cli": "pi"}, 201)
	id3, _ := third["terminalId"].(string)
	if id3 == "" || id3 == id1 || third["reused"] == true {
		t.Fatalf("sign-in after the session died = %v, want a fresh terminal", third)
	}
	t.Cleanup(func() {
		_ = tmux.New().KillSession(context.Background(), tmux.ShellSessionName(id3))
	})
	roster := cliRequest(t, ts, "GET", "/api/terminals", nil, 200)
	for _, row := range roster["terminals"].([]any) {
		if row.(map[string]any)["id"] == id1 {
			t.Fatal("the dead sign-in record is still in the list")
		}
	}
}

// The pane asks for a name BEFORE the write when this login would replace an
// unnamed one: after the write the replaced tokens are already gone. The
// preview answers without touching the vault.
func TestCredentialImportPreviewDoesNotWrite(t *testing.T) {
	ts, _, home := cleanupServer(t)
	writeClaudeLogin(t, home, "access-already", "refresh-already")
	first := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code"}, 201)

	writeClaudeLogin(t, home, "access-other", "refresh-other")
	pre := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code", "preview": true}, 200)
	if pre["preview"] != true || pre["created"] != false {
		t.Fatalf("preview = %v, want the replace reported", pre)
	}
	existing, _ := pre["existing"].(map[string]any)
	if existing == nil || existing["id"] != first["id"] {
		t.Fatalf("existing = %v, want the row the write would replace", existing)
	}
	if id, _ := pre["identity"].(string); id != "" {
		t.Fatalf("unnamed login reported as %q", id)
	}
	if who, _ := pre["who"].(string); who != "" {
		t.Fatalf("unnamed login answered who %q", who)
	}
	rows, _ := credentials.Default().Accounts("anthropic")
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want the vault untouched by a preview", len(rows))
	}
	if row, ok, _ := credentials.Default().Row("anthropic", first["id"].(string)); !ok || !strings.Contains(string(row.Cred), "access-already") {
		t.Fatal("preview changed the saved credential")
	}

	// The real write, named: naming re-keys the row that holds the live
	// login. Two rows only appear once BOTH logins have been seen — the
	// first was already replaced in the file at this point.
	named := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code", "as": "other"}, 201)
	if named["id"] == first["id"] {
		t.Fatal("the named login should take the new key")
	}
	if rows, _ := credentials.Default().Accounts("anthropic"); len(rows) != 1 {
		t.Fatalf("rows = %d, want the live login named, not copied", len(rows))
	}
}

// "Keep both": the login already saved is named from what the vault knows
// and the new login lands as its own row — two rows, the live file matching
// the new one, the old one kept for Use.
func TestCredentialImportKeepKeepsBothLogins(t *testing.T) {
	ts, _, home := cleanupServer(t)
	writeClaudeLogin(t, home, "access-first", "refresh-first")
	cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code"}, 201)

	// The CLI signs into the second account.
	writeClaudeLogin(t, home, "access-second", "refresh-second")
	// The pane's preview flags the replace, and the person keeps both.
	cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code", "preview": true}, 200)
	kept := cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code", "keep": true}, 201)

	provider := rosterFor(t, ts, "claude-code", "anthropic")
	accounts := provider["accounts"].([]any)
	if len(accounts) != 2 {
		t.Fatalf("accounts = %d, want the old login kept and the new one filed", len(accounts))
	}
	live := 0
	for _, a := range accounts {
		row := a.(map[string]any)
		if row["active"] == true {
			live++
			if got, _ := row["origin"].(string); got != "imported:claude-code" {
				t.Fatalf("active row = %v, want the new login", row)
			}
		}
	}
	if live != 1 {
		t.Fatalf("active rows = %d, want exactly the live file's login", live)
	}
	if kept["created"] != true {
		t.Fatalf("keep = %v, want the new login filed as its own row", kept)
	}
}

// A vendor renewal flows into the saved row on a plain roster read: same
// refresh token, newer access pair, the row's masked hint moves to the new
// access — and the row stays matched as the live one.
func TestCredentialRosterHarvestsRenewal(t *testing.T) {
	ts, _, home := cleanupServer(t)
	writeClaudeLogin(t, home, "access-a1", "refresh-stable")
	cliRequest(t, ts, "POST", "/api/credentials/import", map[string]any{"cli": "claude-code"}, 201)

	savedAccess := func() string {
		provider := rosterFor(t, ts, "claude-code", "anthropic")
		row := provider["accounts"].([]any)[0].(map[string]any)
		active, _ := row["active"].(bool)
		id, _ := row["id"].(string)
		r, ok, err := credentials.Default().Row("anthropic", id)
		if err != nil || !ok {
			t.Fatalf("row %v missing: %v", id, err)
		}
		var cred struct {
			Access string `json:"access"`
		}
		if err := json.Unmarshal(r.Cred, &cred); err != nil {
			t.Fatal(err)
		}
		return fmt.Sprint(active, " ", cred.Access)
	}
	before := savedAccess()
	if !strings.Contains(before, "access-a1") {
		t.Fatalf("saved = %q, want the first login", before)
	}

	// The vendor renewed: same refresh token, newer access pair.
	writeClaudeLogin(t, home, "access-a2", "refresh-stable")
	cliRequest(t, ts, "GET", "/api/credentials?cli=claude-code", nil, 200)
	after := savedAccess()

	if !strings.Contains(after, "access-a2") {
		t.Fatalf("after a roster read the saved row = %q, want the renewal harvested", after)
	}
	if active := strings.Contains(after, "true"); !active {
		t.Fatal("after a renewal the row must still match the live file")
	}
}
