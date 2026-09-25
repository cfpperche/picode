package usage

import "context"

// Identity answers the account identity a provider's own profile endpoint
// volunteers for a stored OAuth credential — the same call the quota adapter
// already makes for pi's roster, used here so a second subscription is a
// second row instead of a replacement. Empty when the provider publishes no
// such endpoint or the call fails: the caller treats "unknown" as "one row per
// provider", never as a reason to refuse an import.
func (c *Client) Identity(ctx context.Context, provider, access string) string {
	if access == "" || c == nil {
		return ""
	}
	switch provider {
	case "anthropic":
		hdr := map[string]string{
			"anthropic-beta": "oauth-2025-04-20",
			"User-Agent":     "claude-cli/2.0",
		}
		body, status, err := c.get(ctx, c.url("anthropic.profile", ""), access, hdr)
		if err != nil || status != 200 {
			return ""
		}
		return emailFromProfile(body)
	}
	return ""
}
