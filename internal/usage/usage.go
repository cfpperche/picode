// Package usage fetches live provider plan windows (5h, 7d, weekly, extra)
// using the active auth.json credential. Tokens never leave this package
// in JSON returned to the browser (ADR-0031).
package usage

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/catalog"
)

const (
	StatusOK           = "ok"
	StatusUnsupported  = "unsupported"
	StatusAuthRequired = "auth_required"
	StatusError        = "error"
)

// Window is one quota period. usedPercent is 0–100 when the vendor reports
// utilization. remaining+unit is for extra/prepaid money, not a fake bar.
type Window struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	UsedPercent *float64 `json:"usedPercent,omitempty"`
	Remaining   *float64 `json:"remaining,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	ResetsAt    string   `json:"resetsAt,omitempty"`
}

// ResetCredit is a one-time banked usage-limit reset. id is needed to redeem;
// it is a vendor credit id, not an auth token.
type ResetCredit struct {
	ID        string `json:"id"`
	ExpiresAt string `json:"expiresAt,omitempty"`
}

// Report is the GET /api/providers/{id}/usage body. No secrets.
type Report struct {
	Provider     string        `json:"provider"`
	AccountLabel string        `json:"accountLabel,omitempty"`
	AuthType     string        `json:"authType,omitempty"`
	Plan         string        `json:"plan,omitempty"`
	FetchedAt    string        `json:"fetchedAt"`
	Status       string        `json:"status"`
	Error        string        `json:"error,omitempty"`
	Windows      []Window      `json:"windows"`
	Resets       []ResetCredit `json:"resets,omitempty"`
}

// Client talks to vendor usage endpoints. Tests override Endpoints and HTTP.
type Client struct {
	HTTP      *http.Client
	Now       func() time.Time
	Endpoints map[string]string
}

// Default is the production client (real vendor URLs).
var Default = NewClient(nil)

// NewClient builds a client. nil http uses a 12s timeout.
func NewClient(h *http.Client) *Client {
	if h == nil {
		h = &http.Client{Timeout: 12 * time.Second}
	}
	return &Client{HTTP: h, Endpoints: defaultEndpoints()}
}

func (c *Client) now() time.Time {
	if c != nil && c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) url(key, fallback string) string {
	if c != nil && c.Endpoints != nil {
		if u := c.Endpoints[key]; u != "" {
			return u
		}
	}
	return fallback
}

func defaultEndpoints() map[string]string {
	return map[string]string{
		"anthropic.usage":   "https://api.anthropic.com/api/oauth/usage",
		"anthropic.profile": "https://api.anthropic.com/api/oauth/profile",
		"anthropic.token":   "https://platform.claude.com/v1/oauth/token",
		"codex.usage":       "https://chatgpt.com/backend-api/wham/usage",
		"codex.token":       "https://auth.openai.com/oauth/token",
		"copilot.user":      "https://api.github.com/copilot_internal/user",
		"copilot.token":     "https://api.github.com/copilot_internal/v2/token",
		"kimi.usage":        "https://api.kimi.com/coding/v1/usages",
		"kimi.token":        "https://auth.kimi.com/api/oauth/token",
		"xai.billing":       "https://cli-chat-proxy.grok.com/v1/billing?format=credits",
		"xai.settings":      "https://cli-chat-proxy.grok.com/v1/settings",
		"xai.token":         "https://auth.x.ai/oauth2/token",
		"xai.resets":        "https://grok.com/prod_mc_billing.ConsumerUiSvc/GetRemainingResets",
		"xai.redeem":        "https://grok.com/prod_mc_billing.ConsumerUiSvc/RedeemReset",
		"codex.resets":      "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits",
		"codex.redeem":      "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits/consume",
		"zai.quota":         "https://api.z.ai/api/monitor/usage/quota/limit",
		"zai-cn.quota":      "https://open.bigmodel.cn/api/monitor/usage/quota/limit",
		"opencode-go.usage": "https://opencode.ai/zen/go/v1/usage",
	}
}

// Fetch is Default.Fetch.
func Fetch(ctx context.Context, provider string) Report {
	return Default.Fetch(ctx, provider)
}

// Fetch reads the active cred and the vendor usage payload.
func (c *Client) Fetch(ctx context.Context, provider string) Report {
	if c == nil {
		c = Default
	}
	id := strings.TrimSpace(provider)
	rep := Report{
		Provider:  id,
		FetchedAt: c.now().UTC().Format(time.RFC3339),
		Windows:   []Window{},
		Status:    StatusUnsupported,
	}
	if id == "" {
		return rep
	}
	authType := catalog.ActiveAuthType(id)
	rep.AuthType = authType
	rep.AccountLabel = catalog.ActiveLabel(id)
	kind := catalog.QuotaKind(id, authType)
	if kind == "" {
		return rep
	}
	if kind == catalog.LoginAPIKey {
		key, ok := catalog.ActiveAPIKey(id)
		if !ok {
			rep.Status = StatusAuthRequired
			rep.Error = "Sign in again."
			return rep
		}
		out, _ := c.fetchOnceKey(ctx, id, key)
		return out
	}
	cred, ok := catalog.ActiveOAuth(id)
	if !ok {
		rep.Status = StatusAuthRequired
		rep.Error = "Sign in again."
		return rep
	}
	if cred.Expires > 0 && c.now().UnixMilli() >= cred.Expires {
		next, err := c.refresh(ctx, id, cred)
		if err != nil {
			rep.Status = StatusAuthRequired
			rep.Error = "Sign in again."
			return rep
		}
		cred = next
	}
	out, retry := c.fetchOnce(ctx, id, cred)
	if retry {
		next, err := c.refresh(ctx, id, cred)
		if err != nil {
			out.Status = StatusAuthRequired
			out.Error = "Sign in again."
			out.Windows = []Window{}
			out.Resets = nil
			return out
		}
		out, _ = c.fetchOnce(ctx, id, next)
	}
	return out
}

func (c *Client) fetchOnce(ctx context.Context, provider string, cred catalog.OAuthCred) (Report, bool) {
	rep := Report{
		Provider:     provider,
		AccountLabel: catalog.ActiveLabel(provider),
		AuthType:     catalog.LoginOAuth,
		FetchedAt:    c.now().UTC().Format(time.RFC3339),
		Windows:      []Window{},
		Status:       StatusOK,
	}
	var (
		status int
		err    error
	)
	switch provider {
	case "anthropic":
		status, err = c.anthropic(ctx, cred, &rep)
	case "openai-codex":
		status, err = c.codex(ctx, cred, &rep)
	case "github-copilot":
		status, err = c.copilot(ctx, cred, &rep)
	case "kimi-coding":
		status, err = c.kimi(ctx, cred, &rep)
	case "xai":
		status, err = c.xai(ctx, cred, &rep)
	default:
		rep.Status = StatusUnsupported
		return rep, false
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return rep, true
	}
	if status == http.StatusTooManyRequests {
		rep.Status = StatusError
		rep.Error = "Rate limited."
		rep.Windows = []Window{}
		return rep, false
	}
	if err != nil || (status > 0 && status >= 300) {
		rep.Status = StatusError
		rep.Error = "Couldn't load usage."
		rep.Windows = []Window{}
		rep.Resets = nil
		return rep, false
	}
	return rep, false
}

func (c *Client) fetchOnceKey(ctx context.Context, provider, key string) (Report, bool) {
	rep := Report{
		Provider:     provider,
		AccountLabel: catalog.ActiveLabel(provider),
		AuthType:     catalog.LoginAPIKey,
		FetchedAt:    c.now().UTC().Format(time.RFC3339),
		Windows:      []Window{},
		Status:       StatusOK,
	}
	var (
		status int
		err    error
	)
	switch provider {
	case "zai":
		status, err = c.zai(ctx, key, c.url("zai.quota", ""), &rep)
	case "zai-coding-cn":
		status, err = c.zai(ctx, key, c.url("zai-cn.quota", ""), &rep)
	case "opencode-go":
		status, err = c.opencodeGo(ctx, key, &rep)
	default:
		rep.Status = StatusUnsupported
		return rep, false
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		rep.Status = StatusAuthRequired
		rep.Error = "Sign in again."
		rep.Windows = []Window{}
		return rep, false
	}
	if status == http.StatusTooManyRequests {
		rep.Status = StatusError
		rep.Error = "Rate limited."
		rep.Windows = []Window{}
		return rep, false
	}
	if err != nil || (status > 0 && status >= 300) {
		rep.Status = StatusError
		rep.Error = "Couldn't load usage."
		rep.Windows = []Window{}
		return rep, false
	}
	return rep, false
}

// Redeem spends one banked usage-limit reset for the active account, then
// refetches windows. Empty id picks the soonest available credit.
func Redeem(ctx context.Context, provider, id string) Report {
	return Default.Redeem(ctx, provider, id)
}

func (c *Client) Redeem(ctx context.Context, provider, id string) Report {
	if c == nil {
		c = Default
	}
	provider = strings.TrimSpace(provider)
	rep := c.Fetch(ctx, provider)
	if rep.Status != StatusOK {
		return rep
	}
	if len(rep.Resets) == 0 {
		rep.Status = StatusError
		rep.Error = "No reset available."
		return rep
	}
	if strings.TrimSpace(id) == "" {
		id = rep.Resets[0].ID
	}
	cred, ok := catalog.ActiveOAuth(provider)
	if !ok {
		rep.Status = StatusAuthRequired
		rep.Error = "Sign in again."
		return rep
	}
	var err error
	switch provider {
	case "openai-codex":
		err = c.redeemCodex(ctx, cred, id)
	case "xai":
		err = c.redeemXAI(ctx, cred, id)
	default:
		rep.Status = StatusError
		rep.Error = "This provider has no reset."
		return rep
	}
	if err != nil {
		rep.Status = StatusError
		rep.Error = "Couldn't redeem."
		return rep
	}
	return c.Fetch(ctx, provider)
}
