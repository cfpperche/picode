package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/browserhost"
	"github.com/cfpperche/picode/internal/store"
)

func registerExtensionRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/extension/agents", handleExtensionAgents(deps))
	mux.HandleFunc("POST /api/extension/send", handleExtensionSend(deps))
	registerActRoutes(mux, deps)
}

type extensionAgent struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Workspace string `json:"workspace,omitempty"`
	Mode      string `json:"mode"`
}

type extensionTab struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Selection string `json:"selection"`
}

type extensionImage struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

func handleExtensionAgents(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		out := make([]extensionAgent, 0)
		ws, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, wk := range ws {
			agents, err := deps.Store.ListAgents(wk.ID)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			for _, a := range agents {
				out = append(out, extensionAgent{
					ID: a.ID, Name: a.Name, Workspace: wk.Name,
					Mode: string(deps.runMode(r, a.ID)),
				})
			}
		}
		free, err := deps.Store.ListAgents(store.FreeWorkspaceID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		for _, a := range free {
			out = append(out, extensionAgent{
				ID: a.ID, Name: a.Name,
				Mode: string(deps.runMode(r, a.ID)),
			})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Name == out[j].Name {
				return out[i].ID < out[j].ID
			}
			return out[i].Name < out[j].Name
		})
		writeJSON(w, http.StatusOK, map[string]any{"agents": out})
	}
}

