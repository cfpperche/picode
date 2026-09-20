package clicreds

import (
	"encoding/json"
	"reflect"
	"strconv"
	"testing"
	"time"
)

// itoa renders a millisecond expectation inside a fixture string.
func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// formats is every reader parseLogin dispatches to, the list the garbage and
// no-credential sweeps below run against.
var formats = []string{"pi", "opencode", "claude", "codex", "grok", "hermes", "muse", "agy"}

func cred(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("cred is not JSON: %v (%s)", err, raw)
	}
	return got
}

func wantJSON(t *testing.T, s string) map[string]any {
	t.Helper()
	var want map[string]any
	if err := json.Unmarshal([]byte(s), &want); err != nil {
		t.Fatalf("bad expectation %q: %v", s, err)
	}
	return want
}

// TestParseLogin is the per-format table: every shape the nine CLIs write on
// this machine, synthetic values only, plus the shapes that must be refused.
func TestParseLogin(t *testing.T) {
	const (
		grokExpiry = "2027-01-02T03:04:05.123456789Z"
		agyExpiry  = "2027-01-02T03:04:05.123456789-03:00"
	)
	at := func(s string) int64 {
		ts, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t.Fatalf("fixture timestamp %q: %v", s, err)
		}
		return ts.UnixMilli()
	}

	cases := []struct {
		name         string
		format       string
		provider     string
		raw          string
		wantOK       bool
		wantProvider string
		wantKind     string
		wantLabel    string
		wantCred     string
	}{
		{
			name: "pi api key", format: "pi", provider: "anthropic",
			raw:    `{"anthropic":{"type":"api_key","key":"sk-ant-synthetic"}}`,
			wantOK: true, wantProvider: "anthropic", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"sk-ant-synthetic"}`,
		},
		{
			name: "pi oauth with millisecond expiry", format: "pi", provider: "openai-codex",
			raw:    `{"openai-codex":{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799000000000,"accountId":"acct-1"}}`,
			wantOK: true, wantProvider: "openai-codex", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799000000000,"accountId":"acct-1"}`,
		},
		{
			name: "pi oauth with second-resolution expiry", format: "pi", provider: "xai",
			raw:    `{"xai":{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799000000}}`,
			wantOK: true, wantProvider: "xai", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799000000000}`,
		},
		{
			name: "pi oauth without a refresh token", format: "pi", provider: "xai",
			raw: `{"xai":{"type":"oauth","access":"access-1","expires":1799000000}}`,
		},
		{
			name: "pi entry without a key", format: "pi", provider: "zai",
			raw: `{"zai":{"type":"api_key"}}`,
		},
		{
			name: "pi unknown entry type", format: "pi", provider: "anthropic",
			raw: `{"anthropic":{"type":"cookie","key":"k"}}`,
		},
		{
			name: "pi without the asked provider", format: "pi", provider: "anthropic",
			raw: `{"google":{"type":"api_key","key":"google-key"}}`,
		},
		{
			name: "pi truncated file", format: "pi", provider: "anthropic",
			raw: `{"anthropic":`,
		},
		{
			name: "opencode api key under its own id", format: "opencode", provider: "opencode",
			raw:    `{"opencode-go":{"type":"api","key":"opencode-key"}}`,
			wantOK: true, wantProvider: "opencode", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"opencode-key"}`,
		},
		{
			name: "opencode chatgpt oauth under openai", format: "opencode", provider: "openai-codex",
			raw:    `{"openai":{"type":"oauth","refresh":"refresh-1","access":"access-1","expires":1799000000000,"accountId":"acct-2"}}`,
			wantOK: true, wantProvider: "openai-codex", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799000000000,"accountId":"acct-2"}`,
		},
		{
			name: "opencode zai under zai-coding-plan", format: "opencode", provider: "zai",
			raw:    `{"zai-coding-plan":{"type":"api","key":"zhipu-key"}}`,
			wantOK: true, wantProvider: "zai", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"zhipu-key"}`,
		},
		{
			name: "opencode metadata is not a key", format: "opencode", provider: "openrouter",
			raw: `{"openrouter":{"type":"api","metadata":{"accountId":"acct"}}}`,
		},
		{
			name: "claude oauth", format: "claude", provider: "anthropic",
			raw:    `{"claudeAiOauth":{"accessToken":"access-1","refreshToken":"refresh-1","expiresAt":1799100000000,"refreshTokenExpiresAt":1799900000000,"scopes":["user:inference"],"subscriptionType":"max","rateLimitTier":"default"},"mcpOAuth":{"plugin:linear|abc":{"accessToken":"other","serverName":"linear"}}}`,
			wantOK: true, wantProvider: "anthropic", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":1799100000000}`,
		},
		{
			name: "claude without a refresh token", format: "claude", provider: "anthropic",
			raw: `{"claudeAiOauth":{"accessToken":"access-1","expiresAt":1799100000000}}`,
		},
		{
			name: "claude plugin oauth is not the login", format: "claude", provider: "anthropic",
			raw: `{"mcpOAuth":{"plugin:linear|abc":{"accessToken":"plugin-token","serverName":"linear"}}}`,
		},
		{
			name: "codex api key wins over tokens", format: "codex", provider: "openai-codex",
			raw:    `{"auth_mode":"apikey","OPENAI_API_KEY":"sk-codex-synthetic","tokens":{"access_token":"access-1","refresh_token":"refresh-1","account_id":"acct-3"},"last_refresh":"2027-01-02T03:04:05Z"}`,
			wantOK: true, wantProvider: "openai-codex", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"sk-codex-synthetic"}`,
		},
		{
			name: "codex chatgpt tokens", format: "codex", provider: "openai-codex",
			raw:    `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"id-1","access_token":"access-1","refresh_token":"refresh-1","account_id":"acct-3"},"last_refresh":"2027-01-02T03:04:05Z"}`,
			wantOK: true, wantProvider: "openai-codex", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","accountId":"acct-3"}`,
		},
		{
			name: "codex with no key and no tokens", format: "codex", provider: "openai-codex",
			raw: `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"last_refresh":"2027-01-02T03:04:05Z"}`,
		},
		{
			name: "codex tokens without a refresh token", format: "codex", provider: "openai-codex",
			raw: `{"tokens":{"access_token":"access-1","account_id":"acct-3"}}`,
		},
		{
			name: "grok oauth with an ISO expiry", format: "grok", provider: "xai",
			raw:    `{"https://auth.x.ai::cli-1":{"key":"access-1","auth_mode":"oidc","refresh_token":"refresh-1","expires_at":"` + grokExpiry + `","email":"who@example.com","user_id":"u-1"}}`,
			wantOK: true, wantProvider: "xai", wantKind: KindOAuth, wantLabel: "who@example.com",
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":` + itoa(at(grokExpiry)) + `}`,
		},
		{
			name: "grok picks the entry it can use", format: "grok", provider: "xai",
			raw:    `{"https://auth.x.ai::aaa-old":{"key":"access-old","expires_at":"` + grokExpiry + `"},"https://auth.x.ai::newer":{"key":"access-new","refresh_token":"refresh-new","expires_at":"` + grokExpiry + `","email":"new@example.com"}}`,
			wantOK: true, wantProvider: "xai", wantKind: KindOAuth, wantLabel: "new@example.com",
			wantCred: `{"type":"oauth","access":"access-new","refresh":"refresh-new","expires":` + itoa(at(grokExpiry)) + `}`,
		},
		{
			name: "grok entry without a refresh token", format: "grok", provider: "xai",
			raw: `{"https://auth.x.ai::b1a00492":{"key":"access-1","expires_at":"` + grokExpiry + `"}}`,
		},
		{
			name: "grok entry not keyed by issuer", format: "grok", provider: "xai",
			raw: `{"session":{"key":"access-1","refresh_token":"refresh-1"}}`,
		},
		{
			name: "grok truncated file", format: "grok", provider: "xai",
			raw: `{"https://auth.x.ai::b1a00492":`,
		},
		{
			name: "hermes providers oauth", format: "hermes", provider: "openai-codex",
			raw:    `{"version":2,"providers":{"openai-codex":{"tokens":{"id_token":"id-1","access_token":"access-1","refresh_token":"refresh-1","account_id":"acct-4"},"last_refresh":"2027-01-02T03:04:05Z","auth_mode":"chatgpt"}},"credential_pool":{"openai-codex":[]}}`,
			wantOK: true, wantProvider: "openai-codex", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","accountId":"acct-4"}`,
		},
		{
			name: "hermes xai oauth under its own id", format: "hermes", provider: "xai",
			raw:    `{"version":2,"providers":{"xai-oauth":{"tokens":{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600,"token_type":"Bearer"},"auth_mode":"oidc"}}}`,
			wantOK: true, wantProvider: "xai", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1"}`,
		},
		{
			name: "hermes id_token is not an access token", format: "hermes", provider: "xai",
			raw: `{"version":2,"providers":{"xai-oauth":{"tokens":{"id_token":"id-1","expires_in":3600,"token_type":"Bearer"},"auth_mode":"oidc"}}}`,
		},
		{
			name: "hermes pool key", format: "hermes", provider: "zai",
			raw:    `{"version":2,"credential_pool":{"zai":[{"id":"c-1","label":"Work","auth_type":"api_key","priority":0,"key":"zai-key-synthetic","base_url":"https://api.z.ai/api/paas/v4"}]}}`,
			wantOK: true, wantProvider: "zai", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"zai-key-synthetic"}`,
		},
		{
			name: "hermes pool fingerprint is not a key", format: "hermes", provider: "zai",
			raw: `{"version":2,"credential_pool":{"zai":[{"id":"c-1","auth_type":"api_key","secret_fingerprint":"sha256:0f1e2d"}]}}`,
		},
		{
			name: "hermes pool for another provider", format: "hermes", provider: "zai",
			raw: `{"version":2,"credential_pool":{"copilot":[{"id":"c-2","auth_type":"api_key","key":"gh-token"}]}}`,
		},
		{
			name: "hermes with no credential", format: "hermes", provider: "openai-codex",
			raw: `{"version":2,"providers":{},"credential_pool":{"openai-codex":[]}}`,
		},
		{
			name: "muse api key", format: "muse", provider: "meta-ai",
			raw:    `{"schema_version":1,"providers":{"meta":{"api_key":"meta-key-synthetic"}}}`,
			wantOK: true, wantProvider: "meta-ai", wantKind: KindAPIKey,
			wantCred: `{"type":"api_key","key":"meta-key-synthetic"}`,
		},
		{
			name: "muse provider without a key", format: "muse", provider: "meta-ai",
			raw: `{"schema_version":1,"providers":{"meta":{}}}`,
		},
		{
			name: "muse file for another provider only", format: "muse", provider: "meta-ai",
			raw: `{"schema_version":1,"providers":{"openai":{"api_key":"k"}}}`,
		},
		{
			name: "agy oauth with an offset expiry", format: "agy", provider: "google",
			raw:    `{"token":{"access_token":"access-1","token_type":"Bearer","refresh_token":"refresh-1","expiry":"` + agyExpiry + `"},"auth_method":"consumer","id_token":"id-1"}`,
			wantOK: true, wantProvider: "google", wantKind: KindOAuth,
			wantCred: `{"type":"oauth","access":"access-1","refresh":"refresh-1","expires":` + itoa(at(agyExpiry)) + `}`,
		},
		{
			name: "agy without a refresh token", format: "agy", provider: "google",
			raw: `{"token":{"access_token":"access-1","token_type":"Bearer","expiry":"` + agyExpiry + `"},"auth_method":"consumer"}`,
		},
		{
			name: "agy without a token object", format: "agy", provider: "google",
			raw: `{"auth_method":"consumer","id_token":"id-1"}`,
		},
		{
			name: "unknown format", format: "nonesuch", provider: "anthropic",
			raw: `{"anthropic":{"type":"api_key","key":"sk-ant-synthetic"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			login, ok := parseLogin(tc.format, tc.provider, []byte(tc.raw))
			if ok != tc.wantOK {
				t.Fatalf("parseLogin(%q, %q) ok = %v, want %v (login %+v)", tc.format, tc.provider, ok, tc.wantOK, login)
			}
			if !ok {
				return
			}
			if login.Provider != tc.wantProvider || login.Kind != tc.wantKind || login.Label != tc.wantLabel {
				t.Fatalf("login = %s/%s label %q, want %s/%s label %q",
					login.Provider, login.Kind, login.Label, tc.wantProvider, tc.wantKind, tc.wantLabel)
			}
			if got, want := cred(t, login.Cred), wantJSON(t, tc.wantCred); !reflect.DeepEqual(got, want) {
				t.Fatalf("cred = %v, want %v", got, want)
			}
		})
	}
}

// TestParseLoginRejectsGarbage sweeps every reader with payloads that are not a
// credential file: the parsers must answer no login, never panic and never
// synthesize a field.
func TestParseLoginRejectsGarbage(t *testing.T) {
	for _, format := range formats {
		for _, raw := range []string{``, `{`, `[]`, `{}`, `null`, `"a string"`, `42`} {
			if login, ok := parseLogin(format, "anthropic", []byte(raw)); ok {
				t.Fatalf("parseLogin(%q, %q) = %+v, want no login", format, raw, login)
			}
		}
	}
}

// TestExpiresMillis pins the seconds-vs-milliseconds split: the two magnitudes
// the installed CLIs write, the boundary, and the values that mean "unknown".
func TestExpiresMillis(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{float64(0), 0},
		{nil, 0},
		{"1799000000", 0},
		{float64(-1), 0},
		{float64(1e11 - 1), (1e11 - 1) * 1000},
		{float64(1e11), 1e11},
		{float64(1799000000), 1799000000000},
		{float64(1799000000000), 1799000000000},
	}
	for _, tc := range cases {
		if got := expiresMillis(tc.in); got != tc.want {
			t.Fatalf("expiresMillis(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
