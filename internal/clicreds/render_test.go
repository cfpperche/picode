package clicreds

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Synthetic expiries, far in the future and belonging to nothing: the vault
// counts milliseconds, and the two ISO files carry RFC 3339. isoMS is the same
// instant as isoExpiry — the round trip must not move it.
const (
	vaultExpiry = int64(1799000000000)
	isoExpiry   = "2027-01-02T03:04:05.123Z"
)

var isoMS = func() int64 {
	ts, err := time.Parse(time.RFC3339Nano, isoExpiry)
	if err != nil {
		panic(err)
	}
	return ts.UnixMilli()
}()

// The two vault shapes, synthetic values only. The variants below match what
// the file each one lands in can hold: Claude Code, Grok and Antigravity have
// no field for an account id, and Codex and Hermes carry no expiry — a render
// that invented one of those keys would be writing a shape the CLI does not
// read, so the vault's extra field stays in the vault.
var (
	apiKeyCred = json.RawMessage(`{"type":"api_key","key":"sk-ant-synthetic"}`)
	oauthCred  = json.RawMessage(`{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":` +
		itoa(vaultExpiry) + `,"accountId":"acct-1"}`)
	noAccountCred = json.RawMessage(`{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":` +
		itoa(vaultExpiry) + `}`)
	noExpiryCred = json.RawMessage(
		`{"type":"oauth","access":"access-1","refresh":"refresh-1","accountId":"acct-1"}`)
	// isoCred is the same login as noAccountCred, one instant later and to the
	// millisecond: the value Antigravity's RFC 3339 expiry must not round off.
	isoCred = json.RawMessage(`{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":` +
		itoa(isoMS) + `}`)
)

// renderable is one credential each renderer can hold, with the file it needs
// to hold it — Grok's store has no session to update in an empty file. The
// refusal sweeps below render it, so a false proves the *file* was refused
// rather than the credential, and the coverage test keeps the table in step
// with the formats parse.go declares.
var renderable = map[string]struct {
	cred     json.RawMessage
	existing string
}{
	"pi":       {cred: apiKeyCred},
	"opencode": {cred: json.RawMessage(`{"type":"api_key","key":"opencode-key"}`)},
	"claude":   {cred: oauthCred},
	"codex":    {cred: json.RawMessage(`{"type":"api_key","key":"sk-codex-synthetic"}`)},
	"grok":     {cred: oauthCred, existing: `{"https://auth.x.ai::cli-1":{"key":"old-access","refresh_token":"old-refresh"}}`},
	"hermes":   {cred: oauthCred},
	"muse":     {cred: json.RawMessage(`{"type":"api_key","key":"meta-key-synthetic"}`)},
	"agy":      {cred: oauthCred},
}

// outDoc decodes a rendered file the way a test compares it: plain JSON, so
// values are compared, not Go types.
func outDoc(t *testing.T, out []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("rendered file is not JSON: %v (%s)", err, out)
	}
	return doc
}

// dig walks one path of keys into a decoded document; a step that is absent or
// not an object reads nil. Paths are key lists rather than dotted strings
// because the files name their entries with dots of their own (Grok's
// "https://auth.x.ai::cli-1", Claude Code's "plugin:linear|abc").
func dig(doc any, path []string) any {
	for _, key := range path {
		obj, ok := doc.(map[string]any)
		if !ok {
			return nil
		}
		doc = obj[key]
	}
	return doc
}

