package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerPeerOnboarding(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/communication/workspaces/{id}/history", func(w http.ResponseWriter, r *http.Request) {
		before := int64(0)
		var e error
		if v := r.URL.Query().Get("before"); v != "" {
			before, e = strconv.ParseInt(v, 10, 64)
			if e != nil || before < 0 {
				writeErr(w, 400, "Invalid history cursor.")
				return
			}
		}
		rows, e := deps.Store.PeerWorkspaceHistory(r.PathValue("id"), before)
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"messages": rows})
	})

	mux.HandleFunc("GET /api/communication/workspaces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		owners, e := deps.Store.ListPeerOwners()
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		participants, e := deps.Store.ListPeerParticipants()
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		workspaces, e := deps.Store.ListWorkspaces()
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		if workspaces == nil {
			workspaces = []store.Workspace{}
		}
		peers, e := deps.Store.ListPeerConnections()
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		checks, e := deps.Store.ListPeerChecks()
		if e != nil {
			writeStoreErr(w, e)
			return
		}
		live := map[string]string{}
		for _, o := range owners {
			key := o.Kind + ":" + o.OwnerID
			if o.Kind == "agent" && deps.Runtime != nil {
				if a := deps.Runtime.Get(o.OwnerID); a != nil {
					v := a.Snapshot()
					live[key] = "open"
					if v.Streaming {
						live[key] = "working"
					}
					if v.Waiting {
						live[key] = "needs-you"
					}
				}
			} else if rt, ok := deps.TermRuntimes.Get(o.OwnerID); ok && processAlive(rt) {
				live[key] = "open"
				if v, ok := deps.TermStates.Get(o.OwnerID); ok && v.RunID == rt.RunID {
					live[key] = v.State
				}
			}
		}
		writeJSON(w, 200, map[string]any{"owners": owners, "participants": participants, "workspaces": workspaces, "connections": peers, "checks": checks, "live": live})
	})
	mux.HandleFunc("PUT /api/communication/workspaces/{id}/participants", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Participants []store.PeerSelection `json:"participants"`
		}
		if !readCLIJSON(w, r, &in) {
			return
		}
		for _, p := range in.Participants {
			if p.Enabled {
				owners, e := deps.Store.ListPeerOwners()
				if e != nil {
					writeStoreErr(w, e)
					return
				}
				for _, o := range owners {
					if o.Kind == p.Kind && o.OwnerID == p.OwnerID && !communication.LaunchSupported(o.CLI) {
						writeErr(w, 400, "This participant cannot connect yet.")
						return
					}
				}
			}
		}
		if e := deps.Store.SetPeerParticipants(r.PathValue("id"), in.Participants); e != nil {
			writeErr(w, 409, "Participants changed. Refresh and try again.")
			return
		}
		writeJSON(w, 202, map[string]any{"ok": true})
	})
	mux.HandleFunc("POST /api/communication/workspaces/{id}/test", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if !readCLIJSON(w, r, &in) {
			return
		}
		c, e := deps.Store.CreatePeerCheck(r.PathValue("id"), in.From, in.To)
		if e != nil {
			writeErr(w, 409, "Connect two participants and wait for the current test to finish.")
			return
		}
		writeJSON(w, 202, c)
	})
	mux.HandleFunc("POST /api/communication/workspaces/{id}/open", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Kind      string `json:"kind"`
			OwnerID   string `json:"ownerId"`
			Revision  int64  `json:"revision"`
			Reconnect bool   `json:"reconnect"`
		}
		if !readCLIJSON(w, r, &in) {
			return
		}
		p, e := deps.Store.PeerParticipant(in.Kind, in.OwnerID)
		if e != nil || !p.Enabled || p.Revision != in.Revision || p.WorkspaceID != r.PathValue("id") {
			writeErr(w, 409, "Participation changed. Refresh and try again.")
			return
		}
		if p.Kind == "terminal" {
			unlock := terminalLock(deps, p.OwnerID)
			defer unlock()
			t, e := deps.Store.GetTerminal(p.OwnerID)
			if e != nil || deps.Tmux == nil {
				writeErr(w, 409, "Terminal control is unavailable.")
				return
			}
			name := tmux.ShellSessionName(t.ID)
			live, e := deps.Tmux.HasSession(r.Context(), name)
			if e != nil {
				writeErr(w, 500, e.Error())
				return
			}
			if !live {
				launch, e := deps.Store.TerminalLaunch(t.ID)
				if e != nil || launch == nil {
					writeErr(w, 409, "Choose an Agent CLI for this terminal.")
					return
				}
				o, e := currentPeerOwner(deps, p.Kind, p.OwnerID)
				if e != nil || !peerConsentCurrent(deps, p, o.SessionKey) {
					writeErr(w, 409, "Conversation changed.")
					return
				}
				if o.SessionKey != "" {
					if _, e = prepareParticipantConnection(deps, p, o); e != nil {
						writeErr(w, 409, e.Error())
						return
					}
					args := terminalResumeArgs(launch.LastSession)
					if len(args) == 0 {
						writeErr(w, 409, "This conversation cannot be resumed.")
						return
					}
					copy := *launch
					copy.Overrides.Args = &args
					launch = &copy
				}
				prepared, e := prepareCLITerminal(deps, t.Cwd, launch)
				if e != nil {
					writeErr(w, 409, e.Error())
					return
				}
				defer prepared.discard()
				if e = prepared.start(deps, r, name, t.Cwd); e != nil {
					writeErr(w, 409, e.Error())
					return
				}
				publishTerminalState(deps, r, t, true)
			}
		} else {
			if deps.Runtime == nil || deps.Replies == nil {
				writeErr(w, 503, "Agent control is unavailable.")
				return
			}
			release, e := deps.Replies.Controls.TryBeginMutation(p.OwnerID)
			if e != nil {
				writeErr(w, 409, e.Error())
				return
			}
			defer release()
			a, e := deps.Store.GetAgent(p.OwnerID)
			if e != nil {
				writeStoreErr(w, e)
				return
			}
			if deps.runMode(r, p.OwnerID) == modeInteractive {
				writeErr(w, 409, "This agent is open in a terminal. Close that terminal before starting the managed view.")
				return
			}
			if in.Reconnect {
				o, err := currentPeerOwner(deps, p.Kind, p.OwnerID)
				if err != nil || o.SessionKey == "" || !peerConsentCurrent(deps, p, o.SessionKey) {
					writeErr(w, 409, "Conversation changed.")
					return
				}
				if _, err = prepareParticipantConnection(deps, p, o); err != nil {
					writeErr(w, 409, err.Error())
					return
				}
				if ma := deps.Runtime.Get(p.OwnerID); ma != nil {
					ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
					err = deps.Runtime.StopIdle(ctx, ma, o.SessionKey)
					cancel()
					if err != nil {
						writeErr(w, 409, "Wait for the current action or approval to finish, then reconnect.")
						return
					}
				}
				if !peerConsentCurrent(deps, p, o.SessionKey) {
					writeErr(w, 409, "Participation changed.")
					return
				}
			}
			if !deps.Runtime.Active(p.OwnerID) {
				_, cwd, e := deps.agentHome(a)
				if e != nil {
					writeErr(w, 409, e.Error())
					return
				}
				if e = deps.Runtime.Start(p.OwnerID, cwd); e != nil {
					writeErr(w, 409, e.Error())
					return
				}
				_ = deps.Store.SetAgentRuntimeMode(p.OwnerID, store.StatusRunning, "managed")
			}
		}
		writeJSON(w, 202, map[string]any{"ok": true})
	})

}

