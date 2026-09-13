package store

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func TestPeerParticipationDecisionTable(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, at, b, _ := peerFixture(t, s)
	selection := PeerSelection{Kind: a.Kind, OwnerID: a.OwnerID, Enabled: true}
	if e = s.SetPeerParticipants(a.WorkspaceID, []PeerSelection{selection}); e != nil {
		t.Fatal(e)
	}
	p, e := s.PeerParticipant(a.Kind, a.OwnerID)
	if e != nil || !p.Enabled || p.Revision != 1 {
		t.Fatal(p, e)
	}
	// Repeated reconciliation reuses the current capability, never rotates it.
	got, token, e := s.EnsureParticipantPeer(p, a.SessionKey)
	if e != nil || got.ID != a.ID || token != "" {
		t.Fatal(got, token, e)
	}
	if e = s.SetPeerParticipants(a.WorkspaceID, []PeerSelection{selection}); !errors.Is(e, ErrPeerConflict) {
		t.Fatal("stale selection", e)
	}
	// A stale multi-row request must not partially apply its valid first row.
	if e = s.SetPeerParticipants(a.WorkspaceID, []PeerSelection{{Kind: b.Kind, OwnerID: b.OwnerID, Enabled: true}, selection}); !errors.Is(e, ErrPeerConflict) {
		t.Fatal(e)
	}
	if _, e = s.PeerParticipant(b.Kind, b.OwnerID); e == nil {
		t.Fatal("partial selection")
	}
	if e = s.SetPeerParticipants("another-workspace", []PeerSelection{{Kind: a.Kind, OwnerID: a.OwnerID, Enabled: true, Revision: 1}}); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("cross workspace", e)
	}
	next := a.SessionKey + "-new"
	if _, e = s.UpdateAgent(a.OwnerID, AgentPatch{SessionPath: &next}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AuthorizePeer(at); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("old credential survived /new")
	}
	if _, _, e = s.EnsureParticipantPeer(p, a.SessionKey); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("stale identity minted", e)
	}
	newer, nt, e := s.EnsureParticipantPeer(p, next)
	if e != nil || nt == "" || newer.ID == a.ID {
		t.Fatal(newer, e)
	}
	if _, e = s.SetPeerPreparation(p, a.SessionKey, "connected", "", a.ID); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("stale result accepted", e)
	}
	if e = s.SetPeerParticipants(a.WorkspaceID, []PeerSelection{{Kind: a.Kind, OwnerID: a.OwnerID, Revision: 1, Enabled: false}}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AuthorizePeer(nt); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("disable left capability live")
	}
	if _, _, e = s.EnsureParticipantPeer(p, next); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("late worker recreated capability", e)
	}
	if changed, e := s.SetPeerPreparation(p, next, "connected", "", newer.ID); e != nil || changed {
		t.Fatal("late worker state", changed, e)
	}
}

func TestPeerParticipationDoesNotTransferToAnotherWorkspace(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	agent, agentToken, terminal, terminalToken := peerFixture(t, s)
	t.Run("agent", func(t *testing.T) {
		assertConsentDoesNotFollowWorkspaceMove(t, s, agent, agentToken, `UPDATE agents SET workspace_id=? WHERE id=?`)
	})
	t.Run("terminal", func(t *testing.T) {
		assertConsentDoesNotFollowWorkspaceMove(t, s, terminal, terminalToken, `UPDATE terminals SET workspace_id=? WHERE id=?`)
	})
}

