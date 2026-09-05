package store

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestWebhookCRUD(t *testing.T) {
	s := openTest(t)
	if err := s.AppendEvent("agent.example", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	latest, _ := s.LatestEventID()
	w, err := s.AddWebhook("https://example.com/hook?token=private", []string{" Agent.", "inbox.", "agent.", ""})
	if err != nil {
		t.Fatal(err)
	}
	if w.Secret == "" || w.Cursor != latest || !w.Enabled || strings.Join(w.Types, ",") != "agent.,inbox." {
		t.Fatalf("bad defaults: %#v", w)
	}
	rows, err := s.ListWebhooks()
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %v %v", rows, err)
	}
	raw, _ := json.Marshal(rows)
	if strings.Contains(string(raw), w.Secret) || strings.Contains(string(raw), `"secret"`) {
		t.Fatal("secret serialized")
	}
	events, _ := s.ListEventsSince(latest, 100)
	for _, ev := range events {
		if strings.Contains(string(ev.Data), "private") || strings.Contains(string(ev.Data), w.Secret) {
			t.Fatal("credential in event")
		}
	}
	after := w
	after.Cursor = 42
	after.LastStatus = "delivered"
	after.LastAttemptAt = nowUTC()
	if err := s.SaveWebhookProgress(w, after); err != nil {
		t.Fatal(err)
	}
	w.URL = "https://example.com/v2"
	w.Enabled = false
	up, err := s.UpdateWebhook(w)
	if err != nil || up.Secret != w.Secret || up.Cursor != 42 || up.Enabled || up.Revision != w.Revision+1 {
		t.Fatalf("update: %#v %v", up, err)
	}
	if _, err := s.UpdateWebhook(w); !errors.Is(err, ErrWebhookConflict) {
		t.Fatalf("stale edit: %v", err)
	}
	rotated, err := s.RotateWebhookSecret(up.ID, up.Revision)
	if err != nil || rotated.Secret == up.Secret {
		t.Fatalf("rotate: %v", err)
	}
	if err := s.DeleteWebhook(w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetWebhook(w.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	if err := s.DeleteWebhook(w.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestWebhookValidation(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		types     []string
		ok        bool
	}{
		{"https", "https://example.com/hook", []string{"agent."}, true},
		{"LAN", "http://n8n.lan/hook", []string{"inbox."}, true},
		{"loopback", "http://127.0.0.1:8080/hook", []string{"task."}, true},
		{"empty", "", []string{"agent."}, false},
		{"scheme", "ftp://example.com", []string{"agent."}, false},
		{"missing host", "https://", []string{"agent."}, false},
		{"userinfo", "https://user:secret@example.com", []string{"agent."}, false},
		{"fragment", "https://example.com/#secret", []string{"agent."}, false},
		{"metadata", "http://169.254.169.254", []string{"agent."}, false},
		{"metadata ipv6", "http://[fe80::1]/", []string{"agent."}, false},
		{"no types", "https://example.com", nil, false},
		{"internal", "https://example.com", []string{"webhook."}, false},
		{"wildcard", "https://example.com", []string{"*"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := openTest(t)
			_, err := s.AddWebhook(tc.url, tc.types)
			if (err == nil) != tc.ok {
				t.Fatalf("add: %v", err)
			}
			w, err := s.AddWebhook("https://example.com", []string{"agent."})
			if err != nil {
				t.Fatal(err)
			}
			w.URL = tc.url
			w.Types = tc.types
			_, err = s.UpdateWebhook(w)
			if (err == nil) != tc.ok {
				t.Fatalf("edit: %v", err)
			}
		})
	}
}

func TestWebhookStaleDeliveryDecisionTable(t *testing.T) {
	for _, action := range []string{"edit", "pause", "rotate", "delete", "advance"} {
		t.Run(action, func(t *testing.T) {
			s := openTest(t)
			w, err := s.AddWebhook("https://example.com", []string{"agent."})
			if err != nil {
				t.Fatal(err)
			}
			switch action {
			case "edit":
				next := w
				next.URL = "https://other.example.com"
				_, err = s.UpdateWebhook(next)
			case "pause":
				next := w
				next.Enabled = false
				_, err = s.UpdateWebhook(next)
			case "rotate":
				_, err = s.RotateWebhookSecret(w.ID, w.Revision)
			case "delete":
				err = s.DeleteWebhook(w.ID)
			case "advance":
				next := w
				next.Cursor++
				err = s.SaveWebhookProgress(w, next)
			}
			if err != nil {
				t.Fatal(err)
			}
			after := w
			after.Cursor = 100
			after.LastStatus = "delivered"
			if err := s.SaveWebhookProgress(w, after); !errors.Is(err, ErrWebhookConflict) {
				t.Fatalf("stale ack accepted: %v", err)
			}
		})
	}
}

func TestWebhookTransactionRollback(t *testing.T) {
	for _, op := range []string{"add", "update", "delete", "rotate", "delivery"} {
		t.Run(op, func(t *testing.T) {
			s := openTest(t)
			w, err := s.AddWebhook("https://example.com", []string{"agent."})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.db.Exec(`CREATE TRIGGER fail_event BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT,'injected failure'); END`); err != nil {
				t.Fatal(err)
			}
			switch op {
			case "add":
				_, err = s.AddWebhook("https://other.example.com", []string{"agent."})
			case "update":
				next := w
				next.Enabled = false
				_, err = s.UpdateWebhook(next)
			case "delete":
				err = s.DeleteWebhook(w.ID)
			case "rotate":
				_, err = s.RotateWebhookSecret(w.ID, w.Revision)
			case "delivery":
				next := w
				next.Cursor = 99
				next.LastStatus = "delivered"
				err = s.SaveWebhookProgress(w, next)
			}
			if err == nil {
				t.Fatal("event failure ignored")
			}
			got, err := s.GetWebhook(w.ID)
			if err != nil || got.Enabled != w.Enabled || got.Cursor != w.Cursor || got.Secret != w.Secret || got.Revision != w.Revision {
				t.Fatalf("mutation escaped rollback: %#v %v", got, err)
			}
			rows, _ := s.ListWebhooks()
			if len(rows) != 1 {
				t.Fatal("add escaped rollback")
			}
		})
	}
}
