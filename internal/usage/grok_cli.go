package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

func grokAuthPath() string {
	if h := strings.TrimSpace(os.Getenv("GROK_HOME")); h != "" {
		return filepath.Join(h, "auth.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".grok", "auth.json")
}

func grokCLICookie() string {
	return strings.TrimSpace(os.Getenv("GROK_COOKIE"))
}

type grokCLIEntry struct {
	Key          string `json:"key"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	AuthMode     string `json:"auth_mode"`
}

func grokCLICred() (catalog.OAuthCred, bool) {
	path := grokAuthPath()
	if path == "" {
		return catalog.OAuthCred{}, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return catalog.OAuthCred{}, false
	}
	var obj map[string]grokCLIEntry
	if json.Unmarshal(raw, &obj) != nil || len(obj) == 0 {
		return catalog.OAuthCred{}, false
	}
	prefer := ""
	for k := range obj {
		if strings.HasPrefix(k, "https://auth.x.ai::") {
			prefer = k
			break
		}
	}
	if prefer == "" {
		for k := range obj {
			if strings.TrimSpace(obj[k].Key) != "" {
				prefer = k
				break
			}
		}
	}
	if prefer == "" {
		return catalog.OAuthCred{}, false
	}
	e := obj[prefer]
	access := strings.TrimSpace(e.Key)
	refresh := strings.TrimSpace(e.RefreshToken)
	if access == "" && refresh == "" {
		return catalog.OAuthCred{}, false
	}
	var exp int64
	if s := strings.TrimSpace(e.ExpiresAt); s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			exp = t.UnixMilli()
		} else if t, err := time.Parse(time.RFC3339, s); err == nil {
			exp = t.UnixMilli()
		}
	}
	return catalog.OAuthCred{Access: access, Refresh: refresh, Expires: exp}, true
}

func (c *Client) grokCLIAccess(ctx context.Context) string {
	cred, ok := grokCLICred()
	if !ok {
		return ""
	}
	if cred.Access != "" && (cred.Expires == 0 || c.now().UnixMilli() < cred.Expires) {
		return cred.Access
	}
	if cred.Refresh == "" {
		return cred.Access
	}
	next, err := c.refreshGrokCLI(ctx, cred)
	if err != nil {
		return cred.Access
	}
	return next.Access
}

func (c *Client) refreshGrokCLI(ctx context.Context, cred catalog.OAuthCred) (catalog.OAuthCred, error) {
	body, status, err := c.postForm(ctx, c.url("xai.token", ""), map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": cred.Refresh,
		"client_id":     xaiClientID,
	}, nil)
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
	return next, nil
}