// Owners have no public move API; the grant is keyed to the workspace stored on
// the owner row, so a workspace_id change must not keep the old capability.
func assertConsentDoesNotFollowWorkspaceMove(t *testing.T, s *Store, owner PeerConnection, token, moveSQL string) {
	t.Helper()
	if err := s.SetPeerParticipants(owner.WorkspaceID, []PeerSelection{{Kind: owner.Kind, OwnerID: owner.OwnerID, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	p, err := s.PeerParticipant(owner.Kind, owner.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	dest, err := s.AddWorkspace("moved-"+owner.Kind, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(moveSQL, dest.ID, owner.OwnerID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizePeer(token); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("workspace move retained authentication", err)
	}
	if _, _, err = s.EnsureParticipantPeer(p, owner.SessionKey); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("consent transferred to the new workspace", err)
	}
	if _, err = s.SetPeerPreparation(p, owner.SessionKey, "connected", "", owner.ID); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("stale worker reported readiness after move", err)
	}
	if err = s.SetPeerParticipants(owner.WorkspaceID, []PeerSelection{{Kind: owner.Kind, OwnerID: owner.OwnerID, Revision: p.Revision, Enabled: true}}); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("old workspace still accepted the moved owner", err)
	}
	stale, err := s.PeerParticipant(owner.Kind, owner.OwnerID)
	if err != nil || !stale.Enabled || stale.WorkspaceID != owner.WorkspaceID {
		t.Fatal("move rewrote consent", stale, err)
	}
	if err = s.SetPeerParticipants(dest.ID, []PeerSelection{{Kind: owner.Kind, OwnerID: owner.OwnerID, Revision: stale.Revision, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	next, err := s.PeerParticipant(owner.Kind, owner.OwnerID)
	if err != nil || next.WorkspaceID != dest.ID || !next.Enabled {
		t.Fatal(next, err)
	}
	got, fresh, err := s.EnsureParticipantPeer(next, owner.SessionKey)
	if err != nil || got.ID == owner.ID || got.WorkspaceID != dest.ID || fresh == "" {
		t.Fatal("new workspace reused the old capability", got, fresh, err)
	}
	if _, err = s.AuthorizePeer(token); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("old credential survived re-selection", err)
	}
}

func TestPeerParticipationDoesNotTransferToAnotherCLI(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, _, terminal, token := peerFixture(t, s)
	if err = s.SetPeerParticipants(terminal.WorkspaceID, []PeerSelection{{Kind: terminal.Kind, OwnerID: terminal.OwnerID, Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	p, err := s.PeerParticipant(terminal.Kind, terminal.OwnerID)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetTerminalLaunch(terminal.OwnerID, "hermes", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AuthorizePeer(token); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("CLI change retained authentication", err)
	}
	if _, _, err = s.EnsureParticipantPeer(p, terminal.SessionKey); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("consent transferred to a different CLI", err)
	}
	if _, err = s.SetPeerPreparation(p, terminal.SessionKey, "connected", "", terminal.ID); !errors.Is(err, ErrPeerDenied) {
		t.Fatal("stale worker reported readiness", err)
	}
}
func TestPeerParticipationBeforeConversationAndOwnerDeletion(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	w, a, e := addWorkspaceWithAgent(s, "fresh", t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetPeerParticipants(w.ID, []PeerSelection{{Kind: "agent", OwnerID: a.ID, Enabled: true}}); e != nil {
		t.Fatal(e)
	}
	p, _ := s.PeerParticipant("agent", a.ID)
	if _, _, e = s.EnsureParticipantPeer(p, ""); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("empty session minted", e)
	}
	peers, _ := s.ListPeerConnections()
	if len(peers) != 0 {
		t.Fatal(peers)
	}
	if _, e = s.SetPeerPreparation(p, "", "waiting-conversation", "", ""); e != nil {
		t.Fatal(e)
	}
	if e = s.DeleteAgent(a.ID); e != nil {
		t.Fatal(e)
	}
	prefs, _ := s.ListPeerParticipants()
	if len(prefs) != 0 {
		t.Fatal("orphan preference", prefs)
	}
}
func TestPeerNativeCheckNeedsRoundTripAndBothAcknowledgements(t *testing.T) {
	s, e := Open(filepath.Join(t.TempDir(), "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	a, at, b, bt := peerFixture(t, s)
	c, e := s.CreatePeerCheck(a.WorkspaceID, a.ID, b.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreatePeerCheck(a.WorkspaceID, a.ID, b.ID); !errors.Is(e, ErrPeerConflict) {
		t.Fatal("duplicate pending check", e)
	}
	if _, e = s.CreatePeerCheck("other", a.ID, b.ID); !errors.Is(e, ErrPeerDenied) {
		t.Fatal("cross workspace", e)
	}
	if request, e := s.PeerCheckRequest(at); e != nil || request.ID != c.ID {
		t.Fatal(request, e)
	}
	if request, e := s.PeerCheckRequest(bt); e != nil || request != nil {
		t.Fatal("receiver got sender instruction", request, e)
	}
	if s.PeerCheckPassed(c) {
		t.Fatal("server setup counted as native proof")
	}
	m, e := s.SendPeerMessage(at, b.ID, c.ID, "diagnostic", "")
	if e != nil {
		t.Fatal(e)
	}
	if request, e := s.PeerCheckRequest(at); e != nil || request != nil {
		t.Fatal("sent diagnostic requested another send", e)
	}
	if s.PeerCheckPassed(c) {
		t.Fatal("send counted as roundtrip")
	}
	reply, e := s.SendPeerMessage(bt, a.ID, "reply", "OK", m.ID)
	if e != nil {
		t.Fatal(e)
	}
	s.AckPeerMessages(bt, []string{m.ID})
	if s.PeerCheckPassed(c) {
		t.Fatal("one ACK counted as both")
	}
	s.AckPeerMessages(at, []string{reply.ID})
	if !s.PeerCheckPassed(c) {
		t.Fatal("real roundtrip not recognized")
	}
	if ok, e := s.SetPeerCheckPhase(c.ID, "pending", "passed"); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if request, e := s.PeerCheckRequest(at); e != nil || request != nil {
		t.Fatal("finished diagnostic kept prompting", e)
	}
	history, e := s.PeerWorkspaceHistory(a.WorkspaceID, 0)
	if e != nil || len(history) != 2 {
		t.Fatal(history, e)
	}
	history, e = s.PeerWorkspaceHistory("other", 0)
	if e != nil || len(history) != 0 {
		t.Fatal("history crossed workspace")
	}
}
