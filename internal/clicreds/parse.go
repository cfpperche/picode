package clicreds

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

// parseLogin reads one CLI's own credential file into the vault's shapes —
// {"type":"api_key","key":…} or
// {"type":"oauth","access":…,"refresh":…,"expires":<unix ms>,"accountId":…},
// empty fields omitted. It is a pure function of the bytes: it never invents a
// value, and a file it does not recognize — malformed JSON, a shape it has
// never seen, an entry with no key or no refresh token — is reported as "no
// login" rather than guessed at.
//
// provider is the vault provider id of the row whose Native declared this file
// (clicreds.go's Detect walks the rows). It is the answer for every format
// whose file does not name the provider itself; the provider-map formats
// (pi, opencode) and Hermes' per-provider document use it to pick the entry.
func parseLogin(format, provider string, raw []byte) (Login, bool) {
	switch format {
	case "pi":
		return parseProviderMap(raw, provider, nil)
	case "opencode":
		return parseProviderMap(raw, provider, opencodeIDs)
	case "claude":
		return parseClaude(raw, provider)
	case "codex":
		return parseCodex(raw, provider)
	case "grok":
		return parseGrok(raw, provider)
	case "hermes":
		return parseHermes(raw, provider)
	case "muse":
		return parseMuse(raw, provider)
	case "agy":
		return parseAgy(raw, provider)
	}
	return Login{}, false
}

// opencodeIDs maps a vault provider id to the ids the installed OpenCode
// writes, where the two differ. Observed 2026-09-20 in
// ~/.local/share/opencode/auth.json and ~/.cache/opencode/models.json; an id
// absent here is looked up verbatim.
var opencodeIDs = map[string][]string{
	"openai-codex": {"openai"},
	"zai":          {"zai-coding-plan"},
	"kimi-coding":  {"kimi-code-plan-global", "kimi-code-plan-cn"},
	"meta-ai":      {"meta"},
	"opencode":     {"opencode-go", "opencode-zen"},
}

// hermesIDs is the same map for Hermes' per-provider document: its pool and
// `providers` entries are keyed with its own names (`xai-oauth`, `copilot`,
// `kimi-for-coding`), while the vault id stays pi's.
var hermesIDs = map[string][]string{
	"xai":            {"xai-oauth"},
	"github-copilot": {"copilot"},
	"kimi-coding":    {"kimi-for-coding", "kimi-coding-cn"},
}

// cred is the vault's api-key shape; the field order is the shape the pane and
// the store already read.
type credAPIKey struct {
	Type string `json:"type"`
	Key  string `json:"key"`
}

type credOAuth struct {
	Type      string `json:"type"`
	Access    string `json:"access"`
	Refresh   string `json:"refresh"`
	Expires   int64  `json:"expires,omitempty"`
	AccountID string `json:"accountId,omitempty"`
}

// loginAPIKey builds an api-key Login. No key, no credential: the key is the
// whole fact.
func loginAPIKey(provider, key, label string) (Login, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Login{}, false
	}
	cred, err := json.Marshal(credAPIKey{Type: KindAPIKey, Key: key})
	if err != nil {
		return Login{}, false
	}
	return Login{Provider: provider, Kind: KindAPIKey, Cred: cred, Label: label}, true
}

// loginOAuth builds an oauth Login. A missing refresh token is a missing
// credential — an access token alone expires and cannot be renewed, so it is
// reported as no login, never as a half-usable row.
func loginOAuth(provider, access, refresh string, expires int64, accountID, label string) (Login, bool) {
	access, refresh = strings.TrimSpace(access), strings.TrimSpace(refresh)
	if access == "" || refresh == "" {
		return Login{}, false
	}
	if expires < 0 {
		expires = 0
	}
	cred, err := json.Marshal(credOAuth{
		Type:      KindOAuth,
		Access:    access,
		Refresh:   refresh,
		Expires:   expires,
		AccountID: strings.TrimSpace(accountID),
	})
	if err != nil {
		return Login{}, false
	}
	return Login{Provider: provider, Kind: KindOAuth, Cred: cred, Label: label}, true
}

