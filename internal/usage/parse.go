package usage

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

func ptr(v float64) *float64 { return &v }

func clampPct(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// asPercent maps vendor utilization (0–1 fraction or 0–100) to 0–100.
func asPercent(v float64) float64 {
	if v <= 1.0001 && v >= 0 {
		return clampPct(v * 100)
	}
	return clampPct(v)
}

func num(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	case map[string]any:
		if inner, ok := t["val"]; ok {
			return num(inner)
		}
		if inner, ok := t["value"]; ok {
			return num(inner)
		}
	}
	return 0, false
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t > 1e12 {
			return time.UnixMilli(int64(t)).UTC().Format(time.RFC3339)
		}
		if t > 1e9 {
			return time.Unix(int64(t), 0).UTC().Format(time.RFC3339)
		}
	}
	return ""
}

func mapOf(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func windowFrom(v any, id, label string) (Window, bool) {
	m := mapOf(v)
	if m == nil {
		return Window{}, false
	}
	w := Window{ID: id, Label: label}
	var pct float64
	var has bool
	for _, k := range []string{"utilization", "used_percent", "usedPercent", "usagePercent", "usage_percent", "percent", "percent_used"} {
		if n, ok := num(m[k]); ok {
			if k == "utilization" {
				pct = asPercent(n)
			} else {
				pct = clampPct(n)
				if n <= 1.0001 {
					pct = asPercent(n)
				}
			}
			has = true
			break
		}
	}
	if !has {
		used, uok := num(m["used"])
		limit, lok := num(m["limit"])
		if !lok {
			limit, lok = num(m["entitlement"])
		}
		if uok && lok && limit > 0 {
			pct = clampPct(used / limit * 100)
			has = true
		}
	}
	if !has {
		if rem, rok := num(m["remaining"]); rok {
			if ent, eok := num(m["entitlement"]); eok && ent > 0 {
				pct = clampPct((ent - rem) / ent * 100)
				has = true
			} else if pr, pok := num(m["percent_remaining"]); pok {
				pct = clampPct(100 - asPercent(pr))
				has = true
			}
		}
	}
	if has {
		w.UsedPercent = ptr(pct)
	}
	w.ResetsAt = resetString(m)
	if !has && w.ResetsAt == "" {
		return Window{}, false
	}
	return w, true
}

func resetString(m map[string]any) string {
	for _, k := range []string{"resets_at", "reset_at", "resetsAt", "resetAt", "end", "resets"} {
		if s := str(m[k]); s != "" {
			return normalizeTime(s)
		}
	}
	if n, ok := num(m["reset_after_seconds"]); ok && n > 0 {
		return time.Now().UTC().Add(time.Duration(n) * time.Second).Format(time.RFC3339)
	}
	if n, ok := num(m["reset_after"]); ok && n > 0 {
		d := time.Duration(n) * time.Second
		if n > 1e12 {
			return time.UnixMilli(int64(n)).UTC().Format(time.RFC3339)
		}
		return time.Now().UTC().Add(d).Format(time.RFC3339)
	}
	return ""
}

func normalizeTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format(time.RFC3339)
		}
	}
	return s
}

func moneyWindow(id, label string, remaining float64, unit string) Window {
	return Window{ID: id, Label: label, Remaining: ptr(remaining), Unit: unit}
}
