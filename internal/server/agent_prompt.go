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

func handleAgentPrompt(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAgent(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var req struct {
			Kind    string        `json:"kind"`
			Message string        `json:"message"`
			Images  []promptImage `json:"images"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		if err := checkPromptImages(req.Images); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
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
