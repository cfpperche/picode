package usage

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
)

func (c *Client) zai(ctx context.Context, key, quotaURL string, rep *Report) (int, error) {
	hdr := map[string]string{"Authorization": "Bearer " + key}
	body, status, err := c.get(ctx, quotaURL, "", hdr)
	if err != nil || status >= 300 {
		// some dashboards send the key without Bearer
		if status == 401 || status == 403 {
			body, status, err = c.get(ctx, quotaURL, "", map[string]string{"Authorization": key})
		}
		if err != nil || status >= 300 {
			return status, err
		}
	}
	parseZAIQuota(body, rep)
	return status, nil
}

func parseZAIQuota(raw []byte, rep *Report) {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return
	}
	data := mapOf(root["data"])
	if data == nil {
		data = root
	}
	if s := str(data["level"]); s != "" {
		rep.Plan = s
	} else if s := str(data["productName"]); s != "" {
		rep.Plan = s
	}
	arr, _ := data["limits"].([]any)
	type tok struct {
		pct   float64
		reset string
		unit  float64
		num   float64
		kind  string
	}
	var tokens []tok
	for _, item := range arr {
		im := mapOf(item)
		if im == nil {
			continue
		}
		typ := str(im["type"])
		pct, _ := num(im["percentage"])
		reset := ""
		if n, ok := num(im["nextResetTime"]); ok && n > 0 {
			if n > 1e12 {
				reset = time.UnixMilli(int64(n)).UTC().Format(time.RFC3339)
			} else {
				reset = time.Unix(int64(n), 0).UTC().Format(time.RFC3339)
			}
		}
		u, _ := num(im["unit"])
		n, _ := num(im["number"])
		if typ == "TIME_LIMIT" {
			w := Window{ID: "mcp", Label: "Tools", UsedPercent: ptr(clampPct(pct)), ResetsAt: reset}
			rep.Windows = append(rep.Windows, w)
			continue
		}
		// GLM Coding Plan reports CREDIT_LIMIT (credits); older payloads used TOKENS_LIMIT.
		if typ == "TOKENS_LIMIT" || typ == "CREDIT_LIMIT" {
			tokens = append(tokens, tok{pct: clampPct(pct), reset: reset, unit: u, num: n, kind: typ})
		}
	}
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].reset < tokens[j].reset
	})
	for i, t := range tokens {
		id, label := "5h", "5 hours"
		if t.unit == 3 && t.num == 5 {
			id, label = "5h", "5 hours"
		} else if t.unit == 6 && (t.num == 7 || t.num == 1) {
			id, label = "7d", "7 days"
		} else if i == 1 && (t.unit != 3 || t.num != 5) {
			id, label = "7d", "7 days"
		}
		if hasWindow(rep.Windows, id) {
			if id == "5h" {
				id, label = "7d", "7 days"
			} else {
				continue
			}
		}
		rep.Windows = append(rep.Windows, Window{ID: id, Label: label, UsedPercent: ptr(t.pct), ResetsAt: t.reset})
	}
}

func (c *Client) opencodeGo(ctx context.Context, key string, rep *Report) (int, error) {
	body, status, err := c.get(ctx, c.url("opencode-go.usage", ""), key, nil)
	if err != nil || status >= 300 {
		return status, err
	}
	parseOpenCodeGo(body, rep)
	return status, nil
}

func parseOpenCodeGo(raw []byte, rep *Report) {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return
	}
	inner := m
	if d := mapOf(m["data"]); d != nil {
		inner = d
	}
	if usage := mapOf(inner["usage"]); usage != nil {
		inner = usage
	}
	add := func(v any, id, label string) {
		block := mapOf(v)
		if block == nil {
			return
		}
		pct, ok := num(block["usagePercent"])
		if !ok {
			pct, ok = num(block["percent"])
		}
		if !ok {
			used, uok := num(block["usageDollars"])
			lim, lok := num(block["limitDollars"])
			if uok && lok && lim > 0 {
				pct = used / lim * 100
				ok = true
			}
		}
		if !ok {
			return
		}
		w := Window{ID: id, Label: label, UsedPercent: ptr(clampPct(pct))}
		if s := str(block["resetsAt"]); s != "" {
			w.ResetsAt = normalizeTime(s)
		} else if n, nOk := num(block["resetInSec"]); nOk && n > 0 {
			w.ResetsAt = time.Now().UTC().Add(time.Duration(n) * time.Second).Format(time.RFC3339)
		}
		rep.Windows = append(rep.Windows, w)
	}
	add(inner["rolling"], "5h", "5 hours")
	if !hasWindow(rep.Windows, "5h") {
		add(inner["rollingUsage"], "5h", "5 hours")
	}
	add(inner["weekly"], "7d", "7 days")
	if !hasWindow(rep.Windows, "7d") {
		add(inner["weeklyUsage"], "7d", "7 days")
	}
	add(inner["monthly"], "1m", "This month")
	if !hasWindow(rep.Windows, "1m") {
		add(inner["monthlyUsage"], "1m", "This month")
	}
}

