package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func reminderEvents(evs []Event) []ReminderFire {
	var out []ReminderFire
	for _, ev := range evs {
		if ev.Type != "pin.reminded" {
			continue
		}
		var f ReminderFire
		_ = json.Unmarshal(ev.Data, &f)
		out = append(out, f)
	}
	return out
}

func TestPinReminderSetAndRead(t *testing.T) {
	s := openTest(t)
	p, _ := s.CreatePin("Deploy checklist", nil, "# Steps\n\n```\ncode\n```\nRun the smoke test first.")
	if _, err := s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 1, TZ: "UTC"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("1 min interval: %v", err)
	}
	if _, err := s.SetPinReminder(p.ID, PinReminderParams{Kind: "once", At: "yesterday", TZ: "UTC"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad at: %v", err)
	}
	if _, err := s.SetPinReminder("nope-000000", PinReminderParams{Kind: "cron", Cron: "0 9 * * *", TZ: "UTC"}); err != ErrNotFound {
		t.Fatalf("missing pin: %v", err)
	}
	r, err := s.SetPinReminder(p.ID, PinReminderParams{Kind: "cron", Cron: "0 9 * * *", TZ: "America/Sao_Paulo"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Label != "every day at 09:00" || r.NextAt == nil || r.Anchor != "schedule" || !r.Enabled {
		t.Fatalf("reminder = %+v", r)
	}
	got, _ := s.GetPin(p.ID)
	if got.Reminder == nil || got.Reminder.ID != r.ID {
		t.Fatalf("pin.reminder = %+v", got.Reminder)
	}
	list, _ := s.ListPins()
	if list[0].Reminder == nil || list[0].Reminder.Label != "every day at 09:00" {
		t.Fatalf("list reminder = %+v", list[0].Reminder)
	}
	// Replacing keeps the id; the rule starts over.
	r2, err := s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 180, Anchor: "completion", TZ: "UTC"})
	if err != nil || r2.ID != r.ID || r2.Cron != "" || r2.Label != "every 3 h after you close it" {
		t.Fatalf("replace = %+v %v", r2, err)
	}
	if err := s.DeletePinReminder(p.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeletePinReminder(p.ID); err != ErrNotFound {
		t.Fatalf("second delete = %v", err)
	}
	if got, _ := s.GetPin(p.ID); got.Reminder != nil {
		t.Fatal("reminder survived delete")
	}
}

func TestPinReminderFireOnce(t *testing.T) {
	s := openTest(t)
	var evs []Event
	s.OnEvent = func(ev Event) { evs = append(evs, ev) }
	p, _ := s.CreatePin("Call the bank", nil, "# Note\n\nAsk about the fee.")
	at := time.Now().UTC().Add(2 * time.Minute).Format(time.RFC3339)
	r, err := s.SetPinReminder(p.ID, PinReminderParams{Kind: "once", At: at, TZ: "Europe/Lisbon"})
	if err != nil {
		t.Fatal(err)
	}
	if ids, _ := s.DueReminderIDs(time.Now()); len(ids) != 0 {
		t.Fatalf("due before its time: %v", ids)
	}
	now := time.Now().UTC().Add(3 * time.Minute)
	ids, _ := s.DueReminderIDs(now)
	if len(ids) != 1 || ids[0] != r.ID {
		t.Fatalf("due = %v", ids)
	}
	if err := s.FireReminder(r.ID, now); err != nil {
		t.Fatal(err)
	}
	items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder})
	if len(items) != 1 || items[0].SourceKind != InboxFromPin || items[0].SourceID != p.ID || items[0].Title != "Call the bank" || items[0].Body != "Ask about the fee." || items[0].Blocking || items[0].State != InboxUnread {
		t.Fatalf("items = %+v", items)
	}
	fires := reminderEvents(evs)
	if len(fires) != 1 || fires[0].InboxID != items[0].ID || fires[0].PinID != p.ID || fires[0].CatchUp || fires[0].Label == "" {
		t.Fatalf("fires = %+v", fires)
	}
	after, _ := s.GetPinReminder(p.ID)
	if after.NextAt != nil || after.Enabled || after.LastFiredAt == nil {
		t.Fatalf("once after fire = %+v", after)
	}
	if ids, _ := s.DueReminderIDs(now.Add(time.Hour)); len(ids) != 0 {
		t.Fatal("a finished once is still due")
	}
}

