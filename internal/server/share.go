package server

import (
	"net/http"
	"net/url"

	"github.com/cfpperche/picode/internal/share"
)

func handleShare(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		share.SyncCert(deps.DataDir)
		port := 0
		if deps.PortSnapshot != nil {
			port = deps.PortSnapshot().Current
		}
		rep := share.Diagnose(share.Input{
			Insecure: deps.Insecure,
			BindHost: deps.BindHost,
			Port:     port,
			DataDir:  deps.DataDir,
		})
		if tp := share.EnsureTrustHTTP(); tp != "" && rep.URL != "" {
			if u, err := url.Parse(rep.URL); err == nil {
				base := share.TrustURL(u.Hostname(), tp)
				rep.TrustURL = base + "?next=" + url.QueryEscape(rep.URL)
			}
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
