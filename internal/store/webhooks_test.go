package store

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWebhookCRUD(t *testing.T) {
	s := openTest(t)

	w, err := s.AddWebhook("https://example.com/hook", []string{" Agent.", " inbox.", "agent.", ""})
	if err != nil {
		t.Fatal(err)
	}
	if w.Secret == "" {
		t.Fatal("AddWebhook must return the generated secret once")
	}
	if len(w.Types) != 2 || w.Types[0] != "agent." || w.Types[1] != "inbox." {
		t.Fatalf("types not normalized: %v", w.Types)
	}
	if w.Enabled != true || w.Cursor < 0 {
		t.Fatalf("unexpected defaults: %+v", w)
	}

	// The stored row keeps the secret but it never marshals — the HTTP
	// layer answers with the marshaled struct, so the secret cannot cross
	// to the browser after creation.
	listed, err := s.ListWebhooks()
	if err != nil || len(listed) != 1 {
		t.Fatalf("list: %v %+v", err, listed)
	}
	if j, _ := json.Marshal(listed[0]); strings.Contains(string(j), "secret") {
		t.Fatalf("secret must not marshal: %s", j)
	}
	got, err := s.GetWebhook(w.ID)
	if err != nil || got.Secret != w.Secret {
		t.Fatalf("store lost the secret: %v", err)
	}

	// Delivery bookkeeping advances cursor and status without events.
	deliveries := 0
	s.OnEvent = func(ev Event) { deliveries++ }
	if err := s.SetWebhookDelivery(w.ID, 42, "ok", ""); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetWebhook(w.ID)
	if err != nil || got.Cursor != 42 || got.LastStatus != "ok" {
		t.Fatalf("delivery state not recorded: %+v %v", got, err)
	}
	if deliveries != 0 {
		t.Fatalf("delivery bookkeeping must not append events, got %d", deliveries)
	}

	// Update changes url/types/enabled, keeps secret and cursor.
	got.URL = "https://example.com/v2"
	got.Types = []string{"inbox."}
	got.Enabled = false
	up, err := s.UpdateWebhook(got)
	if err != nil {
		t.Fatal(err)
	}
	if up.URL != "https://example.com/v2" || up.Enabled || up.Secret != w.Secret || up.Cursor != 42 {
		t.Fatalf("update lost state: %+v", up)
	}

	if err := s.DeleteWebhook(w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetWebhook(w.ID); err == nil {
		t.Fatal("deleted webhook still exists")
	}
	if err := s.DeleteWebhook(w.ID); err == nil {
		t.Fatal("deleting a missing webhook must be ErrNotFound")
	}
}

func TestAddWebhookValidation(t *testing.T) {
	s := openTest(t)
	if _, err := s.AddWebhook("ftp://example.com", nil); err == nil {
		t.Fatal("non-http url must be refused")
	}
	if _, err := s.AddWebhook("   ", nil); err == nil {
		t.Fatal("empty url must be refused")
	}
	// http on a tailnet/LAN is the owner's call — allowed alongside https.
	if _, err := s.AddWebhook("http://n8n.lan/hook", nil); err != nil {
		t.Fatalf("http url must be allowed: %v", err)
	}
}

func TestWebhookCursorStartsAtLatest(t *testing.T) {
	s := openTest(t)
	if _, err := s.AddWebhook("https://example.com/first", nil); err != nil {
		t.Fatal(err)
	} // seeds a webhook.created event
	latest, _ := s.LatestEventID()
	w, err := s.AddWebhook("https://example.com/second", nil)
	if err != nil {
		t.Fatal(err)
	}
	if w.Cursor != latest {
		t.Fatalf("cursor = %d, want the newest event id %d (history is not replayed to new subscriptions)", w.Cursor, latest)
	}
}
