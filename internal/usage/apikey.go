package usage

import (
	"context"
	"encoding/json"
	"sort"
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
		if typ == "TOKENS_LIMIT" {
			tokens = append(tokens, tok{pct: clampPct(pct), reset: reset, unit: u, num: n, kind: typ})
		}
	}
	sort.Slice(tokens, func(i, j int) bool {
		return tokens[i].reset < tokens[j].reset
	})
	for i, t := range tokens {
		id, label := "5h", "5 hours"
		if t.unit == 6 && t.num == 7 {
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
