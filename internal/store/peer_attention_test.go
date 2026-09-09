package store

import (
	"fmt"
	"github.com/cfpperche/picode/internal/clilaunch"
	"testing"
)

func TestPeerAttentionClaimsDoNotConsumeOrRepeat(t *testing.T) {
	s := openTest(t)
	_, token, q, qt := peerFixture(t, s)
	m, err := s.SendPeerMessage(token, q.ID, "notice", "hello", "")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SetPeerAttention(m.ID, q.ID, "pending", "attempted"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if ok, err := s.SetPeerAttention(m.ID, q.ID, "pending", "attempted"); err != nil || ok {
		t.Fatal("duplicate attempt", ok, err)
	}
	pending, err := s.PendingPeerAttention()
	if err != nil || len(pending) != 0 {
		t.Fatal("claimed notification scheduled again")
	}
	inbox, err := s.ReadPeerMessages(qt, 0, 50, true)
	if err != nil || len(inbox) != 1 || inbox[0].AckedAt != nil {
		t.Fatal("notification consumed message")
	}
	if ok, err := s.SetPeerAttention(m.ID, q.ID, "attempted", "uncertain"); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if _, err := s.SetPeerAttention(m.ID, q.ID, "uncertain", "pending"); err == nil {
		t.Fatal("ambiguous write allowed automatic retry")
	}
	second, err := s.SendPeerMessage(token, q.ID, "other", "other", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AckPeerMessages(qt, []string{second.ID}); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.SetPeerAttention(second.ID, q.ID, "pending", "attempted"); err != nil || ok {
		t.Fatal("acknowledged mail notified")
	}
	s.RevokePeer(q.ID)
	if _, err := s.SetPeerAttention(m.ID, q.ID, "attempted", "notified"); err == nil {
		t.Fatal("revoked recipient accepted")
	}
}

func TestNativeConversationCannotEnrollTwice(t *testing.T) {
	s := openTest(t)
	_, _, q, _ := peerFixture(t, s)
	terminal, err := s.CreateTerminalIn(q.WorkspaceID, "duplicate", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLaunch(terminal.ID, q.CLI, clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLastSession(terminal.ID, TerminalLastSession{CLI: q.CLI, SessionID: q.SessionKey}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.EnablePeer("terminal", terminal.ID, q.SessionKey); err == nil {
		t.Fatal("same native conversation received two addresses")
	}
	if err := s.RevokePeer(q.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.EnablePeer("terminal", terminal.ID, q.SessionKey); err != nil {
		t.Fatal("revoked address blocked enrollment", err)
	}
}

func TestPeerAttentionBacklogDoesNotStarveOtherRecipients(t *testing.T) {
	s := openTest(t)
	p, pt, q, qt := peerFixture(t, s)
	for n := 0; n < 120; n++ {
		if _, err := s.SendPeerMessage(pt, q.ID, fmt.Sprint("backlog-", n), "pending", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SendPeerMessage(qt, p.ID, "later-recipient", "hello", ""); err != nil {
		t.Fatal(err)
	}
	rows, err := s.PendingPeerAttention()
	if err != nil || len(rows) != 2 || rows[0].RecipientID != q.ID || rows[1].RecipientID != p.ID {
		t.Fatalf("recipient starvation: %+v, %v", rows, err)
	}
	if err := s.RevokePeer(q.ID); err != nil {
		t.Fatal(err)
	}
	rows, err = s.PendingPeerAttention()
	if err != nil || len(rows) != 1 || rows[0].RecipientID != p.ID {
		t.Fatalf("revoked mailbox scheduled: %+v, %v", rows, err)
	}
}

func TestNativeAddressCannotResurrectOnSecondOwner(t *testing.T) {
	s := openTest(t)
	_, _, q, _ := peerFixture(t, s)
	if err := s.SetTerminalLastSession(q.OwnerID, TerminalLastSession{CLI: q.CLI, SessionID: "new-session"}); err != nil {
		t.Fatal(err)
	}
	b, err := s.CreateTerminalIn(q.WorkspaceID, "second owner", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s.SetTerminalLaunch(b.ID, q.CLI, clilaunch.Overrides{})
	s.SetTerminalLastSession(b.ID, TerminalLastSession{CLI: q.CLI, SessionID: q.SessionKey})
	if _, _, err := s.EnablePeer("terminal", b.ID, q.SessionKey); err == nil {
		t.Fatal("temporarily inactive address was duplicated")
	}
	if err := s.SetTerminalLastSession(q.OwnerID, TerminalLastSession{CLI: q.CLI, SessionID: q.SessionKey}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PeerConnection(q.ID); err != nil {
		t.Fatal("original resume lost address", err)
	}
}
