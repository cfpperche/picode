package server

// ADR-0106's maintenance concern: private setup must not outlive its
// connection. The sweep reconciles the communication directory against the
// store — live connections keep their files, everything else is removed.
import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/store"
)

func TestPeerLaunchOrphanSweep(t *testing.T) {
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	w, err := st.AddWorkspace("Sweep", data)
	if err != nil {
		t.Fatal(err)
	}
	var live, revoked, gone string
	for i, holder := range []struct {
		name string
		out  *string
	}{
		{"Live", &live}, {"Revoked", &revoked}, {"Deleted", &gone},
	} {
		a, err := st.AddAgent(w.ID, holder.name, "")
		if err != nil {
			t.Fatal(err)
		}
		session := filepath.Join(data, holder.name+".jsonl")
		if _, err = st.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &session}); err != nil {
			t.Fatal(err)
		}
		p, token, err := st.EnablePeer("agent", a.ID, session)
		if err != nil {
			t.Fatal(err)
		}
		if err = communication.SaveLaunch(data, communication.LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + communication.Path}); err != nil {
			t.Fatal(err)
		}
		*holder.out = p.ID
		if i == 1 {
			if err = st.RevokePeer(p.ID); err != nil {
				t.Fatal(err)
			}
		}
	}
	// A crash between revoke and file removal, and unrelated directory junk.
	stray := filepath.Join(data, "communication", "peer_STRAYLEAK")
	if err = os.MkdirAll(filepath.Join(stray, "keep"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(stray, "keep", "connection.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(data, "communication", "notes")
	if err = os.MkdirAll(note, 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(data, "communication", "README"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	// First sweep: only the revoked connection and the crash leak are orphans;
	// the third agent is still alive and keeps its setup.
	sweepPeerLaunchFiles(Deps{Store: st, DataDir: data})

	for id, want := range map[string]bool{live: true, revoked: false, gone: true, "peer_STRAYLEAK": false} {
		_, err := os.Stat(filepath.Join(data, "communication", id))
		if got := !os.IsNotExist(err); got != want {
			t.Fatalf("connection %s kept=%v, want %v", id, got, want)
		}
	}
	if _, err := os.Stat(note); err != nil {
		t.Fatal("unrelated directory removed")
	}
	if _, err := os.Stat(filepath.Join(data, "communication", "README")); err != nil {
		t.Fatal("unrelated file removed")
	}
	// A deleted owner cascades its connection row, then the sweep follows.
	a, err := st.AddAgent(w.ID, "Doomed", "")
	if err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(data, "Doomed.jsonl")
	if _, err = st.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &session}); err != nil {
		t.Fatal(err)
	}
	p, token, err := st.EnablePeer("agent", a.ID, session)
	if err != nil {
		t.Fatal(err)
	}
	if err = communication.SaveLaunch(data, communication.LaunchConfig{Connection: p, Token: token, URL: "http://localhost" + communication.Path}); err != nil {
		t.Fatal(err)
	}
	if err = st.DeleteAgent(a.ID); err != nil {
		t.Fatal(err)
	}
	sweepPeerLaunchFiles(Deps{Store: st, DataDir: data})
	if _, err := os.Stat(filepath.Join(data, "communication", p.ID)); !os.IsNotExist(err) {
		t.Fatal("deleted owner kept setup files")
	}
}
