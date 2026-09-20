package catalog

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/credentials"
)

// OAuthCred is the active oauth slot. Access and refresh are secrets —
// callers must not log them or return them to the browser.
type OAuthCred struct {
	Access    string
	Refresh   string
	Expires   int64 // unix milliseconds; 0 if unknown
	AccountID string
}

// QuotaKind is "oauth" or "api_key" when GET /api/providers/{id}/usage can
// fetch plan windows for this signed-in method. Empty means hide Usage.
func QuotaKind(id, authType string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	switch id {
	case "anthropic", "openai-codex", "github-copilot", "xai":
		if authType == LoginOAuth {
			return LoginOAuth
		}
	case "kimi-coding":
		if authType == LoginOAuth || authType == LoginAPIKey {
			return authType
		}
	case "zai", "zai-coding-cn", "opencode-go", "openrouter", "minimax", "minimax-cn":
		if authType == LoginAPIKey {
			return LoginAPIKey
		}
	}
	return ""
}

// ActiveAuthType is api_key, oauth, or empty if the provider is not signed in.
func ActiveAuthType(provider string) string {
	raw := peekCred(provider)
	if len(raw) == 0 {
		if _, _, ok := EnvKeyName(provider); ok {
			return LoginAPIKey
		}
		return ""
	}
	return credType(raw)
}

// ActiveLabel is the vault display name for the active slot. An env-supplied
// provider has no vault row, so it is labelled by the variable that supplies it.
func ActiveLabel(provider string) string {
	for _, a := range accountsOf(provider) {
		if a.Active {
			if a.Label != "" {
				return a.Label
			}
			break
		}
	}
	if len(peekCred(provider)) == 0 {
		if name, _, ok := EnvKeyName(provider); ok {
			return name
		}
	}
	return "Default"
}

// peekCred is the credential pi's auth.json currently holds for a provider
// (what the agent would use right now). Nil when the file or the key is
// missing.
func peekCred(provider string) json.RawMessage {
	raw, err := os.ReadFile(AuthPath())
	if err != nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	return obj[provider]
}

// credType is the credential's shape as pi wrote it: "api_key" unless the
// entry says otherwise.
func credType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return LoginAPIKey
	}
	return credentials.Type(raw)
}

func parseOAuth(raw json.RawMessage) (OAuthCred, bool) {
	if len(raw) == 0 {
		return OAuthCred{}, false
	}
	var m struct {
		Type      string  `json:"type"`
		Access    string  `json:"access"`
		Refresh   string  `json:"refresh"`
		Expires   float64 `json:"expires"`
		AccountID string  `json:"accountId"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return OAuthCred{}, false
	}
	if m.Type != LoginOAuth {
		return OAuthCred{}, false
	}
	if strings.TrimSpace(m.Access) == "" && strings.TrimSpace(m.Refresh) == "" {
		return OAuthCred{}, false
	}
	return OAuthCred{
		Access:    strings.TrimSpace(m.Access),
		Refresh:   strings.TrimSpace(m.Refresh),
		Expires:   int64(m.Expires),
		AccountID: strings.TrimSpace(m.AccountID),
	}, true
}

func parseAPIKey(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var m struct {
		Type string `json:"type"`
		Key  string `json:"key"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return "", false
	}
	if m.Type == LoginOAuth {
		return "", false
	}
	k := strings.TrimSpace(m.Key)
	if k == "" {
		return "", false
	}
	return k, true
}

// ActiveOAuth returns the live auth.json oauth cred. false if missing or not oauth.
func ActiveOAuth(provider string) (OAuthCred, bool) {
	return parseOAuth(peekCred(provider))
}

// ActiveAPIKey is the live auth.json key. false if missing or oauth.
func ActiveAPIKey(provider string) (string, bool) {
	if k, ok := parseAPIKey(peekCred(provider)); ok {
		return k, true
	}
	// No auth.json entry: pi falls back to the env var, so Usage must too,
	// or an env-supplied OpenRouter key would show "sign in again" while
	// every agent on this machine is happily using it.
	return EnvAPIKey(provider)
}

// VaultOAuth is the saved oauth cred for one vault row. Does not swap auth.json.
func VaultOAuth(provider, accountID string) (OAuthCred, bool) {
	row, ok, err := vault().Row(strings.TrimSpace(provider), strings.TrimSpace(accountID))
	if err != nil || !ok {
		return OAuthCred{}, false
	}
	return parseOAuth(row.Cred)
}

// VaultAPIKey is the saved key for one vault row. Does not swap auth.json.
func VaultAPIKey(provider, accountID string) (string, bool) {
	row, ok, err := vault().Row(strings.TrimSpace(provider), strings.TrimSpace(accountID))
	if err != nil || !ok {
		return "", false
	}
	return parseAPIKey(row.Cred)
}

// VaultAuthType is api_key, oauth, or empty if the vault row is missing.
func VaultAuthType(provider, accountID string) string {
	row, ok, err := vault().Row(strings.TrimSpace(provider), strings.TrimSpace(accountID))
	if err != nil || !ok {
		return ""
	}
	if row.Type != "" {
		return row.Type
	}
	return credentials.Type(row.Cred)
}

// VaultLabel is the display name for a vault row, or "Default".
func VaultLabel(provider, accountID string) string {
	row, ok, err := vault().Row(strings.TrimSpace(provider), strings.TrimSpace(accountID))
	if err != nil || !ok || row.Label == "" {
		return "Default"
	}
	return row.Label
}

// UpdateOAuthTokens writes new access/refresh/expiry into the active slot,
// keeping extra fields (accountId). Never logs token values.
func UpdateOAuthTokens(provider, access, refresh string, expires int64) error {
	raw := peekCred(provider)
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil || m == nil {
		m = map[string]any{}
	}
	m["type"] = LoginOAuth
	if access != "" {
		m["access"] = access
	}
	if refresh != "" {
		m["refresh"] = refresh
	}
	if expires != 0 {
		m["expires"] = expires
	}
	return PutOAuth(provider, m)
}

// UpdateOAuthTokensAccount writes new tokens onto one vault row. auth.json
// is updated only when that row is the active slot. Does not swap the active
// account.
func UpdateOAuthTokensAccount(provider, accountID, access, refresh string, expires int64) error {
	provider = strings.TrimSpace(provider)
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return UpdateOAuthTokens(provider, access, refresh, expires)
	}
	if _, ok, err := vault().Row(provider, accountID); err != nil || !ok {
		return UpdateOAuthTokens(provider, access, refresh, expires)
	}
	active, err := vault().UpdateTokens(provider, accountID, access, refresh, expires)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	row, _, err := vault().Row(provider, accountID)
	if err != nil {
		return err
	}
	cred := row.Cred
	return mutateAuth(func(obj map[string]json.RawMessage) error {
		obj[provider] = cred
		return nil
	})
}
