package remind

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// The decision table from docs/plans/pins-v2.md, one row per case.
func TestRuleTable(t *testing.T) {
	now := at("2026-09-08T12:00:00Z")
	sp := "America/Sao_Paulo" // UTC-3, no DST since 2019
	cases := []struct {
		name      string
		rule      Rule
		slot      time.Time
		wantNext  string // after fire; "" = nothing owed
		wantClose string // after close at now; "" = unchanged
	}{
		{"once fires and finishes", Rule{Kind: KindOnce, At: at("2026-09-08T11:59:00Z"), TZ: sp}, at("2026-09-08T11:59:00Z"), "", ""},
		{"interval from schedule steps from the slot", Rule{Kind: KindInterval, Interval: 3 * time.Hour, Anchor: AnchorSchedule, TZ: sp}, at("2026-09-08T11:59:00Z"), "2026-09-08T14:59:00Z", ""},
		{"interval backlog collapses to now", Rule{Kind: KindInterval, Interval: time.Hour, Anchor: AnchorSchedule, TZ: sp}, at("2026-09-08T05:00:00Z"), "2026-09-08T13:00:00Z", ""},
		{"interval from completion owes nothing until closed", Rule{Kind: KindInterval, Interval: 24 * time.Hour, Anchor: AnchorCompletion, TZ: sp}, at("2026-09-08T11:59:00Z"), "", "2026-09-09T12:00:00Z"},
		{"cron takes the next match in the zone", Rule{Kind: KindCron, Cron: "0 9 * * *", TZ: sp}, at("2026-09-08T12:00:00Z"), "2026-09-09T12:00:00Z", ""},
		{"cron backlog collapses (missed two days)", Rule{Kind: KindCron, Cron: "0 9 * * *", TZ: sp}, at("2026-09-06T12:00:00Z"), "2026-09-09T12:00:00Z", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			next, ok := c.rule.AfterFire(c.slot, now)
			got := ""
			if ok {
				got = next.Format(time.RFC3339)
			}
			if got != c.wantNext {
				t.Fatalf("AfterFire = %q ok=%v, want %q", got, ok, c.wantNext)
			}
			closed, ok := c.rule.AfterClose(now)
			gotC := ""
			if ok {
				gotC = closed.Format(time.RFC3339)
			}
			if gotC != c.wantClose {
				t.Fatalf("AfterClose = %q, want %q", gotC, c.wantClose)
			}
		})
	}
}

// "Every day at 09:00" is a wall-clock rule: across a DST change the UTC
// instant moves, the local hour does not. "Every 24 hours" drifts instead.
func TestCronKeepsLocalHourAcrossDST(t *testing.T) {
	ny := Rule{Kind: KindCron, Cron: "0 9 * * *", TZ: "America/New_York"}
	before := at("2026-03-07T15:00:00Z") // Sat 10:00 EST; DST starts Sun 08 Mar 2026
	first, _ := ny.AfterFire(before, before)
	if first.Format(time.RFC3339) != "2026-03-08T13:00:00Z" { // 09:00 EDT
		t.Fatalf("next after spring-forward = %s", first)
	}
	loc, _ := time.LoadLocation("America/New_York")
	if first.In(loc).Hour() != 9 {
		t.Fatalf("local hour = %d", first.In(loc).Hour())
	}
	day := Rule{Kind: KindInterval, Interval: 24 * time.Hour, Anchor: AnchorSchedule, TZ: "America/New_York"}
	slot := at("2026-03-07T14:00:00Z") // 09:00 EST
	next, _ := day.AfterFire(slot, slot)
	if next.In(loc).Hour() != 10 {
		t.Fatalf("a 24 h interval should drift to 10:00 local, got %d", next.In(loc).Hour())
	}
}

