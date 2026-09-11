// User-described package configurations (ADR-0119; C5 of
// docs/plans/package-config-manifest.md). The owner describes a package's
// config file and fields once, through this API, and the package becomes
// GUI-configurable like any catalog entry. Descriptors live as one JSON
// file each under DataDir/package-configs/ — deleted when the description
// is cleared, at which point the package honestly loses its Configure
// button again.
package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/pipkg"
)

func registerPackageDescribeRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/packages/describe", handleGetPackageDescribe(deps))
	mux.HandleFunc("PUT /api/packages/describe", handlePutPackageDescribe(deps))
	mux.HandleFunc("DELETE /api/packages/describe", handleDeletePackageDescribe(deps))
}

// userConfigDir is where user descriptors persist, under the server's data
// dir. Empty DataDir means the feature is off (the endpoint refuses rather
// than pretending to save).
func (d Deps) userConfigDir() string {
	if d.DataDir == "" {
		return ""
	}
	return d.DataDir + "/package-configs"
}

// templateFor seeds the describe form with what can be guessed: the id and
// title come from the package name; everything else is the user's job.
func templateFor(pkg string) *pipkg.ConfigDescriptor {
	return &pipkg.ConfigDescriptor{
		ID:    pkg,
		Match: pkg,
		Title: pkg,
		Application: "Applies the next time the package reads its config. " +
			"Empty fields stay unset.",
		Files:  []pipkg.ConfigFile{{Scope: "agent", Path: pkg + ".json", Format: "json"}},
		Fields: []pipkg.ConfigField{},
	}
}

func handleGetPackageDescribe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pkg := r.URL.Query().Get("package")
		if pkg == "" {
			writeErr(w, http.StatusBadRequest, "package is required")
			return
		}
		if d := pipkg.UserDescriptorByID(pkg); d != nil {
			writeJSON(w, http.StatusOK, map[string]any{"package": pkg, "exists": true, "descriptor": d})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"package": pkg, "exists": false, "descriptor": templateFor(pkg)})
	}
}

func handlePutPackageDescribe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := deps.userConfigDir()
		if dir == "" {
			writeErr(w, http.StatusServiceUnavailable, "no data dir — user-described configs are unavailable")
			return
		}
		var d pipkg.ConfigDescriptor
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := pipkg.ValidateUserDescriptor(&d); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := pipkg.SaveUserDescriptor(dir, &d); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceConfigChange(deps, d.ID, d.Files[0].Scope)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "descriptor": d})
	}
}

func handleDeletePackageDescribe(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := deps.userConfigDir()
		if dir == "" {
			writeErr(w, http.StatusServiceUnavailable, "no data dir — user-described configs are unavailable")
			return
		}
		pkg := r.URL.Query().Get("package")
		if pkg == "" {
			writeErr(w, http.StatusBadRequest, "package is required")
			return
		}
		if err := pipkg.DeleteUserDescriptor(dir, pkg); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceConfigChange(deps, pkg, "agent")
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