// TestRenderLoginRoundTrip is the contract with parse.go: whatever RenderLogin
// writes, parseLogin reads back as the same account, in the same shape — for
// every format, both kinds where the format holds them, and with the file's
// other content in place.
func TestRenderLoginRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		format   string
		provider string
		cred     json.RawMessage
		existing string
		label    string
	}{
		{
			name: "pi api key into an empty file", format: "pi", provider: "anthropic",
			cred: apiKeyCred,
		},
		{
			name: "pi api key into a blank file", format: "pi", provider: "anthropic",
			cred:     apiKeyCred,
			existing: "\n",
		},
		{
			name: "pi api key beside another provider", format: "pi", provider: "anthropic",
			cred:     apiKeyCred,
			existing: `{"google":{"type":"api_key","key":"google-key"}}`,
		},
		{
			name: "pi oauth over an existing key", format: "pi", provider: "openai-codex",
			cred:     oauthCred,
			existing: `{"anthropic":{"type":"api_key","key":"sk-ant-synthetic"}}`,
		},
		{
			name: "opencode api key under the CLI's own id", format: "opencode", provider: "zai",
			cred:     json.RawMessage(`{"type":"api_key","key":"zhipu-key-synthetic"}`),
			existing: `{"zai-coding-plan":{"type":"api","key":"zhipu-old"},"openai":{"type":"oauth","access":"old-access","refresh":"old-refresh"}}`,
		},
		{
			name: "opencode oauth under the CLI's own id", format: "opencode", provider: "openai-codex",
			cred:     oauthCred,
			existing: `{"openai":{"type":"oauth","access":"old-access","refresh":"old-refresh"}}`,
		},
		{
			name: "claude oauth", format: "claude", provider: "anthropic",
			cred:     noAccountCred,
			existing: `{"claudeAiOauth":{"accessToken":"old-access","refreshToken":"old-refresh","expiresAt":1,"subscriptionType":"max"},"mcpOAuth":{"plugin:linear|abc":{"accessToken":"plugin-token"}}}`,
		},
		{
			name: "codex api key", format: "codex", provider: "openai-codex",
			cred:     json.RawMessage(`{"type":"api_key","key":"sk-codex-synthetic"}`),
			existing: `{"tokens":{"id_token":"id-1"}}`,
		},
		{
			name: "codex chatgpt login", format: "codex", provider: "openai-codex",
			cred:     noExpiryCred,
			existing: `{"auth_mode":"apikey","OPENAI_API_KEY":"old-key","tokens":{"id_token":"id-1","access_token":"old-access","refresh_token":"old-refresh"}}`,
		},
		{
			name: "grok oauth keeps the session key", format: "grok", provider: "xai",
			cred: noAccountCred,
			existing: `{"https://auth.x.ai::cli-1":{"key":"old-access","auth_mode":"oidc","refresh_token":"old-refresh",` +
				`"expires_at":"2026-01-02T03:04:05Z","email":"who@example.com","user_id":"u-1"}}`,
			label: "who@example.com",
		},
		{
			name: "hermes oauth under the CLI's own id", format: "hermes", provider: "xai",
			cred:     noExpiryCred,
			existing: `{"version":2,"providers":{"xai-oauth":{"tokens":{"access_token":"old-access","refresh_token":"old-refresh"},"auth_mode":"oidc"}},"credential_pool":{"xai":[]}}`,
		},
		{
			name: "hermes oauth under the vault id", format: "hermes", provider: "openai-codex",
			cred:     noExpiryCred,
			existing: `{"version":2,"providers":{"google":{"tokens":{"access_token":"other"}}}}`,
		},
		{
			name: "muse api key", format: "muse", provider: "meta-ai",
			cred:     json.RawMessage(`{"type":"api_key","key":"meta-key-synthetic"}`),
			existing: `{"schema_version":1,"providers":{"meta":{"api_key":"old-key"}}}`,
		},
		{
			name: "agy oauth with a millisecond expiry", format: "agy", provider: "google",
			cred:     isoCred,
			existing: `{"token":{"access_token":"old-access","token_type":"Bearer","refresh_token":"old-refresh","expiry":"2026-01-02T03:04:05Z"},"auth_method":"consumer","id_token":"id-1"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := RenderLogin(tc.format, tc.provider, tc.cred, []byte(tc.existing))
			if !ok {
				t.Fatalf("RenderLogin(%q, %q) refused a credential it declares", tc.format, tc.provider)
			}
			login, ok := parseLogin(tc.format, tc.provider, out)
			if !ok {
				t.Fatalf("parseLogin(%q, %q) found no login in %s", tc.format, tc.provider, out)
			}
			wantKind, _ := cred(t, tc.cred)["type"].(string)
			if login.Provider != tc.provider || login.Kind != wantKind || login.Label != tc.label {
				t.Fatalf("read back %s/%s label %q, want %s/%s label %q",
					login.Provider, login.Kind, login.Label, tc.provider, wantKind, tc.label)
			}
			if got, want := cred(t, login.Cred), cred(t, tc.cred); !reflect.DeepEqual(got, want) {
				t.Fatalf("cred round trip = %v, want %v", got, want)
			}
		})
	}
}

// TestRenderLoginPreservesTheFile holds every renderer to the other half of the
// promise: the keys PiCode does not own survive the write — the file's other
// providers, the object's other fields, its unknown metadata. Every keep path
// must also hold something in the fixture, so a case cannot pass by asserting
// an absence.
func TestRenderLoginPreservesTheFile(t *testing.T) {
	grokEntry := []string{"https://auth.x.ai::cli-1"}
	grok := func(key string) []string { return append(append([]string{}, grokEntry...), key) }

	cases := []struct {
		name     string
		format   string
		provider string
		cred     json.RawMessage
		existing string
		// keep is every path the fixture carries and the render must not touch.
		keep [][]string
		// want is the one path the render must have written, and its value.
		want  []string
		value string
		// absent is every path that must not appear: a second entry for a
		// provider the file already names the CLI's way.
		absent [][]string
	}{
		{
			name: "pi", format: "pi", provider: "xai",
			cred:     json.RawMessage(`{"type":"api_key","key":"xai-key-synthetic"}`),
			existing: `{"anthropic":{"type":"api_key","key":"sk-ant-synthetic"},"_meta":{"rateLimits":{"tier":"plus"},"sessionCount":3}}`,
			keep:     [][]string{{"anthropic"}, {"_meta"}, {"_meta", "rateLimits", "tier"}, {"_meta", "sessionCount"}},
			want:     []string{"xai", "key"}, value: "xai-key-synthetic",
		},
		{
			name: "opencode", format: "opencode", provider: "zai",
			cred:     json.RawMessage(`{"type":"api_key","key":"zhipu-key-synthetic"}`),
			existing: `{"zai-coding-plan":{"type":"api","key":"zhipu-old"},"openai":{"type":"oauth","access":"old-access","refresh":"old-refresh"},"_meta":{"version":3}}`,
			keep:     [][]string{{"openai"}, {"openai", "access"}, {"_meta", "version"}},
			want:     []string{"zai-coding-plan", "key"}, value: "zhipu-key-synthetic",
			absent: [][]string{{"zai"}},
		},
		{
			name: "claude", format: "claude", provider: "anthropic",
			cred: oauthCred,
			existing: `{"claudeAiOauth":{"accessToken":"old-access","refreshToken":"old-refresh","expiresAt":1,` +
				`"subscriptionType":"max","scopes":["user:inference"]},"mcpOAuth":{"plugin:linear|abc":{"accessToken":"plugin-token","serverName":"linear"}},"_meta":{"install":1}}`,
			keep: [][]string{
				{"claudeAiOauth", "subscriptionType"}, {"claudeAiOauth", "scopes"},
				{"mcpOAuth", "plugin:linear|abc", "accessToken"}, {"_meta", "install"},
			},
			want: []string{"claudeAiOauth", "accessToken"}, value: "access-1",
		},
		{
			name: "codex api key", format: "codex", provider: "openai-codex",
			cred:     json.RawMessage(`{"type":"api_key","key":"sk-codex-synthetic"}`),
			existing: `{"OPENAI_API_KEY":"old-key","tokens":{"id_token":"id-1","access_token":"old-access","refresh_token":"old-refresh"},"last_refresh":"2026-01-02T03:04:05Z","sandbox":{"mode":"workspace-write"}}`,
			keep:     [][]string{{"tokens"}, {"last_refresh"}, {"sandbox"}, {"sandbox", "mode"}},
			want:     []string{"OPENAI_API_KEY"}, value: "sk-codex-synthetic",
		},
		{
			name: "codex chatgpt login", format: "codex", provider: "openai-codex",
			cred:     oauthCred,
			existing: `{"OPENAI_API_KEY":"old-key","tokens":{"id_token":"id-1"},"sandbox":{"mode":"workspace-write"}}`,
			keep:     [][]string{{"tokens", "id_token"}, {"sandbox", "mode"}},
			want:     []string{"tokens", "access_token"}, value: "access-1",
		},
		{
			name: "grok", format: "grok", provider: "xai",
			cred: oauthCred,
			existing: `{"https://auth.x.ai::cli-1":{"key":"old-access","auth_mode":"oidc","refresh_token":"old-refresh",` +
				`"expires_at":"2026-01-02T03:04:05Z","email":"who@example.com","user_id":"u-1",` +
				`"oidc_issuer":"https://auth.x.ai","oidc_client_id":"cli-1"}}`,
			keep: [][]string{grok("oidc_issuer"), grok("oidc_client_id"), grok("user_id"), grok("email"), grok("auth_mode")},
			want: grok("key"), value: "access-1",
		},
		{
			name: "hermes", format: "hermes", provider: "xai",
			cred: oauthCred,
			existing: `{"version":2,"providers":{"xai-oauth":{"tokens":{"id_token":"id-1","access_token":"old-access",` +
				`"refresh_token":"old-refresh","token_type":"Bearer"},"auth_mode":"oidc","last_refresh":"2026-01-02T03:04:05Z"},` +
				`"google":{"tokens":{"access_token":"other"}}},"credential_pool":{"xai":[{"id":"c-1","auth_type":"api_key","key":"pool-key"}]}}`,
			keep: [][]string{
				{"version"}, {"providers", "xai-oauth", "tokens", "id_token"},
				{"providers", "xai-oauth", "tokens", "token_type"}, {"providers", "xai-oauth", "auth_mode"},
				{"providers", "xai-oauth", "last_refresh"}, {"providers", "google"}, {"credential_pool"},
			},
			want: []string{"providers", "xai-oauth", "tokens", "access_token"}, value: "access-1",
			absent: [][]string{{"providers", "xai"}},
		},
		{
			name: "muse", format: "muse", provider: "meta-ai",
			cred:     json.RawMessage(`{"type":"api_key","key":"meta-key-synthetic"}`),
			existing: `{"schema_version":1,"providers":{"meta":{"api_key":"old-key","base_url":"https://api.meta.ai"},"other":{"api_key":"other-key"}}}`,
			keep:     [][]string{{"schema_version"}, {"providers", "meta", "base_url"}, {"providers", "other"}},
			want:     []string{"providers", "meta", "api_key"}, value: "meta-key-synthetic",
		},
		{
			name: "agy", format: "agy", provider: "google",
			cred:     oauthCred,
			existing: `{"token":{"access_token":"old-access","token_type":"Bearer","refresh_token":"old-refresh","expiry":"2026-01-02T03:04:05Z"},"auth_method":"consumer","id_token":"id-1","_meta":{"install":"old"}}`,
			keep:     [][]string{{"auth_method"}, {"id_token"}, {"token", "token_type"}, {"_meta"}},
			want:     []string{"token", "access_token"}, value: "access-1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := RenderLogin(tc.format, tc.provider, tc.cred, []byte(tc.existing))
			if !ok {
				t.Fatalf("RenderLogin(%q, %q) refused", tc.format, tc.provider)
			}
			doc, before := outDoc(t, out), outDoc(t, []byte(tc.existing))
			if got := dig(doc, tc.want); got != tc.value {
				t.Fatalf("%v = %v, want %q", tc.want, got, tc.value)
			}
			for _, path := range tc.keep {
				want := dig(before, path)
				if want == nil {
					t.Fatalf("the fixture carries nothing at %v — the case proves nothing", path)
				}
				if got := dig(doc, path); !reflect.DeepEqual(got, want) {
					t.Fatalf("%v = %v, want the fixture's %v", path, got, want)
				}
			}
			for _, path := range tc.absent {
				if got := dig(doc, path); got != nil {
					t.Fatalf("%v = %v, must not be written", path, got)
				}
			}
		})
	}
}

// TestRenderLoginWritesOnlyWhatTheFileHolds pins what the round-trip fixtures
// above are shaped around: a vault field the CLI's own file has no place for is
// not invented for it. Claude Code, Grok and Antigravity store no account id,
// and Codex and Hermes store no expiry, so a credential carrying one renders
// without it — the vault row keeps the fact, the file stays the shape the CLI
// reads.
func TestRenderLoginWritesOnlyWhatTheFileHolds(t *testing.T) {
	grokSession := `{"https://auth.x.ai::cli-1":{"key":"old-access","refresh_token":"old-refresh"}}`
	cases := []struct {
		name     string
		format   string
		provider string
		existing string
		// absent is the vault value that must not appear in the file.
		absent string
	}{
		{name: "claude has no account id", format: "claude", provider: "anthropic", absent: "acct-1"},
		{name: "grok has no account id", format: "grok", provider: "xai", existing: grokSession, absent: "acct-1"},
		{name: "agy has no account id", format: "agy", provider: "google", absent: "acct-1"},
		{name: "codex has no expiry", format: "codex", provider: "openai-codex", absent: itoa(vaultExpiry)},
		{name: "hermes has no expiry", format: "hermes", provider: "xai", absent: itoa(vaultExpiry)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, ok := RenderLogin(tc.format, tc.provider, oauthCred, []byte(tc.existing))
			if !ok {
				t.Fatalf("RenderLogin(%q, %q) refused", tc.format, tc.provider)
			}
			if strings.Contains(string(out), tc.absent) {
				t.Fatalf("the file gained %q:\n%s", tc.absent, out)
			}
			if !strings.Contains(string(out), "access-1") {
				t.Fatalf("the tokens were not written:\n%s", out)
			}
		})
	}
}

// TestRenderLoginSpeaksEachCLIsDialect: the entry is spelled the way that CLI
// spells it, not the way the vault does. pi reads "api_key" and OpenCode "api"
// for the same key (parseProviderMap accepts both), and the file being written
// is the CLI's own — a spelling OpenCode does not read would make the write do
// nothing. oauth needs no translation: both read "oauth".
func TestRenderLoginSpeaksEachCLIsDialect(t *testing.T) {
	for _, tc := range []struct{ format, want string }{
		{format: "pi", want: KindAPIKey},
		{format: "opencode", want: "api"},
	} {
		out, ok := RenderLogin(tc.format, "anthropic", apiKeyCred, nil)
		if !ok {
			t.Fatalf("RenderLogin(%q, anthropic) refused an api key", tc.format)
		}
		if got := dig(outDoc(t, out), []string{"anthropic", "type"}); got != tc.want {
			t.Fatalf("%s writes an api key as type %v, want %q", tc.format, got, tc.want)
		}
	}
	out, ok := RenderLogin("opencode", "anthropic", oauthCred, nil)
	if !ok {
		t.Fatal("RenderLogin(opencode, anthropic) refused an oauth row")
	}
	if got := dig(outDoc(t, out), []string{"anthropic", "type"}); got != KindOAuth {
		t.Fatalf("opencode writes an oauth row as type %v, want %q", got, KindOAuth)
	}
}

// TestRenderLoginWritesMillisecondExpiries pins the two number-valued expiry
// fields (Claude Code's expiresAt, pi's and OpenCode's expires) to the vault's
// own milliseconds. parseLogin's seconds-or-milliseconds split would forgive a
// renderer that divided by 1000 — the CLI would not: it reads the login as long
// expired and goes off to refresh a token it was just handed. A credential with
// no expiry writes no field at all, because expires 0 means the same thing.
func TestRenderLoginWritesMillisecondExpiries(t *testing.T) {
	for _, tc := range []struct {
		format, provider string
		path             []string
	}{
		{format: "claude", provider: "anthropic", path: []string{"claudeAiOauth", "expiresAt"}},
		{format: "pi", provider: "anthropic", path: []string{"anthropic", "expires"}},
		{format: "opencode", provider: "anthropic", path: []string{"anthropic", "expires"}},
	} {
		cred := oauthCred
		if tc.format == "claude" {
			cred = noAccountCred
		}
		out, ok := RenderLogin(tc.format, tc.provider, cred, nil)
		if !ok {
			t.Fatalf("RenderLogin(%q, %q) refused", tc.format, tc.provider)
		}
		if got := dig(outDoc(t, out), tc.path); got != float64(vaultExpiry) {
			t.Fatalf("%s %v = %v, want the vault's milliseconds %d", tc.format, tc.path, got, vaultExpiry)
		}
	}

	noExpiry := json.RawMessage(`{"type":"oauth","access":"access-1","refresh":"refresh-1"}`)
	for _, format := range []string{"claude", "pi"} {
		out, ok := RenderLogin(format, "anthropic", noExpiry, nil)
		if !ok {
			t.Fatalf("RenderLogin(%q, anthropic) refused an oauth row without an expiry", format)
		}
		if strings.Contains(string(out), "expire") {
			t.Fatalf("%s invented an expiry for a credential that has none:\n%s", format, out)
		}
	}
}

// TestRenderLoginRefuses sweeps the shapes that must be refused: an unknown
// format (omp has no credential file at all), a kind the format's file cannot
// hold, and a credential that is not whole — the same refusals the readers
// make, so nothing half-usable is ever written.
func TestRenderLoginRefuses(t *testing.T) {
	cases := []struct {
		name     string
		format   string
		provider string
		cred     json.RawMessage
		existing string
	}{
		{name: "unknown format", format: "nonesuch", provider: "anthropic", cred: apiKeyCred},
		{name: "omp", format: "omp", provider: "anthropic", cred: apiKeyCred},
		{name: "empty format", provider: "anthropic", cred: apiKeyCred},
		{name: "claude api key", format: "claude", provider: "anthropic", cred: apiKeyCred},
		{name: "hermes api key", format: "hermes", provider: "xai", cred: json.RawMessage(`{"type":"api_key","key":"xai-key-synthetic"}`)},
		{name: "agy api key", format: "agy", provider: "google", cred: json.RawMessage(`{"type":"api_key","key":"google-key"}`)},
		{name: "muse oauth", format: "muse", provider: "meta-ai", cred: oauthCred},
		{
			name: "grok api key", format: "grok", provider: "xai",
			cred:     json.RawMessage(`{"type":"api_key","key":"xai-key-synthetic"}`),
			existing: `{"https://auth.x.ai::cli-1":{"key":"old-access","refresh_token":"old-refresh"}}`,
		},
		{name: "grok with no session yet", format: "grok", provider: "xai", cred: oauthCred},
		{
			name: "grok entry not keyed by issuer", format: "grok", provider: "xai",
			cred:     oauthCred,
			existing: `{"session":{"key":"old-access","refresh_token":"old-refresh"}}`,
		},
		{name: "api key without a key", format: "pi", provider: "anthropic", cred: json.RawMessage(`{"type":"api_key","key":"   "}`)},
		{name: "api key with no key field", format: "pi", provider: "anthropic", cred: json.RawMessage(`{"type":"api_key"}`)},
		{name: "oauth without a refresh token", format: "pi", provider: "openai-codex", cred: json.RawMessage(`{"type":"oauth","access":"access-1"}`)},
		{name: "oauth without an access token", format: "pi", provider: "openai-codex", cred: json.RawMessage(`{"type":"oauth","refresh":"refresh-1"}`)},
		{name: "unknown credential type", format: "pi", provider: "anthropic", cred: json.RawMessage(`{"type":"token","key":"k"}`)},
		{name: "credential without a type", format: "pi", provider: "anthropic", cred: json.RawMessage(`{"key":"k"}`)},
		{name: "empty credential", format: "pi", provider: "anthropic", cred: json.RawMessage(`{}`)},
		{name: "null credential", format: "pi", provider: "anthropic", cred: json.RawMessage(`null`)},
		{name: "string credential", format: "pi", provider: "anthropic", cred: json.RawMessage(`"a string"`)},
		{name: "malformed credential", format: "pi", provider: "anthropic", cred: json.RawMessage(`{"type":`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out, ok := RenderLogin(tc.format, tc.provider, tc.cred, []byte(tc.existing)); ok {
				t.Fatalf("RenderLogin(%q, %q) wrote %s, want a refusal", tc.format, tc.provider, out)
			}
		})
	}
}

// TestRenderLoginRefusesUnreadableFiles sweeps every renderer with files that
// are not one JSON object — malformed bytes, an array, a scalar, two documents
// in a row — each holding a credential that format can render, so the only
// reason left to refuse is the file. A file PiCode cannot read may hold the
// person's login, so it is never clobbered.
func TestRenderLoginRefusesUnreadableFiles(t *testing.T) {
	bad := []string{`{`, `{"a":`, `[]`, `null`, `"a string"`, `42`, `not json`, `{"a":1} {"b":2}`}
	for _, format := range formats {
		r, ok := renderable[format]
		if !ok {
			t.Fatalf("%s has no renderable credential in the table", format)
		}
		for _, existing := range bad {
			if out, ok := RenderLogin(format, "anthropic", r.cred, []byte(existing)); ok {
				t.Fatalf("RenderLogin(%q, existing %q) wrote %s", format, existing, out)
			}
		}
	}
}

// TestRenderLoginCoversEveryFormat keeps the renderable table in step with the
// formats the package reads: a reader is added to parse.go, the sweeps above
// must not silently skip a renderer, and each format's own credential must
// render.
func TestRenderLoginCoversEveryFormat(t *testing.T) {
	if len(renderable) != len(formats) {
		t.Fatalf("%d renderable credentials for %d formats", len(renderable), len(formats))
	}
	for _, format := range formats {
		r, ok := renderable[format]
		if !ok {
			t.Fatalf("%s has no renderable credential in the table", format)
		}
		if _, ok := RenderLogin(format, "anthropic", r.cred, []byte(r.existing)); !ok {
			t.Fatalf("RenderLogin(%q, anthropic) refused", format)
		}
	}
}

// TestRenderLoginWritesCodexModes covers the one file where the mode is part of
// the credential: auth_mode must follow the login written, and a key left in the
// file must stop winning over the tokens — otherwise the activation would do
// nothing at all.
func TestRenderLoginWritesCodexModes(t *testing.T) {
	key := json.RawMessage(`{"type":"api_key","key":"sk-codex-synthetic"}`)
	existing := []byte(`{"OPENAI_API_KEY":"old-key","tokens":{"id_token":"id-1","access_token":"old-access",` +
		`"refresh_token":"old-refresh"},"last_refresh":"2026-01-02T03:04:05Z","sandbox":{"mode":"workspace-write"}}`)

	out, ok := RenderLogin("codex", "openai-codex", key, existing)
	if !ok {
		t.Fatal("an api key into Codex's file was refused")
	}
	doc := outDoc(t, out)
	if got := dig(doc, []string{"auth_mode"}); got != "apikey" {
		t.Fatalf("auth_mode = %v, want \"apikey\"", got)
	}
	if got := dig(doc, []string{"OPENAI_API_KEY"}); got != "sk-codex-synthetic" {
		t.Fatalf("OPENAI_API_KEY = %v", got)
	}
	if got := dig(doc, []string{"tokens", "id_token"}); got != "id-1" {
		t.Fatalf("the key render dropped the file's id_token: %v", got)
	}

	// The switch that matters in practice: a machine holding an API key now
	// activates a ChatGPT account.
	start := time.Now()
	out, ok = RenderLogin("codex", "openai-codex", oauthCred, out)
	if !ok {
		t.Fatal("a ChatGPT login into Codex's file was refused")
	}
	doc = outDoc(t, out)
	if got := dig(doc, []string{"auth_mode"}); got != "chatgpt" {
		t.Fatalf("auth_mode = %v, want \"chatgpt\"", got)
	}
	if got, present := doc["OPENAI_API_KEY"]; !present || got != nil {
		t.Fatalf("OPENAI_API_KEY = %v (present %v), want an explicit null: a key left here wins over the tokens", got, present)
	}
	if got := dig(doc, []string{"tokens", "access_token"}); got != "access-1" {
		t.Fatalf("tokens.access_token = %v", got)
	}
	if got := dig(doc, []string{"tokens", "id_token"}); got != "id-1" {
		t.Fatalf("the oauth render dropped the file's id_token: %v", got)
	}
	stamp, _ := dig(doc, []string{"last_refresh"}).(string)
	refreshed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		t.Fatalf("last_refresh = %q, want RFC 3339: %v", stamp, err)
	}
	if refreshed.Before(start.Add(-time.Minute)) {
		t.Fatalf("last_refresh = %q, want the activation's own time", stamp)
	}
}

// TestRenderLoginIsStableJSON pins the output form: two-space indented JSON
// with a trailing newline and sorted keys, identical for identical inputs. It is
// a diff, not a contract — the CLI re-reads the file — but an unstable one would
// rewrite the file on every activation.
func TestRenderLoginIsStableJSON(t *testing.T) {
	out, ok := RenderLogin("pi", "anthropic", apiKeyCred, nil)
	if !ok {
		t.Fatal("RenderLogin refused a pi api key")
	}
	want := "{\n  \"anthropic\": {\n    \"key\": \"sk-ant-synthetic\",\n    \"type\": \"api_key\"\n  }\n}\n"
	if string(out) != want {
		t.Fatalf("rendered file =\n%s\nwant\n%s", out, want)
	}
	again, ok := RenderLogin("pi", "anthropic", apiKeyCred, nil)
	if !ok || !bytes.Equal(out, again) {
		t.Fatalf("the same inputs rendered differently:\n%s\n%s", out, again)
	}
}

// TestRenderLoginKeepsNumberLiterals covers the values PiCode does not own: a
// file's number is written back as it was read, not through float64 — a large
// id or a nanosecond timestamp must not come back rounded.
func TestRenderLoginKeepsNumberLiterals(t *testing.T) {
	existing := []byte(`{"_meta":{"build":12345678901234567890,"ratio":1.50}}`)
	out, ok := RenderLogin("pi", "anthropic", apiKeyCred, existing)
	if !ok {
		t.Fatal("RenderLogin refused a pi api key")
	}
	for _, literal := range []string{"12345678901234567890", "1.50"} {
		if !strings.Contains(string(out), literal) {
			t.Fatalf("the rendered file lost the number %s:\n%s", literal, out)
		}
	}
}