func (c *Client) kimiKey(ctx context.Context, key string, rep *Report) (int, error) {
	body, status, err := c.get(ctx, c.url("kimi.usage", ""), key, nil)
	if err != nil || status >= 300 {
		return status, err
	}
	parseKimiUsage(body, rep)
	return status, nil
}

func (c *Client) openrouter(ctx context.Context, key string, rep *Report) (int, error) {
	body, status, err := c.get(ctx, c.url("openrouter.key", ""), key, nil)
	if err != nil || status >= 300 {
		return status, err
	}
	parseOpenRouterKey(body, rep)
	if !hasWindow(rep.Windows, "credits") {
		if cbody, cst, cerr := c.get(ctx, c.url("openrouter.credits", ""), key, nil); cerr == nil && cst == 200 {
			parseOpenRouterCredits(cbody, rep)
		}
	}
	return status, nil
}

func parseOpenRouterKey(raw []byte, rep *Report) {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return
	}
	data := mapOf(root["data"])
	if data == nil {
		data = root
	}
	if s := str(data["label"]); s != "" && !strings.HasPrefix(strings.ToLower(s), "sk-") {
		rep.Plan = s
	}
	remaining, rok := num(data["limit_remaining"])
	if rok {
		rep.Windows = append(rep.Windows, moneyWindow("credits", "Credits", remaining, "usd"))
	}
}

func parseOpenRouterCredits(raw []byte, rep *Report) {
	if hasWindow(rep.Windows, "credits") {
		return
	}
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return
	}
	data := mapOf(root["data"])
	if data == nil {
		data = root
	}
	total, tok := num(data["total_credits"])
	used, uok := num(data["total_usage"])
	if tok && uok {
		left := total - used
		if left < 0 {
			left = 0
		}
		rep.Windows = append(rep.Windows, moneyWindow("credits", "Credits", left, "usd"))
	}
}

func (c *Client) minimax(ctx context.Context, key, quotaURL, codingURL string, rep *Report) (int, error) {
	body, status, err := c.get(ctx, quotaURL, key, nil)
	if err == nil && status < 300 {
		parseMiniMaxRemains(body, rep)
		if len(rep.Windows) > 0 {
			return status, nil
		}
	}
	if codingURL != "" && codingURL != quotaURL {
		b2, s2, e2 := c.get(ctx, codingURL, key, nil)
		if e2 == nil && s2 < 300 {
			parseMiniMaxRemains(b2, rep)
			return s2, e2
		}
		if status >= 300 || err != nil {
			return s2, e2
		}
	}
	if err != nil || status >= 300 {
		return status, err
	}
	return status, nil
}

func parseMiniMaxRemains(raw []byte, rep *Report) {
	var root map[string]any
	if json.Unmarshal(raw, &root) != nil {
		return
	}
	data := mapOf(root["data"])
	if data == nil {
		data = root
	}
	arr, _ := data["model_remains"].([]any)
	if arr == nil {
		arr, _ = root["model_remains"].([]any)
	}
	var pick map[string]any
	for _, item := range arr {
		im := mapOf(item)
		if im == nil {
			continue
		}
		name := strings.ToLower(str(im["model_name"]))
		total, _ := num(im["current_interval_total_count"])
		if total <= 0 {
			continue
		}
		if name == "general" || name == "" {
			pick = im
			break
		}
		if pick == nil {
			pick = im
		}
	}
	if pick == nil && len(arr) == 0 {
		pick = data
	}
	if pick == nil {
		return
	}
	addMiniMaxWindow(rep, pick, "5h", "5 hours",
		"current_interval_remaining_percent", "current_interval_total_count", "current_interval_usage_count", "remains_time", "end_time")
	addMiniMaxWindow(rep, pick, "7d", "7 days",
		"current_weekly_remaining_percent", "current_weekly_total_count", "current_weekly_usage_count", "weekly_remains_time", "weekly_end_time")
}

func addMiniMaxWindow(rep *Report, m map[string]any, id, label, remPctKey, totalKey, remainCountKey, remainsTimeKey, endKey string) {
	if hasWindow(rep.Windows, id) {
		return
	}
	total, tok := num(m[totalKey])
	if tok && total <= 0 {
		return
	}
	var pct float64
	var has bool
	if n, ok := num(m[remPctKey]); ok {
		pct = clampPct(100 - n)
		has = true
	} else if tok {
		remain, rok := num(m[remainCountKey])
		if rok && total > 0 {
			used := total - remain
			if used < 0 {
				used = 0
			}
			pct = clampPct(used / total * 100)
			has = true
		}
	}
	if !has {
		return
	}
	w := Window{ID: id, Label: label, UsedPercent: ptr(pct)}
	if n, ok := num(m[remainsTimeKey]); ok && n > 0 {
		d := time.Duration(n) * time.Millisecond
		if n < 1e10 {
			d = time.Duration(n) * time.Second
		}
		w.ResetsAt = time.Now().UTC().Add(d).Format(time.RFC3339)
	}
	if w.ResetsAt == "" {
		if s := str(m[endKey]); s != "" {
			w.ResetsAt = normalizeTime(s)
		}
	}
	rep.Windows = append(rep.Windows, w)
}
