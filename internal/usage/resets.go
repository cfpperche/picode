package usage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
)

func (c *Client) fetchCodexResets(ctx context.Context, cred catalog.OAuthCred, rep *Report) {
	hdr := map[string]string{
		"OpenAI-Beta": "codex-1",
		"originator":  "Codex Desktop",
	}
	if cred.AccountID != "" {
		hdr["ChatGPT-Account-ID"] = cred.AccountID
	}
	body, status, err := c.get(ctx, c.url("codex.resets", ""), cred.Access, hdr)
	if err != nil || status >= 300 {
		return
	}
	rep.Resets = parseCodexResets(body)
}

func parseCodexResets(raw []byte) []ResetCredit {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	arr, _ := m["credits"].([]any)
	if arr == nil {
		arr, _ = m["rateLimitResetCredits"].([]any)
	}
	out := []ResetCredit{}
	for _, item := range arr {
		im := mapOf(item)
		if im == nil {
			continue
		}
		st := strings.ToLower(str(im["status"]))
		if st != "" && st != "available" {
			continue
		}
		id := str(im["id"])
		if id == "" {
			id = str(im["credit_id"])
		}
		if id == "" {
			continue
		}
		exp := str(im["expires_at"])
		if exp == "" {
			exp = str(im["expiresAt"])
		}
		out = append(out, ResetCredit{ID: id, ExpiresAt: normalizeTime(exp)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExpiresAt < out[j].ExpiresAt })
	return out
}

func (c *Client) redeemCodex(ctx context.Context, cred catalog.OAuthCred, creditID string) error {
	hdr := map[string]string{
		"OpenAI-Beta": "codex-1",
		"originator":  "Codex Desktop",
	}
	if cred.AccountID != "" {
		hdr["ChatGPT-Account-ID"] = cred.AccountID
	}
	payload, _ := json.Marshal(map[string]string{
		"credit_id":         creditID,
		"redeem_request_id": newID(),
	})
	_, status, err := c.postJSON(ctx, c.url("codex.redeem", ""), string(payload), mergeAuth(hdr, cred.Access))
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("redeem http %d", status)
	}
	return nil
}

func mergeAuth(hdr map[string]string, bearer string) map[string]string {
	out := map[string]string{}
	for k, v := range hdr {
		out[k] = v
	}
	if bearer != "" {
		out["Authorization"] = "Bearer " + bearer
	}
	return out
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", timeUnixNano())
	}
	return hex.EncodeToString(b)
}

func timeUnixNano() int64 {
	return Default.now().UnixNano()
}

func (c *Client) fetchXAIResets(ctx context.Context, cred catalog.OAuthCred, rep *Report) {
	hdr := map[string]string{
		"content-type":             "application/grpc-web+proto",
		"connect-protocol-version": "1",
		"x-grpc-web":               "1",
		"Authorization":            "Bearer " + cred.Access,
		"x-xai-token-auth":         "xai-grok-cli",
	}
	body, status, err := c.postBytes(ctx, c.url("xai.resets", ""), []byte{0, 0, 0, 0, 0}, hdr)
	if err == nil && status < 300 {
		if toks := parseRemainingResets(body); len(toks) > 0 {
			rep.Resets = toks
			return
		}
	}
	jhdr := map[string]string{
		"content-type":     "application/json",
		"Accept":           "application/json",
		"Authorization":    "Bearer " + cred.Access,
		"x-xai-token-auth": "xai-grok-cli",
	}
	jbody, jstatus, jerr := c.postJSON(ctx, c.url("xai.resets", ""), "{}", jhdr)
	if jerr != nil || jstatus >= 300 {
		return
	}
	rep.Resets = parseRemainingResetsJSON(jbody)
}

func (c *Client) redeemXAI(ctx context.Context, cred catalog.OAuthCred, tokenID string) error {
	hdr := map[string]string{
		"content-type":             "application/grpc-web+proto",
		"connect-protocol-version": "1",
		"x-grpc-web":               "1",
		"Authorization":            "Bearer " + cred.Access,
		"x-xai-token-auth":         "xai-grok-cli",
	}
	_, status, err := c.postBytes(ctx, c.url("xai.redeem", ""), encodeRedeemResetRequest(tokenID), hdr)
	if err != nil {
		return err
	}
	if status >= 300 {
		jhdr := map[string]string{
			"content-type":     "application/json",
			"Authorization":    "Bearer " + cred.Access,
			"x-xai-token-auth": "xai-grok-cli",
		}
		payload, _ := json.Marshal(map[string]string{"tokenId": tokenID})
		_, jstatus, jerr := c.postJSON(ctx, c.url("xai.redeem", ""), string(payload), jhdr)
		if jerr != nil {
			return jerr
		}
		if jstatus >= 300 {
			return fmt.Errorf("redeem http %d", jstatus)
		}
	}
	return nil
}
