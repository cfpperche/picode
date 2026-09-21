package server

import (
	"encoding/json"
	"net/http"
	"os"
	"runtime"

	"github.com/cfpperche/picode/internal/pikeys"
)

func registerPiKeysRoutes(mux Registrar) {
	mux.HandleFunc("GET /api/pi-keys", handleGetPiKeys)
	mux.HandleFunc("PUT /api/pi-keys", handlePutPiKeys)
}

// piKeysReport is what the Keyboard pane renders. File is this machine's real
// path, so the pane can name what it writes, and Platform picks the default pi
// actually binds here: nine actions carry a Windows or WSL alternate that the
// default column alone would misreport (pikeys.Catalog's Alt).
type piKeysReport struct {
	Actions  []pikeys.Action     `json:"actions"`
	User     map[string][]string `json:"user"`
	File     string              `json:"file"`
	Exists   bool                `json:"exists"`
	Platform string              `json:"platform"`
}

// piKeysPlatform names the host the way the catalog's Alt map does: "wsl"
// before GOOS, because a WSL distro is linux with pi's WSL row.
func piKeysPlatform() string {
	if isWSL() {
		return "wsl"
	}
	return runtime.GOOS
}

func piKeysSnapshot() (piKeysReport, error) {
	user, err := pikeys.LoadUser()
	if err != nil {
		return piKeysReport{}, err
	}
	path := pikeys.File()
	exists := false
	if path != "" {
		_, statErr := os.Stat(path)
		exists = statErr == nil
	}
	return piKeysReport{
		Actions:  pikeys.Catalog,
		User:     user,
		File:     path,
		Exists:   exists,
		Platform: piKeysPlatform(),
	}, nil
}

func handleGetPiKeys(w http.ResponseWriter, _ *http.Request) {
	rep, err := piKeysSnapshot()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

type piKeysPut struct {
	Action   string    `json:"action"`
	Keys     *[]string `json:"keys"`
	Reset    bool      `json:"reset"`
	ResetAll bool      `json:"resetAll"`
}

func handlePutPiKeys(w http.ResponseWriter, r *http.Request) {
	var req piKeysPut
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ResetAll {
		// Every override this catalog knows, in one write; a key the file
		// carries and PiCode does not stays (pikeys.ResetKnown).
		if _, err := pikeys.ResetKnown(); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	} else {
		var keys []string
		if req.Reset {
			keys = nil
		} else if req.Keys == nil {
			writeErr(w, http.StatusBadRequest, "keys or reset")
			return
		} else {
			keys = *req.Keys
		}
		if err := pikeys.Set(req.Action, keys); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	rep, err := piKeysSnapshot()
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
