package automate

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/cron"
	"github.com/cfpperche/picode/internal/store"
)

type fakeRunner struct{ calls []string }

func (f *fakeRunner) Fire(a store.Automation, trigger, payload string) (store.Run, error) {
	f.calls = append(f.calls, a.Name+":"+trigger)
	return store.Run{}, nil
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func local(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func TestJitterBounded(t *testing.T) {
	if j := Jitter("abc", time.Minute); j != 0 {
		t.Fatalf("1-minute schedule must not jitter: %v", j)
	}
	if j := Jitter("abc", 24*time.Hour); j < 0 || j >= 30*time.Minute {
		t.Fatalf("daily jitter out of bounds: %v", j)
	}
	if j := Jitter("abc", 10*time.Minute); j >= 5*time.Minute {
		t.Fatalf("10-minute jitter must be < 5m: %v", j)
	}
	if Jitter("abc", time.Hour) != Jitter("abc", time.Hour) {
		t.Fatal("jitter must be deterministic")
	}
}

func TestDue(t *testing.T) {
	every, _ := cron.Parse("* * * * *") // jitter 0
	a := store.Automation{ID: "x"}
	slot, trig, ok := Due(every, a, local("2026-09-01 10:00"))
	if !ok || trig != store.TriggerSchedule || !slot.Equal(local("2026-09-01 10:00")) {
		t.Fatalf("never-fired due = %v %s %v", slot, trig, ok)
	}
	fired := slot.UTC().Format(time.RFC3339Nano)
	a.LastFiredAt = &fired
	if _, _, ok := Due(every, a, local("2026-09-01 10:00")); ok {
		t.Fatal("same slot must not fire twice")
	}
	// Daemon was down 10:01-10:04; at 10:05 the current slot is due and
	// wins — no catch-up on top of a live fire.
	if _, trig, ok := Due(every, a, local("2026-09-01 10:05")); !ok || trig != store.TriggerSchedule {
		t.Fatalf("live slot = %s %v", trig, ok)
	}

	daily, _ := cron.Parse("0 9 * * *")
	b := store.Automation{ID: "daily"}
	j := Jitter("daily", 24*time.Hour)
	if _, _, ok := Due(daily, b, local("2026-09-01 12:00")); ok {
		t.Fatal("never-fired daily automation must not catch up")
	}
	last := local("2026-08-30 09:00").UTC().Format(time.RFC3339Nano)
	b.LastFiredAt = &last
	slot, trig, ok = Due(daily, b, local("2026-09-02 12:00"))
	if !ok || trig != store.TriggerCatchUp || !slot.Equal(local("2026-09-02 09:00")) {
		t.Fatalf("catch-up = %v %s %v (jitter %v)", slot, trig, ok, j)
	}
	// Stamp it: nothing left to catch up.
	stamped := slot.UTC().Format(time.RFC3339Nano)
	b.LastFiredAt = &stamped
	if _, _, ok := Due(daily, b, local("2026-09-02 12:00")); ok {
		t.Fatal("catch-up must fire once")
	}
	// The jittered slot itself is a schedule fire.
	slot, trig, ok = Due(daily, b, local("2026-09-03 09:00").Add(j))
	if !ok || trig != store.TriggerSchedule || !slot.Equal(local("2026-09-03 09:00")) {
		t.Fatalf("jittered fire = %v %s %v", slot, trig, ok)
	}
}

func TestTickFiresOncePerSlotAndSkipsDisabled(t *testing.T) {
	st := openStore(t)
	on, _, _ := st.CreateAutomation(store.AutomationParams{Name: "on", Action: store.AutomationStart, Prompt: "p", Cron: "* * * * *"})
	off, _, _ := st.CreateAutomation(store.AutomationParams{Name: "off", Action: store.AutomationStart, Prompt: "p", Cron: "* * * * *"})
	dis := false
	_, _ = st.UpdateAutomation(off.ID, store.AutomationPatch{Enabled: &dis})
	_, _, _ = st.CreateAutomation(store.AutomationParams{Name: "hook", Action: store.AutomationStart, Prompt: "p", Webhook: true})

	now := local("2026-09-01 10:00")
	fr := &fakeRunner{}
	e := &Engine{Store: st, Runner: fr, Now: func() time.Time { return now }}
	e.Tick()
	e.Tick() // same minute again (ticker slip)
	if len(fr.calls) != 1 || fr.calls[0] != "on:schedule" {
		t.Fatalf("calls = %v", fr.calls)
	}
	got, _ := st.GetAutomation(on.ID)
	if got.LastFiredAt == nil {
		t.Fatal("last_fired_at not stamped")
	}
	now = now.Add(time.Minute)
	e.Tick()
	if len(fr.calls) != 2 {
		t.Fatalf("next minute did not fire: %v", fr.calls)
	}
}

func TestReconcileFailsStaleRunsAndNotifies(t *testing.T) {
	st := openStore(t)
	a, _, _ := st.CreateAutomation(store.AutomationParams{Name: "a", Action: store.AutomationStart, Prompt: "p", Cron: "* * * * *"})
	r, _ := st.CreateRun(a.ID, store.TriggerSchedule, store.RunRunning, "")
	(&Engine{Store: st}).Reconcile()
	got, _ := st.GetRun(r.ID)
	if got.Status != store.RunFailed || got.Reason != "daemon restarted" {
		t.Fatalf("run = %+v", got)
	}
	items, _ := st.ListInboxItems(store.InboxFilter{})
	if len(items) != 1 || items[0].SourceKind != store.InboxFromAutomation || items[0].SourceID != a.ID {
		t.Fatalf("inbox = %+v", items)
	}
}

func TestNextFireMatchesDue(t *testing.T) {
	daily, _ := cron.Parse("0 9 * * *")
	j := Jitter("x", 24*time.Hour)
	// Created at 09:03 with jitter j >= 5m: today's slot still fires at 09:00+j.
	if j >= 5*time.Minute {
		next, ok := NextFire(daily, "x", local("2026-09-01 09:03"))
		if !ok || !next.Equal(local("2026-09-01 09:00").Add(j)) {
			t.Fatalf("next = %v (jitter %v)", next, j)
		}
		if _, trig, ok := Due(daily, store.Automation{ID: "x"}, next); !ok || trig != store.TriggerSchedule {
			t.Fatalf("Due disagrees with NextFire at %v", next)
		}
	}
	next, ok := NextFire(daily, "x", local("2026-09-01 12:00"))
	if !ok || !next.Equal(local("2026-09-02 09:00").Add(j)) {
		t.Fatalf("tomorrow = %v", next)
	}
}