func TestPinReminderIntervalDedupesAndCatchesUp(t *testing.T) {
	s := openTest(t)
	var evs []Event
	s.OnEvent = func(ev Event) { evs = append(evs, ev) }
	p, _ := s.CreatePin("Water the plants", nil, "")
	r, _ := s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 60, TZ: "UTC"})
	// The daemon was down: the slot is two hours old when the tick sees it.
	slot, _ := time.Parse(time.RFC3339Nano, *r.NextAt)
	now := slot.Add(2 * time.Hour)
	if err := s.FireReminder(r.ID, now); err != nil {
		t.Fatal(err)
	}
	items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder})
	if len(items) != 1 || !strings.Contains(items[0].Body, "Was due") {
		t.Fatalf("catch-up body = %+v", items)
	}
	first := items[0].ID
	fires := reminderEvents(evs)
	if len(fires) != 1 || !fires[0].CatchUp {
		t.Fatalf("fires = %+v", fires)
	}
	after, _ := s.GetPinReminder(p.ID)
	next, _ := time.Parse(time.RFC3339Nano, *after.NextAt)
	if !next.Equal(now.Add(time.Hour)) {
		t.Fatalf("backlog did not collapse: next = %s, now = %s", next, now)
	}
	// Next fire while the item is still open: no second item, one more announcement.
	if err := s.FireReminder(r.ID, next); err != nil {
		t.Fatal(err)
	}
	items, _ = s.ListInboxItems(InboxFilter{Kind: InboxReminder, IncludeSnoozed: true})
	if len(items) != 1 || items[0].ID != first {
		t.Fatalf("second fire piled up: %+v", items)
	}
	if fires = reminderEvents(evs); len(fires) != 2 || fires[1].InboxID != first || fires[1].CatchUp {
		t.Fatalf("re-raise = %+v", fires)
	}
	// Snoozed: the fire advances the clock and says nothing.
	until := next.Add(3 * time.Hour).Format(time.RFC3339)
	if _, err := s.SetInboxItemState(first, "", &until); err != nil {
		t.Fatal(err)
	}
	after, _ = s.GetPinReminder(p.ID)
	next2, _ := time.Parse(time.RFC3339Nano, *after.NextAt)
	if err := s.FireReminder(r.ID, next2); err != nil {
		t.Fatal(err)
	}
	if fires = reminderEvents(evs); len(fires) != 2 {
		t.Fatalf("a snoozed item was re-raised: %+v", fires)
	}
	after, _ = s.GetPinReminder(p.ID)
	if next3, _ := time.Parse(time.RFC3339Nano, *after.NextAt); !next3.After(next2) {
		t.Fatal("snoozed fire did not advance")
	}
	// The snooze ends: the tick wakes the item once and announces it.
	n, err := s.WakeSnoozedReminders(next.Add(4 * time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("wake = %d %v", n, err)
	}
	if n, _ = s.WakeSnoozedReminders(next.Add(5 * time.Hour)); n != 0 {
		t.Fatal("woke twice")
	}
	if fires = reminderEvents(evs); len(fires) != 3 || fires[2].InboxID != first {
		t.Fatalf("wake announce = %+v", fires)
	}
	// Done closes it; the next fire files a fresh item.
	if _, err := s.SetInboxItemState(first, InboxDone, nil); err != nil {
		t.Fatal(err)
	}
	after, _ = s.GetPinReminder(p.ID)
	next4, _ := time.Parse(time.RFC3339Nano, *after.NextAt)
	if err := s.FireReminder(r.ID, next4); err != nil {
		t.Fatal(err)
	}
	items, _ = s.ListInboxItems(InboxFilter{Kind: InboxReminder})
	if len(items) != 1 || items[0].ID == first {
		t.Fatalf("after done = %+v", items)
	}
}

func TestPinReminderCompletionAnchor(t *testing.T) {
	s := openTest(t)
	p, _ := s.CreatePin("Stretch", nil, "")
	r, _ := s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 120, Anchor: "completion", TZ: "UTC"})
	slot, _ := time.Parse(time.RFC3339Nano, *r.NextAt)
	if err := s.FireReminder(r.ID, slot); err != nil {
		t.Fatal(err)
	}
	after, _ := s.GetPinReminder(p.ID)
	if after.NextAt != nil {
		t.Fatalf("completion-anchored reminder owes a fire before it is closed: %s", *after.NextAt)
	}
	items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder})
	before := time.Now().UTC()
	if _, err := s.SetInboxItemState(items[0].ID, InboxDone, nil); err != nil {
		t.Fatal(err)
	}
	after, _ = s.GetPinReminder(p.ID)
	if after.NextAt == nil {
		t.Fatal("close did not schedule the next fire")
	}
	next, _ := time.Parse(time.RFC3339Nano, *after.NextAt)
	if d := next.Sub(before); d < 119*time.Minute || d > 121*time.Minute {
		t.Fatalf("next after close = %s from close", d)
	}
}

func TestPinReminderGoneWithPinOrRule(t *testing.T) {
	s := openTest(t)
	p, _ := s.CreatePin("Gone", nil, "")
	r, _ := s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 5, TZ: "UTC"})
	slot, _ := time.Parse(time.RFC3339Nano, *r.NextAt)
	_ = s.FireReminder(r.ID, slot)
	if items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder}); len(items) != 1 {
		t.Fatal("no item")
	}
	if err := s.DeletePinReminder(p.ID); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder}); len(items) != 0 {
		t.Fatalf("rule gone but item open: %+v", items)
	}
	r, _ = s.SetPinReminder(p.ID, PinReminderParams{Kind: "interval", IntervalMin: 5, TZ: "UTC"})
	slot, _ = time.Parse(time.RFC3339Nano, *r.NextAt)
	_ = s.FireReminder(r.ID, slot)
	if err := s.DeletePin(p.ID); err != nil {
		t.Fatal(err)
	}
	if items, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder}); len(items) != 0 {
		t.Fatalf("pin gone but item open: %+v", items)
	}
	if ids, _ := s.DueReminderIDs(time.Now().Add(time.Hour)); len(ids) != 0 {
		t.Fatal("reminder survived the pin")
	}
	done, _ := s.ListInboxItems(InboxFilter{Kind: InboxReminder, State: InboxDone, IncludeSnoozed: true})
	if len(done) != 2 {
		t.Fatalf("closed items = %d", len(done))
	}
}