func peerConsentCurrent(deps Deps, p store.PeerParticipant, session string) bool {
	v, e := deps.Store.PeerParticipant(p.Kind, p.OwnerID)
	if e != nil || !v.Enabled || v.Revision != p.Revision {
		return false
	}
	owners, e := deps.Store.ListPeerOwners()
	if e != nil {
		return false
	}
	for _, o := range owners {
		if o.Kind == p.Kind && o.OwnerID == p.OwnerID {
			return o.CLI == p.CLI && o.WorkspaceID == p.WorkspaceID && o.SessionKey == session
		}
	}
	return false
}

func prepareParticipantConnection(deps Deps, p store.PeerParticipant, o store.PeerOwner) (store.PeerConnection, error) {
	adapter := ""
	if o.CLI == "pi" {
		adapter = filepath.Join(pipkg.UserDir(), "npm", "node_modules", "pi-mcp-adapter", "index.ts")
		if _, e := os.Stat(adapter); e != nil {
			return store.PeerConnection{}, errors.New("Install the Pi connection adapter in Packages.")
		}
	}
	extras := []string{os.Getenv("NODE_EXTRA_CA_CERTS"), os.Getenv("CODEX_CA_CERTIFICATE"), os.Getenv("SSL_CERT_FILE")}
	if o.Kind == "terminal" {
		v, e := deps.Store.TerminalLaunch(o.OwnerID)
		if e != nil {
			return store.PeerConnection{}, e
		}
		if v == nil {
			return store.PeerConnection{}, errors.New("Choose an Agent CLI for this terminal.")
		}
		_, cfg, _, e := resolvedTerminalLaunch(deps, v)
		if e != nil {
			return store.PeerConnection{}, e
		}
		if !cfg.Integration {
			return store.PeerConnection{}, errors.New("Enable terminal integration in Agent CLIs.")
		}
		for _, k := range []string{"NODE_EXTRA_CA_CERTS", "CODEX_CA_CERTIFICATE", "SSL_CERT_FILE"} {
			extras = append(extras, cfg.Env[k])
		}
	}
	ca, e := communication.LocalTrust(deps.DataDir, loopbackURL(deps)+communication.Path, extras)
	if e != nil {
		return store.PeerConnection{}, e
	}
	c, token, e := deps.Store.EnsureParticipantPeer(p, o.SessionKey)
	if e != nil {
		return c, e
	}
	peers, err := deps.Store.ListPeerConnections()
	if err != nil {
		return c, err
	}
	for _, old := range peers {
		if old.Kind == p.Kind && old.OwnerID == p.OwnerID && old.RevokedAt != nil {
			if err = communication.RemoveLaunch(deps.DataDir, old.ID); err != nil {
				return c, errors.New("Could not remove old connection setup. Try again.")
			}
		}
	}
	if token == "" {
		if loaded, e := communication.LoadLaunch(deps.Store, deps.DataDir, p.Kind, p.OwnerID); e != nil || loaded == nil || loaded.Connection.ID != c.ID {
			return c, errors.New("Private setup is missing. Replace the connection in Advanced.")
		}
		return c, nil
	}
	e = communication.SaveLaunch(deps.DataDir, communication.LaunchConfig{Connection: c, Token: token, URL: loopbackURL(deps) + communication.Path, Adapter: adapter, CABundle: ca})
	if e != nil {
		_ = deps.Store.RevokePeer(c.ID)
		return c, errors.New("Could not prepare the connection. Try again.")
	}
	return c, nil
}

