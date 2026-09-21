package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/store"
)

// appsGridOrderKey is the saved Apps-tab grid order. Values are tile ids
// ("inbox", a web app id). Absent means alphabetical. The catalog is code,
// not rows, so the order lives in settings and announces setting.updated.
const appsGridOrderKey = "apps.grid.order"

func handleAppsOrderGet(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		ids, err := readAppsOrder(deps.Store)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ids": ids})
	}
}

func handleAppsOrderPut(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Store == nil {
			writeErr(w, http.StatusServiceUnavailable, "store is not open")
			return
		}
		var body struct {
			IDs []string `json:"ids"`
		}
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&body); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		seen := map[string]bool{}
		for _, id := range body.IDs {
			if id == "" || seen[id] {
				writeErr(w, http.StatusBadRequest, "ids must not repeat")
				return
			}
			seen[id] = true
		}
		raw, err := json.Marshal(body.IDs)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.Store.SetSetting(appsGridOrderKey, string(raw)); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func readAppsOrder(st *store.Store) ([]string, error) {
	raw, ok, err := st.GetSetting(appsGridOrderKey)
	if err != nil || !ok || raw == "" {
		return []string{}, err
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return []string{}, nil
	}
	if ids == nil {
		ids = []string{}
	}
	return ids, nil
}
