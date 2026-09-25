package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/skillcatalog"
)

// The Skills Marketplace (ADR-0196 slice 5): one catalog for every CLI. A
// card is a description; Install opens the Skills tab's own preview, scan and
// consent. The person's sources and the skills.sh switch are settings, whose
// writes already announce setting.updated.
const (
	skillSourcesKey  = "skills.sources"
	skillsSHKey      = "skills.skillssh"
	skillsSHDeadline = 8 * time.Second
)

// newSkillCatalog builds the catalog; tests swap it for one that reads a fake
// GitHub, so no test reads the network or writes after its folder is gone.
var newSkillCatalog = skillcatalog.NewStore

func registerSkillCatalogRoutes(mux Registrar, deps Deps) {
	cat := newSkillCatalog(deps.DataDir)
	cat.OnRead = func(input string) {
		if deps.Feed != nil {
			deps.Feed.Ephemeral("skills.catalog", map[string]any{"source": input})
		}
	}
	mux.HandleFunc("GET /api/skills/catalog", handleSkillCatalog(deps, cat))
	mux.HandleFunc("POST /api/skills/sources", handleSkillSourceAdd(deps, cat))
	mux.HandleFunc("DELETE /api/skills/sources", handleSkillSourceRemove(deps, cat))
	mux.HandleFunc("PUT /api/skills/skillssh", handleSkillsSHSwitch(deps))
}

func skillSources(deps Deps) []string {
	if deps.Store == nil {
		return nil
	}
	raw, ok, err := deps.Store.GetSetting(skillSourcesKey)
	if err != nil || !ok {
		return nil
	}
	var out []string
	if json.Unmarshal([]byte(raw), &out) != nil {
		return nil
	}
	return out
}

func skillsSHOn(deps Deps) bool {
	if deps.Store == nil {
		return false
	}
	v, ok, err := deps.Store.GetSetting(skillsSHKey)
	return err == nil && ok && v == "1"
}

func handleSkillCatalog(deps Deps, cat *skillcatalog.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		items, sources := cat.Search(q, skillSources(deps))
		out := map[string]any{"items": items, "sources": sources, "skillssh": skillsSHOn(deps)}
		// skills.sh only when the person switched it on, and only for a query:
		// the query text goes to Vercel, and the switch says so.
		if skillsSHOn(deps) && len(q) >= 2 {
			ctx, cancel := context.WithTimeout(r.Context(), skillsSHDeadline)
			hits, err := cat.SearchSkillsSH(ctx, q)
			cancel()
			if err != nil {
				out["skillsshError"] = err.Error()
			} else {
				seen := map[string]bool{}
				for _, it := range items {
					seen[it.Source+"|"+it.Name] = true
				}
				for _, h := range hits {
					if !seen[h.Source+"|"+h.Name] {
						items = append(items, h)
					}
				}
				out["items"] = items
			}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func saveSkillSources(deps Deps, list []string) error {
	if list == nil {
		list = []string{}
	}
	raw, _ := json.Marshal(list)
	return deps.Store.SetSetting(skillSourcesKey, string(raw))
}

func handleSkillSourceAdd(deps Deps, cat *skillcatalog.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || deps.Store == nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		home, _ := os.UserHomeDir()
		input, err := skillcatalog.ValidateSource(req.Input, home)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		list := skillSources(deps)
		for _, s := range append(append([]string{}, skillcatalog.Seeds...), list...) {
			if strings.EqualFold(s, input) {
				writeErr(w, http.StatusConflict, input+" is already a source")
				return
			}
		}
		if len(list) >= 50 {
			writeErr(w, http.StatusBadRequest, "at most 50 sources of your own")
			return
		}
		if err := saveSkillSources(deps, append(list, input)); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"input": input})
	}
}

func handleSkillSourceRemove(deps Deps, cat *skillcatalog.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || deps.Store == nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		for _, s := range skillcatalog.Seeds {
			if s == req.Input {
				writeErr(w, http.StatusBadRequest, "a built-in source stays; your own sources can be removed")
				return
			}
		}
		list := skillSources(deps)
		next := make([]string, 0, len(list))
		for _, s := range list {
			if s != req.Input {
				next = append(next, s)
			}
		}
		if len(next) == len(list) {
			writeErr(w, http.StatusNotFound, "not one of your sources")
			return
		}
		if err := saveSkillSources(deps, next); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cat.Forget(req.Input)
		writeJSON(w, http.StatusOK, map[string]string{"removed": req.Input})
	}
}

func handleSkillsSHSwitch(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			On bool `json:"on"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || deps.Store == nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		v := ""
		if req.On {
			v = "1"
		}
		if err := deps.Store.SetSetting(skillsSHKey, v); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"on": req.On})
	}
}
