package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/cfpperche/picode/internal/store"
)

const (
	maxPromptImages = 4
	maxImageB64     = (4*1024*1024*4)/3 + 16
)

var okImageMIME = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

type promptImage struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

func handleAgentDrop(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.runMode(r, id) != modeInteractive {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  "Attach files on an agent is for the in-terminal session.",
				"reason": "stopped",
			})
			return
		}
		var req dropBody
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxDropBytes*2)).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		raw, err := decodeDropData(req.Data)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cwd, err := liveAgentCwd(deps, r, agent)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.TrimSpace(cwd) == "" {
			writeErr(w, http.StatusBadRequest, "can't write in this folder")
			return
		}
		out, err := writeDropFile(cwd, req.Name, raw)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAgentPrompt(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var req struct {
			Kind     string        `json:"kind"`
			Message  string        `json:"message"`
			Images   []promptImage `json:"images"`
			Paths    []string      `json:"paths"`
			Delivery string        `json:"delivery"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if !validDelivery(req.Delivery) {
			writeErr(w, http.StatusBadRequest, "delivery must be prompt, steer or follow_up")
			return
		}
		mode := deps.runMode(r, id)
		if len(req.Paths) > 0 && mode == modeManaged {
			writeErr(w, http.StatusBadRequest, "paths is for the in-terminal session; managed chat uses images")
			return
		}
		if len(req.Images) > 0 && mode == modeInteractive {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":  "Images cannot reach a terminal session. Attach a file instead.",
				"reason": "images",
			})
			return
		}
		if err := checkPromptImages(req.Images); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		switch mode {
		case modeManaged:
			if strings.TrimSpace(req.Message) == "" && len(req.Images) == 0 {
				writeErr(w, http.StatusBadRequest, "message or image is required")
				return
			}
			ma := deps.Runtime.Get(id)
			if ma == nil {
				writeErr(w, http.StatusConflict, "agent is not running")
				return
			}
			imgs := make([]map[string]any, 0, len(req.Images))
			for _, im := range req.Images {
				imgs = append(imgs, map[string]any{
					"type":     "image",
					"data":     im.Data,
					"mimeType": im.MimeType,
				})
			}
			if err := ma.SendTurn(req.Kind, req.Message, imgs); err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		case modeInteractive:
			cwd, err := liveAgentCwd(deps, r, agent)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			rels, err := checkPromptPaths(cwd, req.Paths)
			if err != nil {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			if strings.TrimSpace(req.Message) == "" && len(rels) == 0 {
				writeErr(w, http.StatusBadRequest, "message or file is required")
				return
			}
			payload := buildPromptPaste(req.Message, rels)
			status, body := deps.deliverToInteractiveAgentAs(r.Context(), agent, payload, tuiDeliverPrompt, req.Delivery)
			if status != http.StatusOK {
				writeJSON(w, status, body)
				return
			}
			if deps.Feed != nil {
				deps.Feed.Ephemeral("agent.prompt", map[string]any{"agentId": id, "typed": true})
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "typed": true})
		default:
			writeErr(w, http.StatusConflict, "agent is not running")
		}
	}
}

func checkPromptImages(imgs []promptImage) error {
	if len(imgs) > maxPromptImages {
		return fmt.Errorf("at most 4 images")
	}
	for _, im := range imgs {
		if !okImageMIME[im.MimeType] {
			return fmt.Errorf("unsupported image type")
		}
		if im.Data == "" {
			return fmt.Errorf("image data is required")
		}
		if len(im.Data) > maxImageB64 {
			return fmt.Errorf("each image must be under 4 MB")
		}
	}
	return nil
}
