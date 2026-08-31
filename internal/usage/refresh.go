package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

// Client ids match internal/oauth (pi's public clients). Drift here breaks refresh.
const (
	anthropicClientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	codexClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	kimiClientID      = "17e5f671-d194-4dfb-9706-5516cb48c098"
	xaiClientID       = "b1a00492-073a-47ea-816f-4c329264a828"
)

func (c *Client) refresh(ctx context.Context, provider string, cred catalog.OAuthCred, accountID string) (catalog.OAuthCred, error) {
	if cred.Refresh == "" {
		return cred, fmt.Errorf("no refresh token")
	}
	var (
		body   []byte
		status int
		err    error
	)
	switch provider {
	case "anthropic":
		payload, _ := json.Marshal(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": cred.Refresh,
			"client_id":     anthropicClientID,
		})
		body, status, err = c.postJSON(ctx, c.url("anthropic.token", ""), string(payload), nil)
	case "openai-codex":
		body, status, err = c.postForm(ctx, c.url("codex.token", ""), map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": cred.Refresh,
			"client_id":     codexClientID,
		}, nil)
	case "kimi-coding":
		body, status, err = c.postForm(ctx, c.url("kimi.token", ""), map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": cred.Refresh,
			"client_id":     kimiClientID,
		}, nil)
	case "xai":
		body, status, err = c.postForm(ctx, c.url("xai.token", ""), map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": cred.Refresh,
			"client_id":     xaiClientID,
		}, nil)
	case "github-copilot":
		return c.refreshCopilot(ctx, cred, accountID)
	default:
		return cred, fmt.Errorf("no refresh")
	}
	if err != nil {
		return cred, err
	}
	if status >= 300 {
		return cred, fmt.Errorf("refresh http %d", status)
	}
	access, refresh, exp, err := parseTokenJSON(body, c.now())
	if err != nil {
		return cred, err
	}
	next := cred
	next.Access = access
	if refresh != "" {
		next.Refresh = refresh
	}
	next.Expires = exp
	_ = catalog.UpdateOAuthTokensAccount(provider, accountID, next.Access, next.Refresh, next.Expires)
	return next, nil
}

func (c *Client) refreshCopilot(ctx context.Context, cred catalog.OAuthCred, accountID string) (catalog.OAuthCred, error) {
	// Copilot stores the GitHub token as refresh; access is the short-lived Copilot token.
	body, status, err := c.get(ctx, c.url("copilot.token", ""), cred.Refresh, map[string]string{
		"User-Agent":             "GitHubCopilotChat/0.35.0",
		"Editor-Version":         "vscode/1.107.0",
		"Editor-Plugin-Version":  "copilot-chat/0.35.0",
		"Copilot-Integration-Id": "vscode-chat",
	})
	if err != nil {
		return cred, err
	}
	if status >= 300 {
		return cred, fmt.Errorf("refresh http %d", status)
	}
	var tok struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if json.Unmarshal(body, &tok) != nil || tok.Token == "" {
		return cred, fmt.Errorf("invalid copilot token")
	}
	next := cred
	next.Access = tok.Token
	if tok.ExpiresAt > 0 {
		next.Expires = tok.ExpiresAt*1000 - 5*60*1000
	}
	_ = catalog.UpdateOAuthTokensAccount("github-copilot", accountID, next.Access, next.Refresh, next.Expires)
	return next, nil
}

func parseTokenJSON(raw []byte, now time.Time) (access, refresh string, expires int64, err error) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return "", "", 0, fmt.Errorf("invalid token json")
	}
	access, _ = m["access_token"].(string)
	refresh, _ = m["refresh_token"].(string)
	if access == "" {
		return "", "", 0, fmt.Errorf("missing access_token")
	}
	var expSec float64
	switch v := m["expires_in"].(type) {
	case float64:
		expSec = v
	}
	if expSec <= 0 {
		expSec = 3600
	}
	expires = now.Add(time.Duration(expSec)*time.Second - 5*time.Minute).UnixMilli()
	return access, refresh, expires, nil
}
