package usage

import "testing"

func TestAsPercent(t *testing.T) {
	t.Parallel()
	if asPercent(37) != 37 {
		t.Fatalf("37 -> %v", asPercent(37))
	}
	if asPercent(0.42) < 41.9 || asPercent(0.42) > 42.1 {
		t.Fatalf("0.42 -> %v", asPercent(0.42))
	}
	if asPercent(0) != 0 {
		t.Fatal("0")
	}
}

func TestParseAnthropicUsage(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"five_hour": {"utilization": 37, "resets_at": "2026-08-31T19:00:00Z"},
		"seven_day": {"utilization": 0.04, "resets_at": "2026-09-05T12:00:00Z"},
		"extra_usage": {"remaining": 12.4}
	}`)
	var rep Report
	parseAnthropicUsage(raw, &rep)
	if len(rep.Windows) != 3 {
		t.Fatalf("windows %d: %+v", len(rep.Windows), rep.Windows)
	}
	if rep.Windows[0].ID != "5h" || *rep.Windows[0].UsedPercent != 37 {
		t.Fatalf("5h %+v", rep.Windows[0])
	}
	if rep.Windows[1].ID != "7d" || *rep.Windows[1].UsedPercent < 3.9 {
		t.Fatalf("7d %+v", rep.Windows[1])
	}
	if rep.Windows[2].ID != "extra" || *rep.Windows[2].Remaining != 12.4 {
		t.Fatalf("extra %+v", rep.Windows[2])
	}
}

func TestParseCodexUsage(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"plan_type": "plus",
		"rate_limit": {
			"primary_window": {"used_percent": 23.4, "reset_after_seconds": 120},
			"secondary_window": {"used_percent": 5}
		},
		"credits": {"balance": 8}
	}`)
	var rep Report
	parseCodexUsage(raw, &rep)
	if rep.Plan != "plus" {
		t.Fatalf("plan %s", rep.Plan)
	}
	if !hasWindow(rep.Windows, "5h") || !hasWindow(rep.Windows, "7d") || !hasWindow(rep.Windows, "credits") {
		t.Fatalf("%+v", rep.Windows)
	}
}

func TestParseCopilotUsage(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"copilot_plan": "individual",
		"quota_snapshots": {
			"premium_interactions": {
				"entitlement": 300,
				"remaining": 120,
				"percent_remaining": 40
			}
		}
	}`)
	var rep Report
	parseCopilotUsage(raw, &rep)
	if rep.Plan != "individual" || len(rep.Windows) != 1 {
		t.Fatalf("%+v", rep)
	}
	if *rep.Windows[0].UsedPercent < 59 || *rep.Windows[0].UsedPercent > 61 {
		t.Fatalf("used %v", *rep.Windows[0].UsedPercent)
	}
}

func TestParseXAIBilling(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"config": {
			"creditUsagePercent": 42,
			"currentPeriod": {"type": "USAGE_PERIOD_TYPE_WEEKLY", "end": "2026-09-05T12:00:00Z"},
			"onDemandUsed": {"val": 1},
			"onDemandCap": {"val": 10}
		}
	}`)
	var rep Report
	parseXAIBilling(raw, &rep)
	if len(rep.Windows) < 1 || *rep.Windows[0].UsedPercent != 42 {
		t.Fatalf("%+v", rep.Windows)
	}
	if !hasWindow(rep.Windows, "on-demand") {
		t.Fatalf("missing on-demand: %+v", rep.Windows)
	}
}

func TestParseZAIQuota(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"success":true,"data":{"level":"pro","limits":[
		{"type":"TOKENS_LIMIT","unit":3,"number":5,"percentage":15,"nextResetTime":1770648402389},
		{"type":"TOKENS_LIMIT","unit":6,"number":7,"percentage":44,"nextResetTime":1771253202389},
		{"type":"TIME_LIMIT","percentage":12}
	]}}`)
	var rep Report
	parseZAIQuota(raw, &rep)
	if rep.Plan != "pro" || !hasWindow(rep.Windows, "5h") || !hasWindow(rep.Windows, "7d") || !hasWindow(rep.Windows, "mcp") {
		t.Fatalf("%+v", rep)
	}
}

func TestParseOpenCodeGo(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"rolling":{"usagePercent":19.5,"resetInSec":7200},"weekly":{"percent":29.7,"resetsAt":"2026-09-05T12:00:00Z"},"monthly":{"usagePercent":25}}`)
	var rep Report
	parseOpenCodeGo(raw, &rep)
	if !hasWindow(rep.Windows, "5h") || !hasWindow(rep.Windows, "7d") || !hasWindow(rep.Windows, "1m") {
		t.Fatalf("%+v", rep.Windows)
	}
}

func TestParseCodexResets(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"available_count":2,"credits":[
		{"id":"RateLimitResetCredit_a","status":"available","expires_at":"2026-09-12T00:00:00Z"},
		{"id":"RateLimitResetCredit_b","status":"used","expires_at":"2026-09-01T00:00:00Z"}
	]}`)
	got := parseCodexResets(raw)
	if len(got) != 1 || got[0].ID != "RateLimitResetCredit_a" {
		t.Fatalf("%+v", got)
	}
}

func TestParseRemainingResetsJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"tokens":[{"tokenId":"tok_1","validityEnd":"2099-09-12T00:00:00Z"}]}`)
	got := parseRemainingResetsJSON(raw)
	if len(got) != 1 || got[0].ID != "tok_1" {
		t.Fatalf("%+v", got)
	}
}

func TestParseKimiUsage(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"membership": {"level": "pro"},
		"usages": [
			{"name": "5h", "used": 10, "limit": 100, "resets_at": "2026-08-31T18:00:00Z"},
			{"name": "weekly", "used": 20, "limit": 200}
		]
	}`)
	var rep Report
	parseKimiUsage(raw, &rep)
	if rep.Plan != "pro" || len(rep.Windows) != 2 {
		t.Fatalf("%+v", rep)
	}
}
