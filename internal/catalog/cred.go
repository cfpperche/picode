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
	case "anthropic", "openai-codex", "github-copilot", "kimi-coding", "xai":
		if authType == LoginOAuth {
			return LoginOAuth
		}
	case "zai", "zai-coding-cn", "opencode-go":
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
		return ""
	}
	return credType(raw)
}

// ActiveLabel is the vault display name for the active slot.
func ActiveLabel(provider string) string {
	for _, a := range accountsOf(provider) {
		if a.Active {
			if a.Label != "" {
				return a.Label
			}
			break
		}
	}
	return "Default"
}

// ActiveOAuth returns the live auth.json oauth cred. false if missing or not oauth.
func ActiveOAuth(provider string) (OAuthCred, bool) {
	raw := peekCred(provider)
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

// ActiveAPIKey is the live auth.json key. false if missing or oauth.
func ActiveAPIKey(provider string) (string, bool) {
	raw := peekCred(provider)
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