// A bounded pass shares the attention worker's feed wakeups and local-input tick.
// No model turn is sent by setup and stopped participants are never auto-started.
func reconcilePeerParticipants(ctx context.Context, deps Deps) {
	participants, e := deps.Store.ListPeerParticipants()
	if e != nil {
		return
	}
	// Resolve only pre-minted, owner-attributed Pi session IDs. This is the
	// same adoption used by the conversation view, without requiring a visit.
	for _, p := range participants {
		if p.Enabled && p.Kind == "agent" && deps.Runtime.Active(p.OwnerID) {
			if a, err := deps.Store.GetAgent(p.OwnerID); err == nil && (a.SessionPath == nil || *a.SessionPath == "") {
				deps.Store.ResolvePendingAgentSession(p.OwnerID)
			}
		}
	}
	owners, e := deps.Store.ListPeerOwners()
	if e != nil {
		return
	}
	for _, p := range participants {
		if !p.Enabled {
			continue
		}
		for _, o := range owners {
			if o.Kind != p.Kind || o.OwnerID != p.OwnerID || o.CLI != p.CLI || o.WorkspaceID != p.WorkspaceID {
				continue
			}
			phase, problem, connection := "waiting-conversation", "", ""
			if o.SessionKey != "" {
				c, e := prepareParticipantConnection(deps, p, o)
				if e != nil {
					phase, problem = "error", e.Error()
				} else {
					connection = c.ID
					call, cancel := context.WithTimeout(ctx, 12*time.Second)
					phase, problem = applyPeerParticipant(call, deps, p, c)
					cancel()
				}
			}
			_, _ = deps.Store.SetPeerPreparation(p, o.SessionKey, phase, problem, connection)
		}
	}
}

func peerProcessConfigured(rt TermRuntime, connection string) bool {
	raw, e := os.ReadFile(fmt.Sprintf("/proc/%d/environ", rt.PID))
	if e != nil {
		return false
	}
	for _, v := range strings.Split(string(raw), "\x00") {
		if v == "PICODE_PEER_CONNECTION="+connection {
			return true
		}
	}
	return false
}

