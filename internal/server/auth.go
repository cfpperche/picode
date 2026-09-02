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
	"github.com/cfpperche/picode/internal/presence"
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
	mux.HandleFunc("GET /pair", handlePairPage(deps))
	mux.HandleFunc("POST /pair", handlePair(deps))
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
	rep := share.Diagnose(share.Input{Insecure: deps.Insecure, BindHost: deps.BindHost, Port: port, DataDir: deps.DataDir, PublicURL: deps.publicURL()})
	if rep.URL == "" {
		link = deps.Auth.PairURL(r, code)
		return link, link
	}
	link = auth.PairURLFrom(rep.URL, code)
	qr = link
	if rep.Trusted {
		return link, qr // a public chain: no trust page in the way
	}
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

const askNewLink = "Ask for a new one on a paired device: Devices → Pair a device."

func appHome(r *http.Request) string {
	if strings.Contains(strings.ToLower(r.UserAgent()), "mobile") {
		return "/?mobile=1"
	}
	return "/"
}

// handlePairPage is the link a QR or a message carries. A GET must not
// spend the code: phones and chat apps prefetch links before the person
// opens them (an iPhone camera scan did exactly that). So the page asks
// for one tap, and the POST below does the pairing. A browser that is
// already paired goes straight in.
func handlePairPage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(auth.CookieName); err == nil && c.Value != "" {
			if _, err := deps.Store.LookupSession(c.Value); err == nil {
				http.Redirect(w, r, appHome(r), http.StatusSeeOther)
				return
			}
		}
		code := strings.TrimSpace(r.URL.Query().Get("code"))
		if code == "" {
			pairPage(w, http.StatusBadRequest, pairView{Heading: "This link is not valid", Body: "Ask for a new one on a paired device: Devices → Pair a device."})
			return
		}
		device := strings.TrimSpace(r.URL.Query().Get("device"))
		who := presence.Label(r.UserAgent())
		form := `<form method="post" action="/pair">` +
			`<input type="hidden" name="code" value="` + html.EscapeString(code) + `">` +
			`<input type="hidden" name="device" value="` + html.EscapeString(device) + `">` +
			`<button type="submit" class="pair-btn">Pair this ` + html.EscapeString(who) + `</button></form>`
		pairPage(w, http.StatusOK, pairView{
			Heading: "Pair this " + who,
			Body:    "This link connects this browser to PiCode on <strong>" + html.EscapeString(r.Host) + "</strong>. It only works once.",
			Extra:   form,
		})
	}
}

// handlePair spends the code (POST from the page above), sets the cookie
// and lands in the app. Failures are a plain page — a person, not a
// script, is on the other side.
func handlePair(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			pairPage(w, http.StatusBadRequest, pairView{Heading: "This link is not valid", Body: askNewLink})
			return
		}
		code := strings.TrimSpace(r.Form.Get("code"))
		device := strings.TrimSpace(r.Form.Get("device"))
		_, err := deps.Auth.Pair(w, r, code, device)
		switch {
		case err == nil:
			http.Redirect(w, r, appHome(r), http.StatusSeeOther)
		case auth.IsTooMany(err):
			pairPage(w, http.StatusTooManyRequests, pairView{Heading: "Too many attempts", Body: "Wait ten minutes, then open a fresh link."})
		case errors.Is(err, store.ErrPairingUsed):
			pairPage(w, http.StatusGone, pairView{Heading: "This link was already used", Body: askNewLink})
		case errors.Is(err, store.ErrPairingExpired):
			pairPage(w, http.StatusGone, pairView{Heading: "This link expired", Body: askNewLink})
		default:
			pairPage(w, http.StatusBadRequest, pairView{Heading: "This link is not valid", Body: askNewLink})
		}
	}
}

// pairView is one branded page: a heading, a line of body copy (may
// contain simple inline HTML the caller has already escaped where it
// echoes user input) and an optional extra block (the confirm form).
type pairView struct {
	Heading string
	Body    string
	Extra   string
}

// pairPage renders the /pair surface with PiCode's own look — dark-first,
// the same tokens and π mark as the app shell — instead of an unstyled
// system page. It is server-rendered (no React here: this loads before
// any cookie exists), so the palette is inlined rather than shared with
// app.css.
func pairPage(w http.ResponseWriter, code int, v pairView) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", PageCSP)
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, pairPageTemplate, html.EscapeString(v.Heading)+" · PiCode", html.EscapeString(v.Heading), v.Body, v.Extra)
}

const pairPageTemplate = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="theme-color" content="#0e0e11">
<title>%s</title>
<style>
:root {
  --bg-base: #0e0e11; --bg-panel: #16161c; --border: #232329; --border-strong: #2c2c34;
  --text-primary: #ececf1; --text-secondary: #9b9ba7; --accent: #7c8cf8;
  --sans: -apple-system, "Segoe UI", Inter, Roboto, sans-serif;
  --serif: Georgia, ui-serif, serif;
  color-scheme: dark;
}
@media (prefers-color-scheme: light) {
  :root {
    --bg-base: #ffffff; --bg-panel: #f7f8fa; --border: #dfe3ea; --border-strong: #c9cedb;
    --text-primary: #16181d; --text-secondary: #5b6472; --accent: #2f6fed;
    color-scheme: light;
  }
}
* { box-sizing: border-box; }
body {
  margin: 0; min-height: 100dvh; display: flex; align-items: center; justify-content: center;
  padding: 24px; background: var(--bg-base); color: var(--text-primary);
  font: 15px/1.55 var(--sans);
}
.pair-card {
  width: 100%%; max-width: 380px; background: var(--bg-panel); border: 1px solid var(--border);
  border-radius: 14px; padding: 28px 26px 26px;
}
.pair-brand { display: flex; align-items: center; gap: 10px; margin-bottom: 22px; }
.pair-mark {
  font: 700 26px/1 var(--serif); color: var(--accent); width: 32px; text-align: center;
}
.pair-name { font-weight: 650; letter-spacing: .2px; font-size: 15px; }
.pair-heading { font-size: 18px; font-weight: 650; margin: 0 0 10px; line-height: 1.3; }
.pair-body { margin: 0; color: var(--text-secondary); font-size: 13.5px; }
.pair-body strong { color: var(--text-primary); font-weight: 600; }
.pair-btn {
  font: inherit; font-weight: 600; font-size: 14px; width: 100%%; margin-top: 20px;
  height: 44px; border-radius: 9px; border: 0; cursor: pointer;
  background: var(--accent); color: #fff; transition: filter 120ms ease-out;
}
.pair-btn:hover { filter: brightness(1.08); }
.pair-btn:active { filter: brightness(0.96); }
.pair-foot { margin: 18px 0 0; font-size: 11.5px; color: var(--text-secondary); text-align: center; }
</style>
</head>
<body>
  <main class="pair-card" role="main">
    <div class="pair-brand">
      <span class="pair-mark" aria-hidden="true">π</span>
      <span class="pair-name">PiCode</span>
    </div>
    <h1 class="pair-heading">%s</h1>
    <p class="pair-body">%s</p>
    %s
    <p class="pair-foot">The browser is a door, not a cage.</p>
  </main>
</body>
</html>
`
