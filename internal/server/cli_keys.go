// One door for every CLI's key map (docs/plans/keyboard-pane.md, P1).
//
// This endpoint answers the screen for any CLI the registry knows, pi included
// (its own `/api/pi-keys` route was retired 2026-09-25): pi's report is
// wrapped, flat-file CLIs are read and written here, and the rest answer with
// their state so the pane can say an honest sentence instead of "in development".
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cfpperche/picode/internal/clikeys"
	"github.com/cfpperche/picode/internal/clisettings"
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
	CLI      string   `json:"cli"`
	Label    string   `json:"label"`
	State    string   `json:"state"`
	Keymap   string   `json:"keymap"`
	Writable bool     `json:"writable"`
	Pickup   string   `json:"pickup"`
	Vocab    string   `json:"vocab"`
	Contexts []string `json:"contexts,omitempty"`
	File     string   `json:"file,omitempty"`
	Exists   bool     `json:"exists"`
	Platform string   `json:"platform,omitempty"`
	// Revision is the file's revision as this report read it. A client sends it
	// back with a write, which is how a file that changed underneath is refused
	// (409) instead of overwritten. Empty for a CLI that keeps its own store.
	Revision string `json:"revision,omitempty"`
	// Unreadable names rows the file sets in a shape PiCode does not rewrite.
	Unreadable []string            `json:"unreadable,omitempty"`
	Actions    []clikeys.Action    `json:"actions"`
	User       map[string][]string `json:"user"`
}

func cliKeysEnvelope(cli *clikeys.CLI) (cliKeys, error) {
	out := cliKeys{
		CLI: cli.ID, Label: cli.Label, State: string(cli.State),
		Keymap: string(cli.Keymap), Pickup: string(cli.Pickup), Vocab: cli.Vocab,
		Contexts: cli.Contexts,
		Actions:  []clikeys.Action{},
		User:     map[string][]string{},
	}
	if cli.State != clikeys.Shipped {
		return out, nil
	}
	out.Writable = true
	if cli.OwnStore {
		// Pi keeps its own store, and this envelope wraps its report rather than
		// restating it (ADR-0101): pikeys owns the catalog and the write path.
		rep, err := piKeysSnapshot()
		if err != nil {
			return cliKeys{}, err
		}
		out.File, out.Exists, out.Platform = rep.File, rep.Exists, rep.Platform
		out.Actions, out.User = asEnvelopeRows(rep.Actions), rep.User
		return out, nil
	}
	// Every other shipped CLI is one flat file the settings package's document
	// primitives read and write (ADR-0174).
	flat, ok := clikeys.FlatFor(cli.ID)
	if !ok {
		return cliKeys{}, fmt.Errorf("%s ships an editor PiCode cannot mount", cli.Label)
	}
	m, err := clikeys.ReadFlat(flat)
	if err != nil {
		return cliKeys{}, err
	}
	out.File, out.Exists, out.Revision, out.Platform = m.File, m.Exists, m.Revision, flat.Platform
	out.Unreadable, out.Actions, out.User = m.Unreadable, flat.Catalog, m.Values
	return out, nil
}

// asEnvelopeRows converts pi's catalog to the envelope's row type. The two are
// separate on purpose: pikeys owns pi's catalog, and this envelope is the
// pane's vocabulary (ADR-0174).
func asEnvelopeRows(in []pikeys.Action) []clikeys.Action {
	out := make([]clikeys.Action, 0, len(in))
	for _, a := range in {
		out = append(out, clikeys.Action{ID: a.ID, Group: a.Group, Label: a.Label, Defaults: a.Defaults, Alt: a.Alt})
	}
	return out
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

// putPiKeys is pi's write path, unchanged: its store owns the map and the
// validation of what a key may be.
func putPiKeys(req cliKeysPut) error {
	switch {
	case req.ResetAll:
		_, err := pikeys.ResetKnown()
		return err
	case req.Reset:
		return pikeys.Set(req.Action, nil)
	case req.Keys == nil:
		return errors.New("keys, reset or resetAll")
	default:
		return pikeys.Set(req.Action, *req.Keys)
	}
}

// putFlatKeys writes one row of a guest CLI's own file through the flat engine.
func putFlatKeys(id string, req cliKeysPut) error {
	flat, ok := clikeys.FlatFor(id)
	if !ok {
		return fmt.Errorf("PiCode has no key-map writer for %s", id)
	}
	switch {
	case req.ResetAll:
		return clikeys.ResetFlat(flat, req.Revision)
	case req.Reset:
		return clikeys.WriteFlat(flat, req.Action, nil, true, req.Revision)
	case req.Keys == nil:
		return errors.New("keys, reset or resetAll")
	default:
		return clikeys.WriteFlat(flat, req.Action, *req.Keys, false, req.Revision)
	}
}

type cliKeysPut struct {
	CLI      string    `json:"cli"`
	Action   string    `json:"action"`
	Keys     *[]string `json:"keys"`
	Reset    bool      `json:"reset"`
	ResetAll bool      `json:"resetAll"`
	// Revision is what the editor read. A file that moved since is refused
	// rather than overwritten (the guarantee clisettings' writes already make).
	Revision string `json:"revision"`
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
	var err error
	if cli.OwnStore {
		err = putPiKeys(req)
	} else {
		err = putFlatKeys(cli.ID, req)
	}
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, clisettings.ErrStale) {
			status = http.StatusConflict
		}
		writeErr(w, status, err.Error())
		return
	}
	rep, err := cliKeysEnvelope(cli)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