func handleExtensionSend(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AgentID string          `json:"agentId"`
			Message string          `json:"message"`
			Tab     *extensionTab   `json:"tab"`
			Image   *extensionImage `json:"image"`
			Act     bool            `json:"act"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if strings.TrimSpace(req.AgentID) == "" {
			writeErr(w, http.StatusBadRequest, "agentId is required")
			return
		}

		dec := decideExtensionSend(extensionSendInput{
			HasMessage: strings.TrimSpace(req.Message) != "",
			HasImage:   req.Image != nil && req.Image.Data != "",
			TabURL:     tabURL(req.Tab),
		})
		if dec.Status != 0 {
			writeErr(w, dec.Status, dec.Error)
			return
		}

		agent, err := deps.Store.GetAgent(req.AgentID)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		if !agent.IsPi() {
			// A guest agent (ADR-0160) has no managed mode: its terminal is its
			// process, so the tab reaches it through the unattended prompt door
			// (ADR-0179). Screenshots and the act loop ride Pi's RPC turn.
			sendToGuestAgent(w, r, deps, agent, req.Tab, req.Message, req.Image != nil && req.Image.Data != "", req.Act)
			return
		}

		mode := deps.runMode(r, agent.ID)
		dec = decideExtensionSend(extensionSendInput{
			Mode:       mode,
			HasMessage: strings.TrimSpace(req.Message) != "",
			HasImage:   req.Image != nil && req.Image.Data != "",
			TabURL:     tabURL(req.Tab),
		})
		if dec.Status != 0 {
			writeErr(w, dec.Status, dec.Error)
			return
		}

		started := false
		if dec.Start {
			if err := deps.startManaged(agent); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			started = true
		}

		ma := deps.Runtime.Get(agent.ID)
		if ma == nil {
			writeErr(w, http.StatusConflict, "agent is not running")
			return
		}

		var imgs []promptImage
		if req.Image != nil && req.Image.Data != "" {
			imgs = []promptImage{{MimeType: req.Image.MimeType, Data: req.Image.Data}}
			if imgs[0].MimeType == "" {
				imgs[0].MimeType = "image/jpeg"
			}
			if err := checkPromptImages(imgs); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		body := composeTabPrompt(req.Tab, req.Message)
		if req.Act {
			if tabURL(req.Tab) == "" {
				writeErr(w, http.StatusBadRequest, "acting needs a page")
				return
			}
			body += "\n\n" + browserhost.ActIntro
		}
		rpcImgs := make([]map[string]any, 0, len(imgs))
		for _, im := range imgs {
			rpcImgs = append(rpcImgs, map[string]any{
				"type": "image", "data": im.Data, "mimeType": im.MimeType,
			})
		}
		if err := ma.SendTurn(store.TaskPrompt, body, rpcImgs); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		watching := false
		if req.Act {
			_ = deps.Store.ExpirePendingActBatches(agent.ID)
			if err := startActWatch(deps, agent.ID, originOf(tabURL(req.Tab))); err != nil {
				// The turn was delivered; only the act loop is refused.
				writeJSON(w, http.StatusOK, map[string]any{"ok": true, "started": started, "watching": false, "actError": err.Error()})
				return
			}
			watching = true
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "started": started, "watching": watching})
	}
}

// sendToGuestAgent delivers a tab to a non-Pi agent through its launch
// terminal. The page link and the message are text the CLI reads at its
// prompt; nothing is started — a closed terminal is refused with a reason.
func sendToGuestAgent(w http.ResponseWriter, r *http.Request, deps Deps, agent store.Agent, tab *extensionTab, message string, hasImage, act bool) {
	switch {
	case hasImage:
		writeErr(w, http.StatusBadRequest, "Screenshots go to Pi agents. Send the page without one.")
		return
	case act:
		writeErr(w, http.StatusBadRequest, "Acting on a page needs a Pi agent.")
		return
	case agent.TerminalID == nil || strings.TrimSpace(*agent.TerminalID) == "":
		writeErr(w, http.StatusConflict, "This agent has no terminal yet. Open it once in PiCode.")
		return
	}
	t, err := deps.Store.GetTerminal(*agent.TerminalID)
	if err != nil {
		writeErr(w, http.StatusConflict, "This agent's terminal is gone. Open the agent in PiCode.")
		return
	}
	status, res := doorDeliverUnattended(deps, r.Context(), t, composeTabPrompt(tab, message))
	if status != http.StatusOK {
		msg, _ := res["error"].(string)
		if msg == "" {
			msg = "The tab could not be sent."
		}
		if reason, _ := res["reason"].(string); reason == "closed" {
			msg = "This agent's terminal is closed. Open the agent in PiCode, then send again."
		}
		writeErr(w, status, msg)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "started": false, "watching": false, "delivery": res["delivery"]})
}

// originOf reduces an absolute URL to scheme://host[:port].
func originOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func tabURL(tab *extensionTab) string {
	if tab == nil {
		return ""
	}
	return strings.TrimSpace(tab.URL)
}

type extensionSendInput struct {
	Mode       agentRunMode
	HasMessage bool
	HasImage   bool
	TabURL     string
}

type extensionDecision struct {
	Start  bool
	Status int
	Error  string
}

// decideExtensionSend is the Track A decision table (ADR-0043).
func decideExtensionSend(in extensionSendInput) extensionDecision {
	if !in.HasMessage && !in.HasImage && in.TabURL == "" {
		return extensionDecision{Status: http.StatusBadRequest, Error: "message or page is required"}
	}
	if in.TabURL != "" && !capturableURL(in.TabURL) {
		return extensionDecision{Status: http.StatusBadRequest, Error: "This page can't be sent."}
	}
	switch in.Mode {
	case "":
		return extensionDecision{}
	case modeInteractive:
		return extensionDecision{Status: http.StatusConflict, Error: "This agent is in the terminal."}
	case modeStopped:
		return extensionDecision{Start: true}
	default:
		return extensionDecision{}
	}
}

func capturableURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func composeTabPrompt(tab *extensionTab, message string) string {
	var b strings.Builder
	if tab != nil && strings.TrimSpace(tab.URL) != "" {
		b.WriteString("[browser-tab]\n")
		b.WriteString("url: ")
		b.WriteString(strings.TrimSpace(tab.URL))
		b.WriteByte('\n')
		if t := strings.TrimSpace(tab.Title); t != "" {
			b.WriteString("title: ")
			b.WriteString(t)
			b.WriteByte('\n')
		}
		if sel := strings.TrimSpace(tab.Selection); sel != "" {
			b.WriteString("selection:\n")
			b.WriteString(sel)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString(strings.TrimSpace(message))
	return strings.TrimSpace(b.String())
}

// startManaged brings the agent up in managed mode without touching a live
// TUI session. The extension refuses interactive agents before calling this.
func (deps Deps) startManaged(agent store.Agent) error {
	unlock := terminalLock(deps, "agent:"+agent.ID)
	defer unlock()
	if deps.Runtime.Get(agent.ID) != nil {
		return nil
	}
	if _, err := exec.LookPath(deps.AgentCmd); err != nil {
		return fmt.Errorf("pi is not installed or not on PATH")
	}
	wk, cwd, err := deps.agentHome(agent)
	if err != nil {
		return err
	}
	if err := deps.startAgentRPC(context.Background(), agent.ID, cwd); err != nil {
		return fmt.Errorf("start managed: %w", err)
	}
	_ = deps.Store.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "managed")
	_ = deps.Store.AppendEvent("agent_managed_started", &agent.ID, &wk.ID, nil)
	return nil
}