// Apply prepares the exact resume before stopping; user drafts and live turns
// keep the operation pending. A new conversation is never replaced by old argv.
func applyPeerParticipant(ctx context.Context, deps Deps, p store.PeerParticipant, c store.PeerConnection) (string, string) {
	if !peerConsentCurrent(deps, p, c.SessionKey) {
		return "cancelled", "Participation changed."
	}
	if c.Kind == "agent" {
		return applyManagedPeer(ctx, deps, p, c)
	}
	if deps.Tmux == nil || deps.TermRuntimes == nil || deps.TermStates == nil {
		return "error", "Terminal control is unavailable."
	}
	unlock := terminalLock(deps, c.OwnerID)
	defer unlock()
	t, e := deps.Store.GetTerminal(c.OwnerID)
	if e != nil {
		return "error", "This terminal is unavailable."
	}
	rt, exists := deps.TermRuntimes.Get(c.OwnerID)
	if blocked, e := peerStopPending(deps, c.OwnerID); e != nil || blocked {
		return "error", "The previous process is still closing. Reopen after it exits."
	}
	if !exists || !processAlive(rt) {
		return "stopped", "Open this conversation to connect."
	}
	if rt.SessionID != c.SessionKey {
		return "waiting-conversation", "Waiting for this conversation to identify itself."
	}
	if communication.NativeMessages(c.CLI) || (c.CLI != "pi" && peerProcessConfigured(rt, c.ID)) {
		return "connected", ""
	}
	if c.CLI == "pi" {
		if deps.Replies != nil && deps.Replies.receiverConnection(termReplyKey(c.OwnerID), rt.SessionPath, rt.RunID) == c.ID {
			return "connected", ""
		}
		if _, ok := peerLiveTerminal(deps, c); !ok {
			return "waiting", "Waiting for the current turn or approval."
		}
		if e := configurePeerPi(ctx, deps, termReplyKey(c.OwnerID), rt.SessionPath, c.ID); e == nil {
			return "connected", ""
		}

	}
	if !tryLockPrompt(c.OwnerID) {
		return "waiting", "Waiting for the current action to finish."
	}
	defer unlockPrompt(c.OwnerID)
	live, ok := peerLiveTerminal(deps, c)
	if !ok || live.RunID != rt.RunID {
		return "waiting", "Waiting for the current turn or approval."
	}
	name := tmux.ShellSessionName(c.OwnerID)
	before, e := deps.Tmux.InputSnapshot(ctx, name)
	if e != nil || !peerInputMatches(c.CLI, before, "") {
		return "waiting", "Finish or clear the draft in this conversation."
	}
	launch, e := deps.Store.TerminalLaunch(c.OwnerID)
	if e != nil || launch == nil || launch.LastSession == nil || launch.LastSession.SessionID != c.SessionKey {
		return "error", "The saved conversation changed."
	}
	resume := *launch
	args := terminalResumeArgs(launch.LastSession)
	resume.Overrides.Args = &args
	prepared, e := prepareCLITerminal(deps, t.Cwd, &resume)
	if e != nil {
		return "error", e.Error()
	}
	defer prepared.discard()
	current, ok := peerLiveTerminal(deps, c)
	snap, e := deps.Tmux.InputSnapshot(ctx, name)
	if !ok || current.RunID != rt.RunID || current.PID != rt.PID || e != nil || snap.PaneID != before.PaneID || snap.PanePID != before.PanePID || !peerInputMatches(c.CLI, snap, "") || !peerConsentCurrent(deps, p, c.SessionKey) {
		return "waiting", "The conversation changed; waiting to reconnect safely."
	}

	if e = stopPeerPane(ctx, deps, name, c.OwnerID, before.PanePID, rt); e != nil {
		return "error", e.Error()
	}
	deps.TermRuntimes.Drop(c.OwnerID)
	deps.TermStates.Drop(c.OwnerID)
	if !peerConsentCurrent(deps, p, c.SessionKey) {
		return "cancelled", "Participation changed."
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost/", nil)
	if e = prepared.startSized(deps, req, name, t.Cwd, snap.Width, len(snap.Lines)); e != nil {
		return "error", e.Error()
	}
	publishTerminalState(deps, req, t, true)
	return "connecting", ""
}

func applyManagedPeer(ctx context.Context, deps Deps, p store.PeerParticipant, c store.PeerConnection) (string, string) {
	if deps.Runtime == nil {
		return "error", "Agent control is unavailable."
	}
	ma := deps.Runtime.Get(c.OwnerID)
	if ma == nil {
		return "stopped", "Open this agent to connect."
	}
	if deps.Replies != nil && deps.Replies.receiverConnection(c.OwnerID, c.SessionKey, fmt.Sprint(ma.PID())) == c.ID {
		return "connected", ""
	}
	state := ma.Snapshot()
	if state.Streaming || state.Waiting {
		return "waiting", "Waiting for the current turn or approval."
	}
	if e := configurePeerPi(ctx, deps, c.OwnerID, c.SessionKey, c.ID); e != nil {
		return "error", e.Error()
	}
	if !peerConsentCurrent(deps, p, c.SessionKey) {
		return "cancelled", "Participation changed."
	}
	return "connected", ""

}

func reconcilePeerChecks(ctx context.Context, deps Deps) {
	reconcilePeerChecksAt(ctx, deps, time.Now())
}

func reconcilePeerChecksAt(ctx context.Context, deps Deps, now time.Time) {
	checks, e := deps.Store.ListPeerChecks()
	if e != nil {
		return
	}
	for _, c := range checks {
		if c.Phase != "pending" && c.Phase != "attempted" && c.Phase != "running" {
			continue
		}
		from, e1 := deps.Store.PeerConnection(c.SenderID)
		_, e2 := deps.Store.PeerConnection(c.RecipientID)
		if e1 != nil || e2 != nil {
			_, _ = deps.Store.SetPeerCheckPhase(c.ID, c.Phase, "cancelled")
			continue
		}
		if deps.Store.PeerCheckPassed(c) {
			_, _ = deps.Store.SetPeerCheckPhase(c.ID, c.Phase, "passed")
			continue
		}
		started, _ := time.Parse(time.RFC3339Nano, c.CreatedAt)
		if now.Sub(started) > 5*time.Minute {
			_, _ = deps.Store.SetPeerCheckPhase(c.ID, c.Phase, "expired")
			continue
		}
		if c.Phase != "pending" {
			continue
		}
		prompt := "PiCode: read messages and run the requested connection test."
		if communication.NativeMessages(from.CLI) {
			prompt = "PiCode: run picode messages read for the connection test."
		}
		call, cancel := context.WithTimeout(ctx, 3*time.Second)
		attemptPeerText(call, deps, from, prompt, func() bool {
			ok, e := deps.Store.SetPeerCheckPhase(c.ID, "pending", "attempted")
			return e == nil && ok
		}, func(e error) {
			phase := "running"
			if e != nil {
				phase = "uncertain"
			}
			_, _ = deps.Store.SetPeerCheckPhase(c.ID, "attempted", phase)
		})
		cancel()
	}
}

func configurePeerPi(ctx context.Context, deps Deps, key, session, connection string) error {
	if deps.Replies == nil || !deps.Replies.receiverFresh(key) || deps.Replies.receiverSession(key) != session {
		return errors.New("Reopen this Pi conversation to load its connection.")
	}
	nonce, e := newReplyNonce()
	if e != nil {
		return e
	}
	ack, done := deps.Replies.registerAck(nonce)
	defer done()
	file, e := writeReplyFile(deps.DataDir, key, replyFile{SetupConnection: connection, AttentionOnly: true, Nonce: nonce, SessionPath: session, CreatedAt: time.Now().UTC(), PID: deps.Replies.receiverPID(key)})
	if e != nil {
		return e
	}
	defer os.Remove(file)
	select {
	case <-ctx.Done():
		return errors.New("Waiting for the Pi connection adapter. Check Packages.")
	case a := <-ack:
		if !a.OK {
			return errors.New(a.Reason)
		}
		return nil
	}
}

func currentPeerOwner(deps Deps, kind, id string) (store.PeerOwner, error) {
	owners, e := deps.Store.ListPeerOwners()
	if e != nil {
		return store.PeerOwner{}, e
	}
	for _, o := range owners {
		if o.Kind == kind && o.OwnerID == id {
			return o, nil
		}
	}
	return store.PeerOwner{}, store.ErrNotFound
}
