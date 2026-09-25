package usage

import "context"

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