func TestFirstAndValidate(t *testing.T) {
	now := at("2026-09-08T12:00:00Z")
	bad := []Rule{
		{Kind: KindOnce, At: at("2026-09-08T11:00:00Z"), TZ: "UTC"}, // passed
		{Kind: KindInterval, Interval: time.Minute, Anchor: AnchorSchedule, TZ: "UTC"},
		{Kind: KindInterval, Interval: time.Hour, Anchor: "sometimes", TZ: "UTC"},
		{Kind: KindCron, Cron: "not a cron", TZ: "UTC"},
		{Kind: KindCron, Cron: "0 9 * * *", TZ: "Mars/Olympus"},
		{Kind: KindCron, Cron: "0 9 * * *"},
		{Kind: "weekly", TZ: "UTC"},
	}
	for i, r := range bad {
		if err := r.Validate(now); !errors.Is(err, ErrInvalid) {
			t.Errorf("bad %d accepted: %v", i, err)
		}
	}
	once := Rule{Kind: KindOnce, At: at("2026-09-08T12:30:00Z"), TZ: "UTC"}
	if err := once.Validate(now); err != nil {
		t.Fatal(err)
	}
	if f, ok := once.First(now); !ok || !f.Equal(once.At) {
		t.Fatalf("once first = %v %v", f, ok)
	}
	iv := Rule{Kind: KindInterval, Interval: 90 * time.Minute, Anchor: AnchorCompletion, TZ: "UTC"}
	if f, _ := iv.First(now); f.Format(time.RFC3339) != "2026-09-08T13:30:00Z" {
		t.Fatalf("interval first = %s", f)
	}
	// An interval with a named first fire starts there, then steps.
	named := Rule{Kind: KindInterval, Interval: 3 * 24 * time.Hour, Anchor: AnchorSchedule, TZ: "UTC", At: at("2026-09-10T09:00:00Z")}
	if err := named.Validate(now); err != nil {
		t.Fatal(err)
	}
	if f, _ := named.First(now); f.Format(time.RFC3339) != "2026-09-10T09:00:00Z" {
		t.Fatalf("named first = %s", f)
	}
	if n, _ := named.AfterFire(at("2026-09-10T09:00:00Z"), at("2026-09-10T09:00:30Z")); n.Format(time.RFC3339) != "2026-09-13T09:00:00Z" {
		t.Fatalf("named step = %s", n)
	}
	if err := (Rule{Kind: KindInterval, Interval: time.Hour, Anchor: AnchorSchedule, TZ: "UTC", At: at("2026-09-08T10:00:00Z")}).Validate(now); !errors.Is(err, ErrInvalid) {
		t.Fatalf("past first fire accepted: %v", err)
	}
	cr := Rule{Kind: KindCron, Cron: "30 8 * * 1-5", TZ: "America/Sao_Paulo"}
	if f, _ := cr.First(now); f.Format(time.RFC3339) != "2026-09-09T11:30:00Z" { // Wed 08:30 BRT
		t.Fatalf("cron first = %s", f)
	}
	if !CatchUp(now.Add(-2*time.Minute), now) || CatchUp(now.Add(-30*time.Second), now) {
		t.Fatal("catch-up threshold")
	}
}

func TestLabels(t *testing.T) {
	cases := map[string]Rule{
		"every day at 09:00":           {Kind: KindCron, Cron: "0 9 * * *", TZ: "UTC"},
		"weekdays at 08:30":            {Kind: KindCron, Cron: "30 8 * * 1-5", TZ: "UTC"},
		"every Monday at 09:00":        {Kind: KindCron, Cron: "0 9 * * 1", TZ: "UTC"},
		"cron */15 * * * *":            {Kind: KindCron, Cron: "*/15 * * * *", TZ: "UTC"},
		"every 3 h":                    {Kind: KindInterval, Interval: 3 * time.Hour, Anchor: AnchorSchedule, TZ: "UTC"},
		"every day after you close it": {Kind: KindInterval, Interval: 24 * time.Hour, Anchor: AnchorCompletion, TZ: "UTC"},
		"every 45 min":                 {Kind: KindInterval, Interval: 45 * time.Minute, Anchor: AnchorSchedule, TZ: "UTC"},
		"at Wed 9 Sep 09:00":           {Kind: KindOnce, At: at("2026-09-09T12:00:00Z"), TZ: "America/Sao_Paulo"},
	}
	for want, r := range cases {
		if got := r.Label(); got != want {
			t.Errorf("label = %q, want %q", got, want)
		}
	}
	if !strings.HasPrefix(Rule{Kind: KindCron, Cron: "0 9 * * *", TZ: "UTC"}.Label(), "every") {
		t.Fatal("prefix")
	}
}
