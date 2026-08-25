package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

const (
	copilotClientID = "Iv1.b507a08c87ecfe98"
	kimiClientID    = "17e5f671-d194-4dfb-9706-5516cb48c098"
	xaiClientID     = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiScope        = "openid profile email offline_access grok-cli:access api:access"
)

type deviceCred struct {
	url, code string
	poll      func(ctx context.Context) error
}

func beginDevice(provider string) (deviceCred, error) {
	switch provider {
	case "github-copilot":
		return beginCopilot()
	case "kimi-coding":
		return beginKimi()
	case "xai":
		return beginXAI()
	default:
		return deviceCred{}, fmt.Errorf("no device login for %s", provider)
	}
}

func beginCopilot() (deviceCred, error) {
	form := url.Values{"client_id": {copilotClientID}, "scope": {"read:user"}}
	raw, err := postFormJSON("https://github.com/login/device/code", form, map[string]string{
		"Accept": "application/json", "User-Agent": "GitHubCopilotChat/0.35.0",
	})
	if err != nil {
		return deviceCred{}, err
	}
	var d struct {
		DeviceCode      string  `json:"device_code"`
		UserCode        string  `json:"user_code"`
		VerificationURI string  `json:"verification_uri"`
		Interval        float64 `json:"interval"`
		ExpiresIn       float64 `json:"expires_in"`
	}
	if json.Unmarshal(raw, &d) != nil || d.DeviceCode == "" || d.UserCode == "" {
		return deviceCred{}, fmt.Errorf("invalid github device response")
	}
	interval := time.Duration(d.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	return deviceCred{url: d.VerificationURI, code: d.UserCode, poll: func(ctx context.Context) error {
		ghToken, err := pollForm(ctx, "https://github.com/login/oauth/access_token", url.Values{
			"client_id": {copilotClientID}, "device_code": {d.DeviceCode},
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"},
		}, map[string]string{"Accept": "application/json", "User-Agent": "GitHubCopilotChat/0.35.0"}, interval, "access_token")
		if err != nil {
			return err
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/copilot_internal/v2/token", nil)
		req.Header.Set("Authorization", "Bearer "+ghToken)
		req.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")
		req.Header.Set("Editor-Version", "vscode/1.107.0")
		req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.35.0")
		req.Header.Set("Copilot-Integration-Id", "vscode-chat")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		raw, _ := io.ReadAll(res.Body)
		if res.StatusCode >= 300 {
			return fmt.Errorf("copilot token http %d", res.StatusCode)
		}
		var tok struct {
			Token     string `json:"token"`
			ExpiresAt int64  `json:"expires_at"`
		}
		if json.Unmarshal(raw, &tok) != nil || tok.Token == "" {
			return fmt.Errorf("invalid copilot token")
		}
		return catalog.PutOAuth("github-copilot", map[string]any{
			"type":    "oauth",
			"refresh": ghToken,
			"access":  tok.Token,
			"expires": tok.ExpiresAt*1000 - 5*60*1000,
		})
	}}, nil
}

func beginKimi() (deviceCred, error) {
	form := url.Values{"client_id": {kimiClientID}}
	raw, err := postFormJSON("https://auth.kimi.com/api/oauth/device_authorization", form, map[string]string{"Accept": "application/json"})
	if err != nil {
		return deviceCred{}, err
	}
	var d struct {
		DeviceCode              string  `json:"device_code"`
		UserCode                string  `json:"user_code"`
		VerificationURI         string  `json:"verification_uri"`
		VerificationURIComplete string  `json:"verification_uri_complete"`
		Interval                float64 `json:"interval"`
		ExpiresIn               float64 `json:"expires_in"`
	}
	if json.Unmarshal(raw, &d) != nil || d.DeviceCode == "" {
		return deviceCred{}, fmt.Errorf("invalid kimi device response")
	}
	open := d.VerificationURIComplete
	if open == "" {
		open = d.VerificationURI
	}
	interval := time.Duration(d.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	return deviceCred{url: open, code: d.UserCode, poll: func(ctx context.Context) error {
		return pollOAuthToken(ctx, "https://auth.kimi.com/api/oauth/token", url.Values{
			"client_id": {kimiClientID}, "device_code": {d.DeviceCode},
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"},
		}, interval, "kimi-coding", 0)
	}}, nil
}

func beginXAI() (deviceCred, error) {
	form := url.Values{"client_id": {xaiClientID}, "scope": {xaiScope}, "referrer": {"pi"}}
	raw, err := postFormJSON("https://auth.x.ai/oauth2/device/code", form, map[string]string{"Accept": "application/json"})
	if err != nil {
		return deviceCred{}, err
	}
	var d struct {
		DeviceCode              string  `json:"device_code"`
		UserCode                string  `json:"user_code"`
		VerificationURI         string  `json:"verification_uri"`
		VerificationURIComplete string  `json:"verification_uri_complete"`
		Interval                float64 `json:"interval"`
		ExpiresIn               float64 `json:"expires_in"`
	}
	if json.Unmarshal(raw, &d) != nil || d.DeviceCode == "" {
		return deviceCred{}, fmt.Errorf("invalid xai device response")
	}
	open := d.VerificationURIComplete
	if open == "" {
		open = d.VerificationURI
	}
	interval := time.Duration(d.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}
	return deviceCred{url: open, code: d.UserCode, poll: func(ctx context.Context) error {
		return pollOAuthToken(ctx, "https://auth.x.ai/oauth2/token", url.Values{
			"client_id": {xaiClientID}, "device_code": {d.DeviceCode},
			"grant_type": {"urn:ietf:params:oauth:grant-type:device_code"},
		}, interval, "xai", 5*time.Minute)
	}}, nil
}

func pollOAuthToken(ctx context.Context, tokenURL string, form url.Values, interval time.Duration, provider string, skew time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
		raw, err := postFormJSON(tokenURL, form, map[string]string{"Accept": "application/json"})
		if err != nil {
			continue
		}
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if errStr, _ := body["error"].(string); errStr == "authorization_pending" {
			continue
		} else if errStr == "slow_down" {
			interval += 5 * time.Second
			continue
		} else if errStr != "" {
			return fmt.Errorf("%s", errStr)
		}
		access, _ := body["access_token"].(string)
		refresh, _ := body["refresh_token"].(string)
		var expSec float64
		switch v := body["expires_in"].(type) {
		case float64:
			expSec = v
		}
		if access == "" {
			continue
		}
		exp := time.Now().Add(time.Duration(expSec)*time.Second - skew).UnixMilli()
		return catalog.PutOAuth(provider, map[string]any{
			"type": "oauth", "access": access, "refresh": refresh, "expires": exp,
		})
	}
}

func pollForm(ctx context.Context, u string, form url.Values, headers map[string]string, interval time.Duration, field string) (string, error) {
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
		}
		raw, err := postFormJSON(u, form, headers)
		if err != nil {
			continue
		}
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		if errStr, _ := body["error"].(string); errStr == "authorization_pending" {
			continue
		} else if errStr == "slow_down" {
			interval += 5 * time.Second
			continue
		} else if errStr != "" {
			return "", fmt.Errorf("%s", errStr)
		}
		tok, _ := body[field].(string)
		if tok != "" {
			return tok, nil
		}
	}
}

func postFormJSON(u string, form url.Values, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, u, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}
