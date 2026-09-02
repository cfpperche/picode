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
