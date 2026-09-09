package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/communication"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func registerPeerCommunication(mux Registrar, deps Deps) {
	// The MCP handler has mandatory scoped authentication even with auth off.
	mux.Handle(communication.Path, communication.Handler(deps.Store))
	mux.HandleFunc("GET /api/communication", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		owners, err := deps.Store.ListPeerOwners()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		peers, err := deps.Store.ListPeerConnections()
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		launches := map[string]bool{}
		live := map[string]string{}
		for _, p := range peers {
			launches[p.ID] = p.Active && communication.HasLaunch(deps.DataDir, p.ID)
			if !p.Active {
				continue
			}
			if p.Kind == "agent" && deps.Runtime != nil {
				if ma := deps.Runtime.Get(p.OwnerID); ma != nil {
					live[p.ID] = "open"
					state := ma.Snapshot()
					if state.Streaming {
						live[p.ID] = "working"
					}
					if state.Waiting {
						live[p.ID] = "needs-you"
					}
				}
			} else if rt, ok := deps.TermRuntimes.Get(p.OwnerID); ok && rt.SessionID == p.SessionKey && processAlive(rt) {
				live[p.ID] = "open"
				if deps.TermStates != nil {
					if state, ok := deps.TermStates.Get(p.OwnerID); ok && state.RunID == rt.RunID && state.SessionID == rt.SessionID && state.SessionSeq == rt.SessionSeq {
						live[p.ID] = state.State
					}
				}
			}
		}
		writeJSON(w, 200, map[string]any{"owners": owners, "connections": peers, "endpoint": communication.Path, "launches": launches, "live": live, "launchCLIs": []string{"pi", "claude-code", "codex", "opencode", "grok", "hermes"}})
	})
	mux.HandleFunc("POST /api/communication", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		var in struct {
			Kind       string `json:"kind"`
			OwnerID    string `json:"ownerId"`
			SessionKey string `json:"sessionKey"`
			Automatic  bool   `json:"automatic"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			writeErr(w, 400, "invalid connection request")
			return
		}
		adapter := ""
		caBundle := ""
		if in.Automatic {
			owners, err := deps.Store.ListPeerOwners()
			if err != nil {
				writeErr(w, 500, "Could not read conversations.")
				return
			}
			cli := ""
			for _, o := range owners {
				if o.Kind == in.Kind && o.OwnerID == in.OwnerID {
					cli = o.CLI
				}
			}
			if !communication.LaunchSupported(cli) {
				writeErr(w, 400, "Automatic setup is not available for this CLI. Use the connection guide.")
				return
			}
			if cli == "pi" {
				adapter = filepath.Join(pipkg.UserDir(), "npm", "node_modules", "pi-mcp-adapter", "index.ts")
				if _, err := os.Stat(adapter); err != nil {
					writeJSON(w, 400, map[string]any{"error": "Install the Pi MCP adapter before connecting.", "code": "adapter_missing"})
					return
				}
			}
		}
		if in.Automatic {
			extras := []string{os.Getenv("NODE_EXTRA_CA_CERTS"), os.Getenv("CODEX_CA_CERTIFICATE"), os.Getenv("SSL_CERT_FILE")}
			if in.Kind == "terminal" {
				launch, e := deps.Store.TerminalLaunch(in.OwnerID)
				if e != nil {
					writeErr(w, 500, "Could not read launch configuration.")
					return
				}
				if launch != nil {
					base, e := cliConfig(deps, launch.CLI)
					if e != nil {
						writeErr(w, 500, "Could not read CLI configuration.")
						return
					}
					cfg := clilaunch.Resolve(base, launch.Overrides)
					for _, key := range []string{"NODE_EXTRA_CA_CERTS", "CODEX_CA_CERTIFICATE", "SSL_CERT_FILE"} {
						extras = append(extras, cfg.Env[key])
					}
				}
			}
			var err error
			caBundle, err = communication.LocalTrust(deps.DataDir, loopbackURL(deps)+communication.Path, extras)
			if err != nil {
				writeErr(w, 400, err.Error())
				return
			}
		}
		p, secret, err := deps.Store.EnablePeer(in.Kind, in.OwnerID, in.SessionKey)
		if err != nil {
			status := statusForStore(err)
			if errors.Is(err, store.ErrPeerDenied) {
				status = 409
			}
			writeErr(w, status, err.Error())
			return
		}
		if in.Automatic {
			err = communication.SaveLaunch(deps.DataDir, communication.LaunchConfig{Connection: p, Token: secret, URL: loopbackURL(deps) + communication.Path, Adapter: adapter, CABundle: caBundle})
			if err != nil {
				_ = deps.Store.RevokePeer(p.ID)
				writeErr(w, 500, "Could not save private launch setup. Replace the connection to retry.")
				return
			}
			// Retired private setups must not make native session lookup ambiguous.
			previous, cleanupErr := deps.Store.ListPeerConnections()
			if cleanupErr == nil {
				for _, old := range previous {
					if old.ID != p.ID && old.Kind == p.Kind && old.OwnerID == p.OwnerID && !old.Active {
						if err := communication.RemoveLaunch(deps.DataDir, old.ID); err != nil {
							cleanupErr = err
						}
					}
				}
			}
			if cleanupErr != nil {
				_ = deps.Store.RevokePeer(p.ID)
				writeErr(w, 500, "Could not retire previous private setup. Replace the connection to retry.")
				return
			}
			writeJSON(w, 201, map[string]any{"connection": p, "automatic": true})
			return
		}
		writeJSON(w, 201, map[string]any{"connection": p, "token": secret, "endpoint": communication.Path})
	})
	mux.HandleFunc("DELETE /api/communication/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.Store.RevokePeer(r.PathValue("id")); err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		if err := communication.RemoveLaunch(deps.DataDir, r.PathValue("id")); err != nil {
			writeErr(w, 500, "Connection disabled, but private setup cleanup failed.")
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /api/communication/{id}/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		before := int64(0)
		var err error
		if raw := r.URL.Query().Get("before"); raw != "" {
			before, err = strconv.ParseInt(raw, 10, 64)
			if err != nil || before < 0 {
				writeErr(w, 400, "invalid cursor")
				return
			}
		}
		messages, err := deps.Store.PeerHistory(r.PathValue("id"), before)
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		writeJSON(w, 200, map[string]any{"messages": messages})
	})
}

func (deps Deps) startAgentTUI(ctx context.Context, name, cwd string, agent store.Agent) error {
	flags := deps.spawnFlags(agent)
	peerOptions, err := communication.AgentOptions(deps.Store, deps.DataDir, agent.ID)
	if err != nil {
		return err
	}
	flags = append(flags, peerOptions.Args...)
	env := agent.SpawnEnv()
	for k, v := range peerOptions.Env {
		env = append(env, k+"="+v)
	}
	return deps.Tmux.NewSessionEnv(ctx, name, cwd, env, deps.AgentCmd, flags...)
}
