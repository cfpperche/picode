package server

import (
	"net/http"

	"github.com/cfpperche/picode/internal/share"
)

func handleShare(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		share.SyncCert(deps.DataDir)
		port := 0
		if deps.PortSnapshot != nil {
			port = deps.PortSnapshot().Current
		}
		writeJSON(w, http.StatusOK, share.Diagnose(share.Input{
			Insecure: deps.Insecure,
			BindHost: deps.BindHost,
			Port:     port,
			DataDir:  deps.DataDir,
		}))
	}
}
