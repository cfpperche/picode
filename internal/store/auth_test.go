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

// Backdates a session's last_seen_at (test fixture only — production code
// lets LookupSession keep it fresh).
func ageSession(s *Store, id string, age time.Duration) {
	_, _ = s.db.Exec(`UPDATE auth_sessions SET last_seen_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-age).Format(time.RFC3339Nano), id)
}

// ADR-0049 amendment 2026-09-06: an auto-minted loopback browser session
// ends when its access ends. Decision table (conditions → picked for
// revocation?):
//
//	loopback mint, idle > grace          → yes
//	loopback mint, last request fresh    → no  (open tab, even backgrounded)
//	paired session (device_id set), idle → no  (a phone returns; deliberate pairing)
//	paired ON loopback (device_id), idle → no
//	token session, idle                  → no
//	already revoked, idle                → no  (nothing left to announce)
//	expired, idle                        → no  (invisible already; prune deletes)
//	unparsable last_seen_at              → no  (never guessed stale)
func TestRevokeStaleLoopbackSessions(t *testing.T) {
	s := openTest(t)
	mk := func(kind, deviceID, label, ip string) string {
		sess, _, err := s.CreateSession(kind, deviceID, label, ip, 90*24*time.Hour)
		if err != nil {
			t.Fatalf("create %s/%s: %v", label, ip, err)
		}
		return sess.ID
	}
	dead := mk(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP) // 1
	live := mk(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP) // 2
	pairedPhone := mk(SessionBrowser, "dev-9", "iPhone", "100.64.0.7")                // 3
	pairedLoopback := mk(SessionBrowser, "dev-2", "This machine · Windows", LoopbackMintIP)
	token := mk(SessionToken, "", "Install token", "")
	alreadyGone := mk(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP)
	expired := mk(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP)
	corrupt := mk(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP)

	ageSession(s, dead, time.Hour)
	ageSession(s, pairedPhone, time.Hour)
	ageSession(s, pairedLoopback, time.Hour)
	ageSession(s, token, time.Hour)
	ageSession(s, alreadyGone, time.Hour)
	ageSession(s, expired, time.Hour)
	if err := s.RevokeSession(alreadyGone); err != nil {
		t.Fatal(err)
	}
	ageSession(s, expired, time.Hour)
	if _, err := s.db.Exec(`UPDATE auth_sessions SET expires_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-time.Hour).Format(time.RFC3339Nano), expired); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE auth_sessions SET last_seen_at = 'not-a-time' WHERE id = ?`, corrupt); err != nil {
		t.Fatal(err)
	}

	stale, err := s.StaleLoopbackBrowserSessions(10 * time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 1 || stale[0].ID != dead {
		t.Fatalf("stale = %+v, want only %s", stale, dead)
	}

	got, err := s.RevokeStaleLoopbackSessions(10 * time.Minute)
	if err != nil || got != 1 {
		t.Fatalf("revoked = %d, %v; want 1", got, err)
	}
	if row, err := s.SessionByID(dead); err != nil || row.RevokedAt == nil {
		t.Fatalf("dead session not revoked: %+v %v", row, err)
	}
	// The live tab (second profile of the same family) is untouched.
	liveRow, err := s.SessionByID(live)
	if err != nil || liveRow.RevokedAt != nil {
		t.Fatalf("live session disturbed: %+v %v", liveRow, err)
	}
	// A second sweep finds nothing — idempotent.
	if n, err := s.RevokeStaleLoopbackSessions(10 * time.Minute); err != nil || n != 0 {
		t.Fatalf("second sweep = %d, %v; want 0", n, err)
	}
}

// The sweep reuses RevokeSession, so each revoked row announces exactly
// one session.revoked event (ADR-0048) — nothing for rows it skips.
func TestRevokeStaleLoopbackSessionsAnnouncesEachRevocation(t *testing.T) {
	s := openTest(t)
	gone, _, _ := s.CreateSession(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP, 90*24*time.Hour)
	kept, _, _ := s.CreateSession(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP, 90*24*time.Hour)
	ageSession(s, gone.ID, time.Hour)
	var got []string
	s.OnEvent = func(ev Event) { got = append(got, ev.Type) }
	if n, err := s.RevokeStaleLoopbackSessions(10 * time.Minute); err != nil || n != 1 {
		t.Fatalf("revoked = %d, %v; want 1", n, err)
	}
	if len(got) != 1 || got[0] != "session.revoked" {
		t.Fatalf("events = %v, want [session.revoked]", got)
	}
	if row, err := s.SessionByID(kept.ID); err != nil || row.RevokedAt != nil {
		t.Fatalf("fresh session disturbed: %+v %v", row, err)
	}
}
