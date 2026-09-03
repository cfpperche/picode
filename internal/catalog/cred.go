package catalog

import (
	"encoding/json"
	"strings"
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
	a, ok := peekVaultAccount(provider, accountID)
	if !ok {
		return OAuthCred{}, false
	}
	return parseOAuth(a.Cred)
}

// VaultAPIKey is the saved key for one vault row. Does not swap auth.json.
func VaultAPIKey(provider, accountID string) (string, bool) {
	a, ok := peekVaultAccount(provider, accountID)
	if !ok {
		return "", false
	}
	return parseAPIKey(a.Cred)
}

// VaultAuthType is api_key, oauth, or empty if the vault row is missing.
func VaultAuthType(provider, accountID string) string {
	a, ok := peekVaultAccount(provider, accountID)
	if !ok {
		return ""
	}
	if a.Type != "" {
		return a.Type
	}
	return credType(a.Cred)
}

// VaultLabel is the display name for a vault row, or "Default".
func VaultLabel(provider, accountID string) string {
	a, ok := peekVaultAccount(provider, accountID)
	if !ok {
		return "Default"
	}
	if a.Label != "" {
		return a.Label
	}
	return "Default"
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
	v := loadVault()
	slot := v[provider]
	idx := -1
	for i, a := range slot.Accounts {
		if a.ID == accountID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return UpdateOAuthTokens(provider, access, refresh, expires)
	}
	raw := slot.Accounts[idx].Cred
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
	next, err := json.Marshal(m)
	if err != nil {
		return err
	}
	slot.Accounts[idx].Cred = next
	slot.Accounts[idx].Type = LoginOAuth
	v[provider] = slot
	if err := saveVault(v); err != nil {
		return err
	}
	if slot.Active == accountID {
		return mutateAuth(func(obj map[string]json.RawMessage) error {
			obj[provider] = next
			return nil
		})
	}
	return nil
}
