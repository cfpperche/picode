package server

// Row 9 of the onboarding table: a missing Pi adapter is a visible, bounded
// error that mints nothing, and installing the adapter through Packages
// recovers on a later worker pass without re-selecting the participant.
import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func TestPeerOnboardingAdapterRepair(t *testing.T) {
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	w, err := st.AddWorkspace("Repair", data)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgent(w.ID, "Receiver", "")
	if err != nil {
		t.Fatal(err)
	}
	session := filepath.Join(data, "native.jsonl")
	if _, err = st.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &session}); err != nil {
		t.Fatal(err)
	}
	if err = st.SetPeerParticipants(w.ID, []store.PeerSelection{{Kind: "agent", OwnerID: a.ID, Enabled: true}}); err != nil {
		t.Fatal(err)
	}

	old := pipkg.UserDir
	piDir := t.TempDir()
	pipkg.UserDir = func() string { return piDir }
	t.Cleanup(func() { pipkg.UserDir = old })

	deps := Deps{Store: st, DataDir: data}
	reconcilePeerParticipants(context.Background(), deps)

	got, err := st.PeerParticipant("agent", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != "adapter-missing" || got.Problem != errPeerAdapterMissing.Error() {
		t.Fatalf("missing adapter state = %q / %q", got.Phase, got.Problem)
	}
	if peers, e := st.ListPeerConnections(); e != nil || len(peers) != 0 {
		t.Fatal("capability minted without the adapter", peers, e)
	}

	// The Packages install lands the adapter under the user dir.
	adapter := filepath.Join(piDir, "npm", "node_modules", "pi-mcp-adapter", "index.ts")
	if err = os.MkdirAll(filepath.Dir(adapter), 0o755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(adapter, []byte("export default () => {};\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reconcilePeerParticipants(context.Background(), deps)

	got, err = st.PeerParticipant("agent", a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase == "adapter-missing" || strings.Contains(got.Problem, "adapter") {
		t.Fatal("repair did not leave the error state", got.Phase, got.Problem)
	}
	peers, err := st.ListPeerConnections()
	if err != nil || len(peers) != 1 || peers[0].SessionKey != session || !peers[0].Active {
		t.Fatal("repair did not mint the connection", peers, err)
	}
	loaded, err := communication.LoadLaunch(st, data, "agent", a.ID)
	if err != nil || loaded == nil || loaded.Connection.ID != peers[0].ID || loaded.Adapter != adapter {
		t.Fatal("private setup missing after repair", loaded, err)
	}
	// A later pass is idempotent: the same capability and setup survive.
	reconcilePeerParticipants(context.Background(), deps)
	again, err := st.ListPeerConnections()
	if err != nil || len(again) != 1 || again[0].ID != peers[0].ID {
		t.Fatal("repair pass duplicated the capability", again, err)
	}
}
