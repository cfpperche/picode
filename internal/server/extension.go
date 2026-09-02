package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

func registerExtensionRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/extension/agents", handleExtensionAgents(deps))
	mux.HandleFunc("POST /api/extension/send", handleExtensionSend(deps))
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
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "started": started})
	}
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
	if err := deps.Runtime.Start(agent.ID, cwd); err != nil {
		return fmt.Errorf("start managed: %w", err)
	}
	_ = deps.Store.SetAgentRuntimeMode(agent.ID, store.StatusRunning, "managed")
	_ = deps.Store.AppendEvent("agent_managed_started", &agent.ID, &wk.ID, nil)
	return nil
}