// object is the decoded JSON document every reader walks.
type object = map[string]any

// decode reads a JSON object. Anything else — malformed bytes, an array, a
// scalar, a null — is not a credential file.
func decode(raw []byte) (object, bool) {
	var doc object
	if err := json.Unmarshal(raw, &doc); err != nil || doc == nil {
		return nil, false
	}
	return doc, true
}

// child is decode's step for a nested object.
func child(doc object, key string) (object, bool) {
	nested, ok := doc[key].(object)
	return nested, ok && nested != nil
}

// text reads a string field, trimmed. A number or an object where a string
// belongs reads as empty, so the credential is refused instead of coerced.
func text(doc object, key string) string {
	s, _ := doc[key].(string)
	return strings.TrimSpace(s)
}

// expiresMillis normalises an expiry that may be seconds or milliseconds. A
// millisecond timestamp for any date after 1973 is larger than 1e11, and no
// second-resolution timestamp reaches it before the year 5138, so the split is
// unambiguous. Absent, zero or nonsense expires as 0 — "unknown", never a
// guessed date.
func expiresMillis(v any) int64 {
	f, ok := v.(float64)
	if !ok || f <= 0 {
		return 0
	}
	ms := int64(f)
	if ms < 1e11 {
		ms *= 1000
	}
	return ms
}

