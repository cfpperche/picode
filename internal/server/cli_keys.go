// One door for every CLI's key map (docs/plans/keyboard-pane.md, P1).
//
// `/api/pi-keys` stays what it always was: pi's own map, its own store, its own
// contract. This endpoint answers the same screen for any CLI the registry
// knows — today only pi ships one, and the rest answer with their state so the
// pane can say an honest sentence instead of "in development".
package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/clikeys"
	"github.com/cfpperche/picode/internal/pikeys"
)

func registerCLIKeysRoutes(mux Registrar) {
	mux.HandleFunc("GET /api/cli-keys", handleGetCLIKeys)
	mux.HandleFunc("PUT /api/cli-keys", handlePutCLIKeys)
}

// cliKeys is the envelope the Keyboard pane renders, whoever the CLI is. A
// field a CLI cannot answer is absent rather than zero-valued: a planned CLI
// has no file to name and no actions to list.
type cliKeys struct {
	CLI      string              `json:"cli"`
	Label    string              `json:"label"`
	State    string              `json:"state"`
	Keymap   string              `json:"keymap"`
	Writable bool                `json:"writable"`
	Pickup   string              `json:"pickup"`
	Vocab    string              `json:"vocab"`
	Contexts []string            `json:"contexts,omitempty"`
	File     string              `json:"file,omitempty"`
	Exists   bool                `json:"exists"`
	Platform string              `json:"platform,omitempty"`
	Actions  []pikeys.Action     `json:"actions"`
	User     map[string][]string `json:"user"`
}

func cliKeysEnvelope(cli *clikeys.CLI) (cliKeys, error) {
	out := cliKeys{
		CLI: cli.ID, Label: cli.Label, State: string(cli.State),
		Keymap: string(cli.Keymap), Pickup: string(cli.Pickup), Vocab: cli.Vocab,
		Contexts: cli.Contexts,
		Actions:  []pikeys.Action{},
		User:     map[string][]string{},
	}
	if cli.State != clikeys.Shipped {
		return out, nil
	}
	// The only shipped editor is pi's, and it reads its own store: the envelope
	// wraps that report rather than restating it (pikeys owns the catalog and
	// the write path).
	rep, err := piKeysSnapshot()
	if err != nil {
		return cliKeys{}, err
	}
	out.Writable = true
	out.File, out.Exists, out.Platform = rep.File, rep.Exists, rep.Platform
	out.Actions, out.User = rep.Actions, rep.User
	return out, nil
}

func handleGetCLIKeys(w http.ResponseWriter, r *http.Request) {
	cli := clikeys.For(r.URL.Query().Get("cli"))
	if cli == nil {
		writeErr(w, http.StatusBadRequest, "PiCode has no keyboard editor for that CLI")
		return
	}
	rep, err := cliKeysEnvelope(cli)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

type cliKeysPut struct {
	CLI      string    `json:"cli"`
	Action   string    `json:"action"`
	Keys     *[]string `json:"keys"`
	Reset    bool      `json:"reset"`
	ResetAll bool      `json:"resetAll"`
}

func handlePutCLIKeys(w http.ResponseWriter, r *http.Request) {
	var req cliKeysPut
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	cli := clikeys.For(req.CLI)
	if cli == nil {
		writeErr(w, http.StatusBadRequest, "PiCode has no keyboard editor for that CLI")
		return
	}
	if cli.State != clikeys.Shipped {
		// Refused and planned are different facts with the same answer here:
		// nothing is written, and the message says which one it is.
		msg := "PiCode cannot write " + cli.Label + "'s key map yet"
		switch {
		case cli.State == clikeys.Refused:
			msg = cli.Label + " does not allow its keys to be remapped"
		case cli.Keymap == clikeys.Partial:
			msg = cli.Label + " keeps no key map file; the keys it does allow are in Settings"
		}
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	switch {
	case req.ResetAll:
		if _, err := pikeys.ResetKnown(); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	case req.Reset:
		if err := pikeys.Set(req.Action, nil); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	case req.Keys == nil:
		writeErr(w, http.StatusBadRequest, "keys, reset or resetAll")
		return
	default:
		if err := pikeys.Set(req.Action, *req.Keys); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	rep, err := cliKeysEnvelope(cli)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
