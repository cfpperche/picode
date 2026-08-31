package usage

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cfpperche/picode/internal/catalog"
)

func (c *Client) anthropic(ctx context.Context, cred catalog.OAuthCred, rep *Report) (int, error) {
	hdr := map[string]string{
		"anthropic-beta": "oauth-2025-04-20",
		"User-Agent":     "claude-cli/2.0",
	}
	body, status, err := c.get(ctx, c.url("anthropic.usage", ""), cred.Access, hdr)
	if err != nil || status >= 300 {
		return status, err
	}
	parseAnthropicUsage(body, rep)
	if pbody, pst, perr := c.get(ctx, c.url("anthropic.profile", ""), cred.Access, hdr); perr == nil && pst == 200 {
		if plan := planFromProfile(pbody); plan != "" {
			rep.Plan = plan
		}
	}
	return status, nil
}

func parseAnthropicUsage(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	if w, ok := windowFrom(m["five_hour"], "5h", "5 hours"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if w, ok := windowFrom(m["seven_day"], "7d", "7 days"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if arr, ok := m["limits"].([]any); ok {
		for _, item := range arr {
			im := mapOf(item)
			if im == nil {
				continue
			}
			name := strings.ToLower(str(im["type"]) + " " + str(im["name"]) + " " + str(im["id"]))
			id, label := "win", "Window"
			switch {
			case strings.Contains(name, "5h") || strings.Contains(name, "five") || strings.Contains(name, "session"):
				id, label = "5h", "5 hours"
			case strings.Contains(name, "7d") || strings.Contains(name, "seven") || strings.Contains(name, "week"):
				id, label = "7d", "7 days"
			case strings.Contains(name, "extra") || strings.Contains(name, "overage"):
				if n, ok := num(im["remaining"]); ok {
					rep.Windows = append(rep.Windows, moneyWindow("extra", "Extra usage", n, "usd"))
					continue
				}
			default:
				if s := str(im["name"]); s != "" {
					label = s
					id = strings.ToLower(strings.ReplaceAll(s, " ", "-"))
				}
			}
			if w, ok := windowFrom(item, id, label); ok {
				if !hasWindow(rep.Windows, id) {
					rep.Windows = append(rep.Windows, w)
				}
			}
		}
	}
	if extra := mapOf(m["extra_usage"]); extra != nil {
		if n, ok := num(extra["remaining"]); ok {
			rep.Windows = append(rep.Windows, moneyWindow("extra", "Extra usage", n, "usd"))
		}
	} else if n, ok := num(m["extra_usage"]); ok {
		rep.Windows = append(rep.Windows, moneyWindow("extra", "Extra usage", n, "usd"))
	}
}

func planFromProfile(raw []byte) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	for _, k := range []string{"plan", "subscription_type", "tier", "organization_type"} {
		if s := str(m[k]); s != "" {
			return s
		}
	}
	if org := mapOf(m["organization"]); org != nil {
		if s := str(org["plan"]); s != "" {
			return s
		}
		if s := str(org["billing_type"]); s != "" {
			return s
		}
	}
	return ""
}

func (c *Client) codex(ctx context.Context, cred catalog.OAuthCred, rep *Report) (int, error) {
	hdr := map[string]string{
		"OpenAI-Beta": "codex-1",
		"originator":  "Codex Desktop",
	}
	if cred.AccountID != "" {
		hdr["ChatGPT-Account-ID"] = cred.AccountID
	}
	body, status, err := c.get(ctx, c.url("codex.usage", ""), cred.Access, hdr)
	if err != nil || status >= 300 {
		return status, err
	}
	parseCodexUsage(body, rep)
	return status, nil
}

func parseCodexUsage(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	if s := str(m["plan_type"]); s != "" {
		rep.Plan = s
	} else if s := str(m["plan"]); s != "" {
		rep.Plan = s
	}
	if w, ok := windowFrom(m["five_hour"], "5h", "5 hours"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if w, ok := windowFrom(m["seven_day"], "7d", "7 days"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if rl := mapOf(m["rate_limit"]); rl != nil {
		if w, ok := windowFrom(rl["primary_window"], "5h", "5 hours"); ok && !hasWindow(rep.Windows, "5h") {
			rep.Windows = append(rep.Windows, w)
		}
		if w, ok := windowFrom(rl["secondary_window"], "7d", "7 days"); ok && !hasWindow(rep.Windows, "7d") {
			rep.Windows = append(rep.Windows, w)
		}
	}
	if credits := mapOf(m["credits"]); credits != nil {
		if n, ok := num(credits["balance"]); ok {
			rep.Windows = append(rep.Windows, moneyWindow("credits", "Credits", n, "usd"))
		}
	}
}

func (c *Client) copilot(ctx context.Context, cred catalog.OAuthCred, rep *Report) (int, error) {
	bearer := cred.Refresh
	if bearer == "" {
		bearer = cred.Access
	}
	hdr := map[string]string{
		"User-Agent":             "GitHubCopilotChat/0.35.0",
		"Editor-Version":         "vscode/1.107.0",
		"Editor-Plugin-Version":  "copilot-chat/0.35.0",
		"Copilot-Integration-Id": "vscode-chat",
	}
	body, status, err := c.get(ctx, c.url("copilot.user", ""), bearer, hdr)
	if err != nil || status >= 300 {
		return status, err
	}
	parseCopilotUsage(body, rep)
	return status, nil
}

func parseCopilotUsage(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	if s := str(m["copilot_plan"]); s != "" {
		rep.Plan = s
	} else if s := str(m["access_type_sku"]); s != "" {
		rep.Plan = s
	}
	snaps := mapOf(m["quota_snapshots"])
	if snaps == nil {
		return
	}
	if w, ok := copilotSnap(snaps["premium_interactions"], "premium", "Premium"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if w, ok := copilotSnap(snaps["chat"], "chat", "Chat"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if w, ok := copilotSnap(snaps["completions"], "completions", "Completions"); ok {
		rep.Windows = append(rep.Windows, w)
	}
}

func copilotSnap(v any, id, label string) (Window, bool) {
	m := mapOf(v)
	if m == nil {
		return Window{}, false
	}
	if b, _ := m["unlimited"].(bool); b {
		w := Window{ID: id, Label: label, UsedPercent: ptr(0)}
		w.ResetsAt = resetString(m)
		return w, true
	}
	if w, ok := windowFrom(m, id, label); ok {
		return w, true
	}
	return Window{}, false
}

func (c *Client) kimi(ctx context.Context, cred catalog.OAuthCred, rep *Report) (int, error) {
	body, status, err := c.get(ctx, c.url("kimi.usage", ""), cred.Access, nil)
	if err != nil || status >= 300 {
		return status, err
	}
	parseKimiUsage(body, rep)
	return status, nil
}

func parseKimiUsage(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	if mem := mapOf(m["membership"]); mem != nil {
		if s := str(mem["level"]); s != "" {
			rep.Plan = s
		} else if s := str(mem["name"]); s != "" {
			rep.Plan = s
		}
	}
	if s := str(m["membership_level"]); s != "" && rep.Plan == "" {
		rep.Plan = s
	}
	if w, ok := windowFrom(m["five_hour"], "5h", "5 hours"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if w, ok := windowFrom(m["weekly"], "7d", "7 days"); ok {
		rep.Windows = append(rep.Windows, w)
	}
	if arr, ok := m["usages"].([]any); ok {
		for _, item := range arr {
			im := mapOf(item)
			if im == nil {
				continue
			}
			name := strings.ToLower(str(im["name"]) + " " + str(im["type"]) + " " + str(im["window"]))
			id, label := "win", "Window"
			switch {
			case strings.Contains(name, "5h") || strings.Contains(name, "five") || strings.Contains(name, "hour"):
				id, label = "5h", "5 hours"
			case strings.Contains(name, "week") || strings.Contains(name, "7d"):
				id, label = "7d", "7 days"
			default:
				if s := str(im["name"]); s != "" {
					label = s
					id = strings.ToLower(strings.ReplaceAll(s, " ", "-"))
				}
			}
			if w, ok := windowFrom(item, id, label); ok && !hasWindow(rep.Windows, id) {
				rep.Windows = append(rep.Windows, w)
			}
		}
	}
}

func (c *Client) xai(ctx context.Context, cred catalog.OAuthCred, rep *Report) (int, error) {
	body, status, err := c.get(ctx, c.url("xai.billing", ""), cred.Access, nil)
	if err != nil || status >= 300 {
		return status, err
	}
	parseXAIBilling(body, rep)
	if sbody, sst, serr := c.get(ctx, c.url("xai.settings", ""), cred.Access, nil); serr == nil && sst == 200 {
		if plan := planFromXAISettings(sbody); plan != "" {
			rep.Plan = plan
		}
	}
	return status, nil
}

func parseXAIBilling(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	cfg := mapOf(m["config"])
	if cfg == nil {
		cfg = m
	}
	if n, ok := num(cfg["creditUsagePercent"]); ok {
		w := Window{ID: "1w", Label: "This week", UsedPercent: ptr(clampPct(n))}
		if period := mapOf(cfg["currentPeriod"]); period != nil {
			if s := str(period["end"]); s != "" {
				w.ResetsAt = normalizeTime(s)
			}
		}
		if w.ResetsAt == "" {
			w.ResetsAt = resetString(cfg)
		}
		rep.Windows = append(rep.Windows, w)
	}
	if products := mapOf(cfg["productUsage"]); products != nil {
		if build := mapOf(products["GrokBuild"]); build != nil {
			if n, ok := num(build["usagePercent"]); ok && !hasWindow(rep.Windows, "build") {
				w := Window{ID: "build", Label: "Build", UsedPercent: ptr(clampPct(n))}
				rep.Windows = append(rep.Windows, w)
			}
		}
	}
	used, uok := num(cfg["onDemandUsed"])
	cap, cok := num(cfg["onDemandCap"])
	if uok && cok && cap > 0 {
		left := cap - used
		if left < 0 {
			left = 0
		}
		rep.Windows = append(rep.Windows, moneyWindow("on-demand", "On-demand", left, "usd"))
	}
	if len(rep.Windows) == 0 {
		used, uok := num(m["used"])
		limit, lok := num(m["monthlyLimit"])
		if uok && lok && limit > 0 {
			w := Window{ID: "1m", Label: "This month", UsedPercent: ptr(clampPct(used / limit * 100))}
			rep.Windows = append(rep.Windows, w)
		}
	}
}

func planFromXAISettings(raw []byte) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if s := str(m["subscription_tier_display"]); s != "" {
		return s
	}
	if s := str(m["subscription_tier"]); s != "" {
		return s
	}
	return ""
}

func hasWindow(windows []Window, id string) bool {
	for _, w := range windows {
		if w.ID == id {
			return true
		}
	}
	return false
}
