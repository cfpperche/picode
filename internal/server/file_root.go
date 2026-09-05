package server

import "net/http"

// A tree's root is an equality precondition on the independently resolved
// owner cwd, never an alternative source of authority (ADR-0073). Use the
// same resolved cwd for the check and the subsequent read/write, so a terminal
// cd cannot redirect a relative filename between these operations.
func checkFileRoot(w http.ResponseWriter, r *http.Request, cwd string) bool {
	if expected := r.URL.Query().Get("root"); expected != "" && expected != canonDir(cwd) {
		writeErr(w, http.StatusConflict, "This folder changed. Refresh the file tree.")
		return false
	}
	return true
}
