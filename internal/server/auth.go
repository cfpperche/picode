package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"net/url"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/share"
	"github.com/cfpperche/picode/internal/store"
)

// Auth routes (ADR-0049). The middleware in server.New already gates
// them; these only read or change principals.
func registerAuthRoutes(mux *http.ServeMux, deps Deps) {
	if deps.Auth == nil {
		return
	}
	mux.HandleFunc("GET /api/auth/session", handleAuthSession(deps))
	mux.HandleFunc("GET /api/auth/sessions", handleAuthSessions(deps))
	mux.HandleFunc("DELETE /api/auth/sessions/{id}", handleAuthRevoke(deps))
	mux.HandleFunc("POST /api/auth/pairings", handleAuthPairing(deps))
	mux.HandleFunc("POST /api/auth/logout", handleAuthLogout(deps))
	mux.HandleFunc("PUT /api/auth/mode", handleAuthMode(deps))
	mux.HandleFunc("POST /api/auth/token/rotate", handleAuthRotate(deps))
	mux.HandleFunc("GET /pair", handlePair(deps))
}

func handleAuthSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := auth.From(r)
		out := map[string]any{"mode": deps.Auth.Mode(), "loopback": auth.Loopback(r)}
		if p != nil {
			out["kind"] = p.Kind
			out["session"] = p.Session
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleAuthSessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListSessions()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cur := ""
		if p := auth.From(r); p != nil {
			cur = p.Session.ID
		}
		// Liveness comes from presence: a ping carries its session id, so
		// "online" here means "that paired device pinged within 45 s".
		live := map[string]presenceView{}
		if deps.Presence != nil {
			for _, d := range deps.Presence.List() {
				if d.Session == "" {
					continue
				}
				v := live[d.Session]
				if d.Online {
					v.Online = true
				}
				if d.LastSeen > v.LastSeen {
					v.LastSeen = d.LastSeen
				}
				if d.Kind != "" {
					v.Kind = d.Kind
				}
				live[d.Session] = v
			}
		}
		type row struct {
			store.Session
			Current  bool   `json:"current"`
			Online   bool   `json:"online"`
			PingKind string `json:"pingKind,omitempty"`
			PingSeen string `json:"pingSeen,omitempty"`
		}
		out := make([]row, 0, len(list))
		for _, s := range list {
			v := live[s.ID]
			out = append(out, row{Session: s, Current: s.ID == cur, Online: v.Online, PingKind: v.Kind, PingSeen: v.LastSeen})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out, "tokenPath": deps.Auth.TokenPath()})
	}
}

type presenceView struct {
	Online   bool
	LastSeen string
	Kind     string
}

func handleAuthRevoke(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := deps.Store.RevokeSession(r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "session not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleAuthPairing mints a one-time link for another device.
func handleAuthPairing(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		by := ""
		if p := auth.From(r); p != nil {
			by = p.Session.ID
		}
		code, exp, err := deps.Store.CreatePairing(by, auth.PairingTTL)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		link, qr := deps.pairLinks(r, code)
		writeJSON(w, http.StatusCreated, map[string]any{"code": code, "url": link, "qrUrl": qr, "expiresAt": exp})
	}
}

// pairLinks: the link another device can actually open, and what the QR
// should encode. A request on loopback carries "localhost" as its host —
// useless to a phone — so the share report's reachable address is used,
// and with an mkcert setup the QR goes through the trust page first, the
// same path "Open on phone" takes.
func (deps Deps) pairLinks(r *http.Request, code string) (link, qr string) {
	if pub := deps.Auth.PublicURL(); pub != "" {
		link = auth.PairURLFrom(pub, code)
		return link, link
	}
	if !auth.Loopback(r) {
		link = deps.Auth.PairURL(r, code)
		return link, link
	}
	port := 0
	if deps.PortSnapshot != nil {
		port = deps.PortSnapshot().Current
	}
	rep := share.Diagnose(share.Input{Insecure: deps.Insecure, BindHost: deps.BindHost, Port: port, DataDir: deps.DataDir})
	if rep.URL == "" {
		link = deps.Auth.PairURL(r, code)
		return link, link
	}
	link = auth.PairURLFrom(rep.URL, code)
	qr = link
	if tp := share.EnsureTrustHTTP(); tp != "" {
		if u, err := url.Parse(rep.URL); err == nil {
			qr = share.TrustURL(u.Hostname(), tp) + "?next=" + url.QueryEscape(link)
		}
	}
	return link, qr
}

func handleAuthLogout(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deps.Auth.Logout(w, r)
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleAuthMode(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := deps.Auth.SetMode(req.Mode); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"mode": req.Mode})
	}
}

func handleAuthRotate(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		tok, err := deps.Auth.RotateToken()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": tok, "tokenPath": deps.Auth.TokenPath()})
	}
}

// handlePair is the link a QR or a message carries: spend the code, set
// the cookie, land in the app. Failures are a plain page — a person, not
// a script, is on the other side.
func handlePair(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		device := strings.TrimSpace(r.URL.Query().Get("device"))
		_, err := deps.Auth.Pair(w, r, code, device)
		switch {
		case err == nil:
			next := "/"
			if strings.Contains(strings.ToLower(r.UserAgent()), "mobile") {
				next = "/?mobile=1"
			}
			http.Redirect(w, r, next, http.StatusSeeOther)
		case auth.IsTooMany(err):
			pairPage(w, http.StatusTooManyRequests, "Too many attempts. Wait ten minutes and open a fresh link.")
		case errors.Is(err, store.ErrPairingUsed):
			pairPage(w, http.StatusGone, "This link was already used. Ask for a new one from Preferences → Server on a paired device.")
		case errors.Is(err, store.ErrPairingExpired):
			pairPage(w, http.StatusGone, "This link expired. Ask for a new one from Preferences → Server on a paired device.")
		default:
			pairPage(w, http.StatusBadRequest, "This link is not valid.")
		}
	}
}

func pairPage(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, `<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><title>PiCode</title>
<body style="font:15px/1.5 system-ui,sans-serif;margin:40px auto;max-width:28em;padding:0 16px;color:#222"><h1 style="font-size:18px">Pairing</h1><p>%s</p></body>`, html.EscapeString(msg))
}
