package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
)

func TestWorkspaceParticipationAPI(t *testing.T) {
	ts, _, home := cleanupServer(t)
	ws := cliRequest(t, ts, "POST", "/api/workspaces", map[string]any{"name": "Communication", "path": home}, 201)
	id := ws["id"].(string)
	a := cliRequest(t, ts, "POST", "/api/workspaces/"+id+"/agents", map[string]any{"name": "Sender"}, 201)
	aid := a["id"].(string)
	endpoint := "/api/communication/workspaces/" + id
	body := map[string]any{"participants": []any{map[string]any{"kind": "agent", "ownerId": aid, "enabled": true, "revision": 0}}}
	cliRequest(t, ts, "PUT", endpoint+"/participants", body, 202)
	snapshot := cliRequest(t, ts, "GET", "/api/communication/workspaces", nil, 200)
	rows := snapshot["participants"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["enabled"] != true {
		t.Fatal(snapshot)
	}
	if len(snapshot["connections"].([]any)) != 0 {
		t.Fatal("minted before native identity")
	}
	cliRequest(t, ts, "PUT", endpoint+"/participants", body, 409)
	cliRequest(t, ts, "POST", endpoint+"/open", map[string]any{"kind": "agent", "ownerId": aid, "revision": 0}, 409)
	cliRequest(t, ts, "POST", endpoint+"/test", map[string]any{"from": "missing", "to": "missing"}, 409)
	cliRequest(t, ts, "GET", endpoint+"/history?before=-1", nil, 400)
	history := cliRequest(t, ts, "GET", endpoint+"/history", nil, 200)
	if len(history["messages"].([]any)) != 0 {
		t.Fatal(history)
	}
}

func TestPeerCheckWorkerDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name, initial, want string
		age                 time.Duration
		revoke, busy, proof bool
	}{
		{name: "stopped sender", initial: "pending", want: "pending"},
		{name: "busy control", initial: "pending", want: "pending", busy: true},
		{name: "crash after claim", initial: "attempted", want: "attempted"},
		{name: "submitted once", initial: "running", want: "running"},
		{name: "ambiguous submission", initial: "uncertain", want: "uncertain"},
		{name: "expired pending", initial: "pending", want: "expired", age: 6 * time.Minute},
		{name: "expired claimed", initial: "attempted", want: "expired", age: 6 * time.Minute},
		{name: "revoked recipient", initial: "running", want: "cancelled", revoke: true},
		{name: "native proof", initial: "running", want: "passed", proof: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := t.TempDir()
			st, err := store.Open(filepath.Join(data, "state.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			w, err := st.AddWorkspace("Check", data)
			if err != nil {
				t.Fatal(err)
			}
			var peers []store.PeerConnection
			var tokens []string
			for _, name := range []string{"Sender", "Receiver"} {
				a, e := st.AddAgent(w.ID, name, "")
				if e != nil {
					t.Fatal(e)
				}
				session := filepath.Join(data, name+".jsonl")
				if _, e = st.UpdateAgent(a.ID, store.AgentPatch{SessionPath: &session}); e != nil {
					t.Fatal(e)
				}
				p, token, e := st.EnablePeer("agent", a.ID, session)
				if e != nil {
					t.Fatal(e)
				}
				peers = append(peers, p)
				tokens = append(tokens, token)
			}
			check, err := st.CreatePeerCheck(w.ID, peers[0].ID, peers[1].ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.initial != "pending" {
				if _, err = st.SetPeerCheckPhase(check.ID, "pending", tc.initial); err != nil {
					t.Fatal(err)
				}
			}
			if tc.revoke {
				if err = st.RevokePeer(peers[1].ID); err != nil {
					t.Fatal(err)
				}
			}
			if tc.proof {
				m, e := st.SendPeerMessage(tokens[0], peers[1].ID, check.ID, "test", "")
				if e != nil {
					t.Fatal(e)
				}
				r, e := st.SendPeerMessage(tokens[1], peers[0].ID, "reply", "OK", m.ID)
				if e != nil {
					t.Fatal(e)
				}
				if e = st.AckPeerMessages(tokens[1], []string{m.ID}); e != nil {
					t.Fatal(e)
				}
				if e = st.AckPeerMessages(tokens[0], []string{r.ID}); e != nil {
					t.Fatal(e)
				}
			}
			if tc.age > 0 {
				if _, err = st.SendPeerMessage(tokens[0], peers[1].ID, check.ID, "unfinished test", ""); err != nil {
					t.Fatal(err)
				}
			}
			deps := Deps{Store: st, DataDir: data, Runtime: rpc.NewRuntime("must-not-start", st, nil), Replies: NewTuiReplies()}
			if tc.busy {
				release, e := deps.Replies.Controls.TryBeginMutation(peers[0].OwnerID)
				if e != nil {
					t.Fatal(e)
				}
				defer release()
			}
			for range 2 {
				reconcilePeerChecksAt(context.Background(), deps, time.Now().Add(tc.age))
			}
			checks, e := st.ListPeerChecks()
			if e != nil || len(checks) != 1 || checks[0].Phase != tc.want {
				t.Fatalf("checks=%+v err=%v want=%s", checks, e, tc.want)
			}
			if deps.Runtime.Active(peers[0].OwnerID) || deps.Runtime.Active(peers[1].OwnerID) {
				t.Fatal("diagnostic started a stopped participant")
			}
			history, e := st.PeerWorkspaceHistory(w.ID, 0)
			if e != nil {
				t.Fatal(e)
			}
			if tc.age > 0 {
				if len(history) != 1 || history[0].Body != "unfinished test" {
					t.Fatal("expiry removed activity", history)
				}
				if _, err = st.CreatePeerCheck(w.ID, peers[0].ID, peers[1].ID); err != nil {
					t.Fatal("expired check blocked another test", err)
				}
			} else if !tc.proof && len(history) != 0 {
				t.Fatal("worker impersonated a native sender")
			}
		})
	}
}

func TestReceiverConnectionRequiresCurrentProcess(t *testing.T) {
	r := NewTuiReplies()
	r.helloConnectionProcess("a", "session", "peer_a", "first")
	if r.receiverConnection("a", "session", "first") != "peer_a" {
		t.Fatal("current receiver lost")
	}
	if r.receiverConnection("a", "session", "second") != "" {
		t.Fatal("old process proved new readiness")
	}
	r.helloConnectionProcess("a", "session", "", "first")
	if r.receiverConnection("a", "session", "first") != "" {
		t.Fatal("disposed registration retained")
	}
}

func TestPeerStopReceiptFailsClosed(t *testing.T) {
	deps := Deps{DataDir: t.TempDir()}
	if pending, e := peerStopPending(deps, "fixture"); pending || e != nil {
		t.Fatal(pending, e)
	}
	token := processStartToken(os.Getpid())
	if token == "" {
		t.Skip("process ownership tokens unavailable")
	}
	if e := savePeerStop(deps, "fixture", map[int]string{os.Getpid(): token}); e != nil {
		t.Fatal(e)
	}
	// A new registry/server sees the same outstanding writer after restart.
	if pending, e := peerStopPending(Deps{DataDir: deps.DataDir}, "fixture"); !pending || e != nil {
		t.Fatal(pending, e)
	}
	if e := savePeerStop(deps, "fixture", map[int]string{os.Getpid(): token + "-reused"}); e != nil {
		t.Fatal(e)
	}
	if pending, e := peerStopPending(deps, "fixture"); pending || e != nil {
		t.Fatal("PID reuse blocked forever", pending, e)
	}
	path := peerStopPath(deps, "fixture")
	if e := os.WriteFile(path, []byte("broken"), 0600); e != nil {
		t.Fatal(e)
	}
	if pending, e := peerStopPending(deps, "fixture"); !pending || e == nil {
		t.Fatal("corrupt receipt allowed start")
	}
	if _, e := os.Stat(filepath.Dir(path)); e != nil {
		t.Fatal(e)
	}
}

func TestPeerClaudeResumable(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"empty", "", false},
		{"startup metadata", `{"type":"queue-operation","sessionId":"s"}`, false},
		{"wrong session", `{"type":"user","sessionId":"other","message":{"role":"user","content":"hello"}}`, false},
		{"sidechain", `{"type":"user","sessionId":"s","isSidechain":true,"message":{"role":"user","content":"hello"}}`, false},
		{"metadata user", `{"type":"user","sessionId":"s","isMeta":true,"message":{"role":"user","content":"hello"}}`, false},
		{"missing message", `{"type":"user","sessionId":"s"}`, false},
		{"null message", `{"type":"user","sessionId":"s","message":null}`, false},
		{"partial record", `{"type":"user","sessionId":"s","message":`, false},
		{"empty object", `{"type":"user","sessionId":"s","message":{}}`, false},
		{"empty content", `{"type":"user","sessionId":"s","message":{"role":"user","content":[]}}`, false},
		{"wrong role", `{"type":"user","sessionId":"s","message":{"role":"assistant","content":"hello"}}`, false},
		{"scalar message", `{"type":"user","sessionId":"s","message":false}`, false},
		{"oversized line", strings.Repeat("x", 1<<20) + "\n" + `{"type":"user","sessionId":"s","message":{"role":"user","content":"hello"}}`, false},
		{"scan limit", strings.Repeat("{}\n", (4<<20)/3+1) + `{"type":"user","sessionId":"s","message":{"role":"user","content":"hello"}}`, false},
		{"metadata then user", "{\"type\":\"queue-operation\"}\n" + `{"type":"user","sessionId":"s","message":{"role":"user","content":"hello"}}`, true},
		{"saved user", `{"type":"user","sessionId":"s","message":{"role":"user","content":"hello"}}`, true},
		{"saved assistant", `{"type":"assistant","sessionId":"s","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "session.jsonl")
			if err := os.WriteFile(p, []byte(tc.body+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if got := peerClaudeResumable(p, "s"); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	dir := t.TempDir()
	unreadable := filepath.Join(dir, "unreadable.jsonl")
	if err := os.WriteFile(unreadable, []byte(`{"type":"user","sessionId":"s","message":{"role":"user","content":"hello"}}`), 0000); err != nil {
		t.Fatal(err)
	}
	if file, err := os.Open(unreadable); err != nil {
		if peerClaudeResumable(unreadable, "s") {
			t.Fatal("unreadable accepted")
		}
	} else {
		file.Close()
	} // Root can read mode 0000.
	if peerClaudeResumable(dir, "s") || peerClaudeResumable(filepath.Join(dir, "missing"), "s") || peerClaudeResumable("", "s") {
		t.Fatal("missing/nonregular transcript accepted")
	}
}