// isoMillis reads an RFC 3339 timestamp (Grok, Antigravity) as unix
// milliseconds; an unparseable or absent one is 0.
func isoMillis(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

// candidateIDs is the declared provider id first, then the CLI's own names for
// it, in the order the file's entry is picked.
func candidateIDs(provider string, aliases map[string][]string) []string {
	return append([]string{provider}, aliases[provider]...)
}

// parseProviderMap reads the one shape pi and OpenCode share:
//
//	{"<provider>": {"type":"api_key"|"api"|"oauth","key"?, "access"?, "refresh"?, "expires"?, "accountId"?}}
//
// pi writes "api_key" and OpenCode "api"; both mean the same key.
func parseProviderMap(raw []byte, provider string, aliases map[string][]string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	var entry object
	for _, id := range candidateIDs(provider, aliases) {
		if entry, ok = child(doc, id); ok {
			break
		}
	}
	if !ok {
		return Login{}, false
	}
	switch strings.ToLower(text(entry, "type")) {
	case "api_key", "api":
		return loginAPIKey(provider, text(entry, "key"), "")
	case "oauth":
		return loginOAuth(provider, text(entry, "access"), text(entry, "refresh"),
			expiresMillis(entry["expires"]), text(entry, "accountId"), "")
	}
	return Login{}, false
}

// parseClaude reads Claude Code's `$CLAUDE_CONFIG_DIR/.credentials.json`:
//
//	{"claudeAiOauth":{"accessToken","refreshToken","expiresAt"(ms),"subscriptionType"?}}
//
// The file also carries vendor plugin OAuth (mcpOAuth) that is not Claude's own
// login, so only claudeAiOauth is read. The subscription type is a plan name,
// not an identity, so it is not a Label.
func parseClaude(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	oauth, ok := child(doc, "claudeAiOauth")
	if !ok {
		return Login{}, false
	}
	return loginOAuth(provider, text(oauth, "accessToken"), text(oauth, "refreshToken"),
		expiresMillis(oauth["expiresAt"]), "", "")
}

// parseCodex reads Codex's `$CODEX_HOME/auth.json`:
//
//	{"OPENAI_API_KEY"?, "tokens":{"access_token","refresh_token","account_id"}?, "auth_mode"?}
//
// A key wins when both are present — that is what Codex itself prefers — and
// the ChatGPT tokens are the oauth row with the account id that names the
// workspace. Codex carries no expiry field (it lives inside the JWTs), so the
// oauth row has none either.
func parseCodex(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	if key := text(doc, "OPENAI_API_KEY"); key != "" {
		return loginAPIKey(provider, key, "")
	}
	tokens, ok := child(doc, "tokens")
	if !ok {
		return Login{}, false
	}
	return loginOAuth(provider, text(tokens, "access_token"), text(tokens, "refresh_token"),
		0, text(tokens, "account_id"), "")
}

// parseGrok reads Grok's `$GROK_HOME/auth.json`, a map keyed
// "<oidc_issuer>::<oidc_client_id>", each entry holding `key` (the access
// token), `refresh_token`, `expires_at` (RFC 3339) and the account's `email`.
// The email is a vendor-volunteered label, which is why it is the one Label a
// reader sets.
func parseGrok(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	for _, id := range sortedKeys(doc) {
		// The issuer::client_id key is the documented shape; an entry under
		// any other name is not Grok's session.
		if !strings.Contains(id, "::") {
			continue
		}
		entry, ok := child(doc, id)
		if !ok {
			continue
		}
		if login, ok := loginOAuth(provider, text(entry, "key"), text(entry, "refresh_token"),
			isoMillis(text(entry, "expires_at")), "", text(entry, "email")); ok {
			return login, true
		}
	}
	return Login{}, false
}

// parseHermes reads Hermes' `$HERMES_HOME/auth.json`:
//
//	{"providers":{"<id>":{"tokens":{"access_token","refresh_token","account_id"},"auth_mode"}},
//	 "credential_pool":{"<id>":[{"auth_type","key"?}]}}
//
// The providers document is the login Hermes itself used (oauth); the pool is
// its native multi-credential list, whose api_key entries are read only when
// the providers document has nothing for that provider — the pool's
// secret_fingerprint is a hash, not a key, so an entry without one is no
// credential rather than a guessed one.
func parseHermes(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	if providers, ok := child(doc, "providers"); ok {
		for _, id := range candidateIDs(provider, hermesIDs) {
			entry, ok := child(providers, id)
			if !ok {
				continue
			}
			tokens, ok := child(entry, "tokens")
			if !ok {
				continue
			}
			if login, ok := loginOAuth(provider, text(tokens, "access_token"),
				text(tokens, "refresh_token"), 0, text(tokens, "account_id"), ""); ok {
				return login, true
			}
		}
	}
	if pool, ok := child(doc, "credential_pool"); ok {
		for _, id := range candidateIDs(provider, hermesIDs) {
			list, ok := pool[id].([]any)
			if !ok {
				continue
			}
			for _, item := range list {
				entry, ok := item.(object)
				if !ok {
					continue
				}
				switch strings.ToLower(text(entry, "auth_type")) {
				case "api_key", "key":
					if login, ok := loginAPIKey(provider, text(entry, "key"), ""); ok {
						return login, true
					}
				}
			}
		}
	}
	return Login{}, false
}

// parseMuse reads Muse Code's `~/.config/muse/auth.json`:
//
//	{"schema_version":1,"providers":{"meta":{"api_key"}}}
//
// Muse signs in with Meta only, and `muse auth set --provider` accepts `meta`
// alone; the entry is the one credential the file can hold.
func parseMuse(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	providers, ok := child(doc, "providers")
	if !ok {
		return Login{}, false
	}
	meta, ok := child(providers, "meta")
	if !ok {
		return Login{}, false
	}
	return loginAPIKey(provider, text(meta, "api_key"), "")
}

// parseAgy reads Antigravity's
// `~/.gemini/antigravity-cli/antigravity-oauth-token`:
//
//	{"token":{"access_token","refresh_token","expiry"(RFC 3339)},"auth_method"?,"id_token"?}
//
// The Google account's email lives in the id_token, which is not parsed here:
// reading claims out of an unsigned string would be a claim of its own.
func parseAgy(raw []byte, provider string) (Login, bool) {
	doc, ok := decode(raw)
	if !ok {
		return Login{}, false
	}
	token, ok := child(doc, "token")
	if !ok {
		return Login{}, false
	}
	return loginOAuth(provider, text(token, "access_token"), text(token, "refresh_token"),
		isoMillis(text(token, "expiry")), "", "")
}

// sortedKeys is a document's keys in a fixed order, so a file holding several
// credentials (Grok's issuer entries) always answers with the same one.
func sortedKeys(doc object) []string {
	keys := make([]string, 0, len(doc))
	for key := range doc {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
