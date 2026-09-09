package store

import (
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func peerFixture(t *testing.T, s *Store) (PeerConnection, string, PeerConnection, string) {
	t.Helper()
	w, a, err := addWorkspaceWithAgent(s, "Peers", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(w.Path, "a.jsonl")
	if _, err = s.UpdateAgent(a.ID, AgentPatch{SessionPath: &path}); err != nil {
		t.Fatal(err)
	}
	tm, err := s.CreateTerminalIn(w.ID, "Codex", w.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetTerminalLaunch(tm.ID, "codex", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	if err = s.SetTerminalLastSession(tm.ID, TerminalLastSession{CLI: "codex", SessionID: "codex-session"}); err != nil {
		t.Fatal(err)
	}
	p, pt, err := s.EnablePeer("agent", a.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	q, qt, err := s.EnablePeer("terminal", tm.ID, "codex-session")
	if err != nil {
		t.Fatal(err)
	}
	return p, pt, q, qt
}
func TestPeerRoundTripAndRestart(t *testing.T) {
	db := filepath.Join(t.TempDir(), "peer.db")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	p, pt, q, qt := peerFixture(t, s)
	var events []Event
	s.OnEvent = func(e Event) { events = append(events, e) }
	m, err := s.SendPeerMessage(pt, q.ID, "retry-key", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.SendPeerMessage(pt, q.ID, "retry-key", "hello", "")
	if err != nil || again.ID != m.ID || len(events) != 1 {
		t.Fatalf("retry %+v %v events=%d", again, err, len(events))
	}
	if _, err = s.SendPeerMessage(pt, q.ID, "retry-key", "changed", ""); !errors.Is(err, ErrPeerConflict) {
		t.Fatal(err)
	}
	for range 2 {
		rows, e := s.ReadPeerMessages(qt, 0, 100, true)
		if e != nil || len(rows) != 1 || rows[0].AckedAt != nil {
			t.Fatalf("read %v %v", rows, e)
		}
	}
	own, _ := s.ReadPeerMessages(pt, 0, 100, true)
	if len(own) != 0 {
		t.Fatal("sender read recipient inbox")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	rows, err := s.ReadPeerMessages(qt, 0, 100, true)
	if err != nil || len(rows) != 1 {
		t.Fatalf("restart %v %v", rows, err)
	}
	reply, err := s.SendPeerMessage(qt, p.ID, "reply", "answer", m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AckPeerMessages(qt, []string{m.ID, reply.ID}); !errors.Is(err, ErrPeerInput) {
		t.Fatal("mixed ack accepted", err)
	}
	rows, _ = s.ReadPeerMessages(qt, 0, 100, true)
	if len(rows) != 1 {
		t.Fatal("partial ack")
	}
	if err = s.AckPeerMessages(qt, []string{m.ID, "unknown"}); !errors.Is(err, ErrPeerInput) {
		t.Fatal(err)
	}
	for range 2 {
		if err = s.AckPeerMessages(qt, []string{m.ID, m.ID}); err != nil {
			t.Fatal(err)
		}
	}
	rows, _ = s.ReadPeerMessages(qt, 0, 100, true)
	if len(rows) != 0 {
		t.Fatal("ack missing")
	}
	rows, _ = s.ReadPeerMessages(qt, 0, 100, false)
	if len(rows) != 1 || rows[0].AckedAt == nil {
		t.Fatal("history missing")
	}
	rows, _ = s.ReadPeerMessages(qt, m.Seq, 100, false)
	if len(rows) != 0 {
		t.Fatal("cursor repeated")
	}
	history, err := s.PeerHistory(p.ID, 0)
	if err != nil || len(history) != 2 {
		t.Fatalf("history %v %v", history, err)
	}
	if err = s.RevokePeer(q.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizePeer(qt); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
	if _, err = s.SendPeerMessage(pt, q.ID, "disabled", "x", ""); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
	// The durable receipt remains available even if recipient was revoked.
	if again, err = s.SendPeerMessage(pt, q.ID, "retry-key", "hello", ""); err != nil || again.ID != m.ID {
		t.Fatal(err)
	}
	history, _ = s.PeerHistory(q.ID, 0)
	if len(history) != 2 {
		t.Fatal("revoke erased history")
	}
}
func TestPeerIsolationAndSessionBinding(t *testing.T) {
	s := openTest(t)
	p, pt, q, qt := peerFixture(t, s)
	foreign, ft, _, _ := peerFixture(t, s)
	contacts, err := s.PeerContacts(pt)
	if err != nil || len(contacts) != 1 || contacts[0].ID != q.ID {
		t.Fatalf("contacts %v %v", contacts, err)
	}
	for _, to := range []string{p.ID, foreign.ID, "unknown"} {
		if _, err = s.SendPeerMessage(pt, to, "key", "body", ""); !errors.Is(err, ErrPeerDenied) {
			t.Fatalf("to %s: %v", to, err)
		}
	}
	outside, err := s.SendPeerMessage(ft, contactsFor(t, s, ft)[0].ID, "other", "body", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SendPeerMessage(pt, q.ID, "reply", "x", outside.ID); !errors.Is(err, ErrPeerInput) {
		t.Fatal(err)
	}
	if _, _, err = s.EnablePeer("agent", p.PeerOwner.OwnerID, "stale"); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
	// Changing the recorded conversation invalidates every operation.
	if err = s.SetTerminalLastSession(q.PeerOwner.OwnerID, TerminalLastSession{CLI: "codex", SessionID: "next-session"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizePeer(qt); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("stale session authorized", err)
	}
	if _, err = s.ReadPeerMessages(qt, 0, 10, true); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
	contacts, _ = s.PeerContacts(pt)
	if len(contacts) != 0 {
		t.Fatal("stale contact listed")
	}
	different := "different"
	if _, err = s.UpdateAgent(p.PeerOwner.OwnerID, AgentPatch{SessionPath: &different}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizePeer(pt); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
}
func contactsFor(t *testing.T, s *Store, token string) []PeerConnection {
	t.Helper()
	v, e := s.PeerContacts(token)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func TestPeerRotateDeleteAndSecrets(t *testing.T) {
	s := openTest(t)
	p, pt, q, _ := peerFixture(t, s)
	owners, e := s.ListPeerOwners()
	if e != nil || len(owners) < 2 {
		t.Fatal(owners, e)
	}
	next, nt, e := s.EnablePeer("agent", p.PeerOwner.OwnerID, p.SessionKey)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AuthorizePeer(pt); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("old token retained")
	}
	if _, e = s.AuthorizePeer(nt); e != nil {
		t.Fatal(e)
	}
	var hash string
	if e = s.db.QueryRow(`SELECT token_hash FROM peer_connections WHERE id=?`, next.ID).Scan(&hash); e != nil || strings.Contains(hash, nt) || hash == nt {
		t.Fatal("secret stored", e)
	}
	_, e = s.SendPeerMessage(nt, q.ID, "x", "private body", "")
	if e != nil {
		t.Fatal(e)
	}
	events, e := s.ListEventsSince(0, 1000)
	if e != nil {
		t.Fatal(e)
	}
	for _, ev := range events {
		if strings.Contains(string(ev.Data), nt) || strings.Contains(string(ev.Data), "private body") {
			t.Fatal("secret/body event leak")
		}
	}
	if e = s.DeleteAgent(p.PeerOwner.OwnerID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AuthorizePeer(nt); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("deleted owner authorized")
	}
}
func TestPeerBoundsAndConcurrentRetry(t *testing.T) {
	s := openTest(t)
	_, pt, q, qt := peerFixture(t, s)
	for _, body := range []string{"", "  ", strings.Repeat("x", 16385), string([]byte{0xff})} {
		if _, e := s.SendPeerMessage(pt, q.ID, "x", body, ""); !errors.Is(e, ErrPeerInput) {
			t.Fatal(e)
		}
	}
	for _, limit := range []int{0, 101} {
		if _, e := s.ReadPeerMessages(qt, 0, limit, true); !errors.Is(e, ErrPeerInput) {
			t.Fatal(e)
		}
	}
	if e := s.AckPeerMessages(qt, nil); !errors.Is(e, ErrPeerInput) {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan PeerMessage, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m, e := s.SendPeerMessage(pt, q.ID, "same", "message", "")
			results <- m
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	id := ""
	for m := range results {
		if id != "" && id != m.ID {
			t.Fatal("duplicate")
		}
		id = m.ID
	}
	// Owned database fixture: fill capacity without a thousand redundant events.
	_, e := s.db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<999) INSERT INTO peer_messages(id,sender_id,recipient_id,request_id,body,created_at) SELECT 'fill-'||x,(SELECT sender_id FROM peer_messages LIMIT 1),?,'fill-'||x,'x','now' FROM n`, q.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SendPeerMessage(pt, q.ID, "overflow", "x", ""); !errors.Is(e, ErrPeerCapacity) {
		t.Fatal(e)
	}
	if _, e = s.SendPeerMessage(pt, q.ID, "same", "message", ""); e != nil {
		t.Fatal("capacity blocked receipt", e)
	}
}

func TestPeerEventFailureRollsBack(t *testing.T) {
	s := openTest(t)
	_, pt, q, _ := peerFixture(t, s)
	_, err := s.db.Exec(`CREATE TRIGGER fail_peer_event BEFORE INSERT ON events WHEN NEW.type='peer.message' BEGIN SELECT RAISE(FAIL,'fixture event failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SendPeerMessage(pt, q.ID, "rollback", "not persisted", ""); err == nil {
		t.Fatal("event failure ignored")
	}
	rows, err := s.PeerHistory(q.ID, 0)
	if err != nil || len(rows) != 0 {
		t.Fatal("message survived failed event", rows, err)
	}
}
func TestPeerTotalCapacityAndCLISwitch(t *testing.T) {
	s := openTest(t)
	p, pt, q, qt := peerFixture(t, s)
	_, err := s.db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<10000) INSERT INTO peer_messages(id,sender_id,recipient_id,request_id,body,created_at,acked_at) SELECT 'fill-'||x,?,?,'fill-'||x,'x','now','now' FROM n`, p.ID, q.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SendPeerMessage(pt, q.ID, "overflow", "x", ""); !errors.Is(err, ErrPeerCapacity) {
		t.Fatal(err)
	}
	if err = s.SetTerminalLaunch(q.OwnerID, "pi", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.EnablePeer("terminal", q.OwnerID, q.SessionKey); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("previous CLI pin reused", err)
	}
	if _, err = s.AuthorizePeer(qt); !errors.Is(err, ErrPeerDenied) {
		t.Fatal(err)
	}
}
