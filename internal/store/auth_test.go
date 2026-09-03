package store

import (
	"testing"
	"time"
)

func TestSessionsLifecycle(t *testing.T) {
	s := openTest(t)
	sess, secret, err := s.CreateSession(SessionBrowser, "dev-1", "Chrome on Windows", "10.0.0.5", 90*24*time.Hour)
	if err != nil || len(secret) != 64 || sess.ExpiresAt == nil {
		t.Fatalf("create: %+v %q %v", sess, secret, err)
	}
	got, err := s.LookupSession(secret)
	if err != nil || got.ID != sess.ID {
		t.Fatalf("lookup: %+v %v", got, err)
	}
	if _, err := s.LookupSession("nope"); err != ErrSessionInvalid {
		t.Fatalf("bad secret = %v", err)
	}
	if _, _, err := s.CreateSession("weird", "", "", "", 0); err == nil {
		t.Fatal("bad kind accepted")
	}
	list, _ := s.ListSessions()
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
	if err := s.RevokeSession(sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LookupSession(secret); err != ErrSessionInvalid {
		t.Fatal("revoked session still valid")
	}
	if err := s.RevokeSession(sess.ID); err != ErrNotFound {
		t.Fatalf("second revoke = %v", err)
	}
	if list, _ := s.ListSessions(); len(list) != 0 {
		t.Fatal("revoked sessions listed")
	}
	// Expired.
	_, expired, _ := s.CreateSession(SessionBrowser, "", "old", "", time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	if _, err := s.LookupSession(expired); err != ErrSessionInvalid {
		t.Fatal("expired session still valid")
	}
	if n, _ := s.PruneSessions(time.Now().Add(time.Hour)); n != 2 {
		t.Fatalf("pruned %d", n)
	}
	// Token sessions never expire.
	tok, secret, _ := s.CreateSession(SessionToken, "", "ci", "", 0)
	if tok.ExpiresAt != nil {
		t.Fatal("token session got an expiry")
	}
	if _, err := s.LookupSession(secret); err != nil {
		t.Fatal(err)
	}
}

// The loopback-reuse path (ADR-0049 amendment): the newest live browser
// session with a label+ip is reused, its secret rotated in place; expired
// and revoked rows are never reused.
func TestNewestLiveBrowserSessionAndRotation(t *testing.T) {
	s := openTest(t)
	if _, err := s.NewestLiveBrowserSession("This machine · Linux", "127.0.0.1"); err != ErrNotFound {
		t.Fatalf("empty store = %v, want ErrNotFound", err)
	}
	first, secret, _ := s.CreateSession(SessionBrowser, "", "This machine · Linux", "127.0.0.1", 90*24*time.Hour)
	other, _, _ := s.CreateSession(SessionBrowser, "", "This machine · Windows", "127.0.0.1", 90*24*time.Hour)
	got, err := s.NewestLiveBrowserSession("This machine · Linux", "127.0.0.1")
	if err != nil || got.ID != first.ID {
		t.Fatalf("reuse = %+v %v, want %s", got, err, first.ID)
	}
	if _, err := s.NewestLiveBrowserSession("This machine · Linux", "10.1.1.1"); err != ErrNotFound {
		t.Fatal("a different ip must not reuse")
	}
	_ = other

	// Rotate: same row, new secret works, old one dies, expiry renewed.
	rot, rotated, err := s.RotateSessionSecret(first.ID, 90*24*time.Hour)
	if err != nil || rot.ID != first.ID || rotated == secret {
		t.Fatalf("rotate: %+v %q %v", rot, rotated, err)
	}
	if _, err := s.LookupSession(secret); err != ErrSessionInvalid {
		t.Fatal("old secret still valid after rotation")
	}
	if live, err := s.LookupSession(rotated); err != nil || live.ID != first.ID {
		t.Fatalf("rotated secret: %+v %v", live, err)
	}
	if rot.CreatedAt != first.CreatedAt {
		t.Fatal("rotation must keep created_at")
	}
	old, _ := time.Parse(time.RFC3339Nano, *first.ExpiresAt)
	new, _ := time.Parse(time.RFC3339Nano, *rot.ExpiresAt)
	if !new.After(old) {
		t.Fatalf("expiry not renewed: %s -> %s", *first.ExpiresAt, *rot.ExpiresAt)
	}

	// A revoked row is not reusable, and rotating it fails.
	_ = s.RevokeSession(first.ID)
	if _, err := s.NewestLiveBrowserSession("This machine · Linux", "127.0.0.1"); err != ErrNotFound {
		t.Fatal("revoked session reused")
	}
	if _, _, err := s.RotateSessionSecret(first.ID, time.Hour); err != ErrNotFound {
		t.Fatalf("rotate revoked = %v", err)
	}
	if _, _, err := s.RotateSessionSecret("sess-missing", time.Hour); err != ErrNotFound {
		t.Fatalf("rotate missing = %v", err)
	}

	// An expired row is not reusable.
	_, deadSecret, _ := s.CreateSession(SessionBrowser, "", "This machine · Linux", "127.0.0.1", time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	if _, err := s.LookupSession(deadSecret); err != ErrSessionInvalid {
		t.Fatal("setup: expired should be invalid")
	}
	if _, err := s.NewestLiveBrowserSession("This machine · Linux", "127.0.0.1"); err != ErrNotFound {
		t.Fatal("expired session reused")
	}
}

func TestPairingsAreOneShot(t *testing.T) {
	s := openTest(t)
	code, _, err := s.CreatePairing("sess-1", 10*time.Minute)
	if err != nil || len(code) != 64 {
		t.Fatalf("create: %q %v", code, err)
	}
	if err := s.ConsumePairing(code); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumePairing(code); err != ErrPairingUsed {
		t.Fatalf("second use = %v", err)
	}
	if err := s.ConsumePairing("nope"); err != ErrPairingInvalid {
		t.Fatalf("unknown = %v", err)
	}
	old, _, _ := s.CreatePairing("", time.Nanosecond)
	time.Sleep(2 * time.Millisecond)
	if err := s.ConsumePairing(old); err != ErrPairingExpired {
		t.Fatalf("expired = %v", err)
	}
}
