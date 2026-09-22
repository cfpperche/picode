package server

// Terminal browser hand-off (ADR-0180). A coding CLI's "open in the browser"
// moment — /login OAuth pages above all — used to resolve inside WSL, where
// xdg-open finds an X11 chromium instead of the browser the person is
// looking at. Managed terminals get a `picode-open` wrapper (plus `xdg-open`
// and `wslview` shadows, and BROWSER pointing at it) in the ADR-0056
// intercept bin dir; the wrapper POSTs the URL here, and the daemon routes
// it to a visible client — the desktop app opens its integrated tab, a
// plain browser opens itself — or, when nobody is watching, to the
// Windows default browser through osopen.

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/osopen"
)

// OpenURLID is the wiring row of the browser hand-off. Like the tmux guard
// (ADR-0138) it defaults on and has no UI toggle; an operator opting out
// writes enabled.json "open-url": false and removes the wrappers.
const OpenURLID = "open-url"

// openURLWrappers are the PATH-visible names. picode-open is the BROWSER
// target; the other two shadow what CLIs call directly.
var openURLWrappers = []string{"picode-open", "xdg-open", "wslview"}

// openURLWrapper is one POSIX sh body for every name: forward an http(s)
// argument to the daemon, passthrough everything else. The resolver avoids
// external binaries for the same reason as the tmux guard (${0%/*}, measured
// 2026-09-15); outside a managed terminal PICODE_TERM_URL is absent and the
// script degrades to the real opener it shadows.
const openURLWrapper = `#!/bin/sh
# PiCode intercept — browser hand-off (ADR-0180). Session PATH only.
name=__NAME__
here=${0%/*}
[ "$here" = "$0" ] && here=.
real=
extra=
IFS=:
for d in $PATH; do
  [ "$d" = "$here" ] && continue
  if [ -z "$real" ] && [ -x "$d/$name" ]; then real=$d/$name; fi
  if [ -z "$extra" ] && [ "$name" != "xdg-open" ] && [ -x "$d/xdg-open" ]; then extra=$d/xdg-open; fi
done
unset IFS
[ -n "$real" ] || real=$extra

url=
for a in "$@"; do
  case "$url:$a" in
    :http://*| :https://*) url=$a ;;
  esac
done

if [ -n "$url" ] && [ -n "${PICODE_TERM_ID-}" ] && [ -n "${PICODE_TERM_URL-}" ] && command -v curl >/dev/null 2>&1; then
  post=$PICODE_TERM_URL
  case "$post" in
    https://127.0.0.1:*) post="https://localhost:${post##*:}" ;;
  esac
  TOKEN=$(cat "__DATADIR__/token" 2>/dev/null)
  json=$(printf '%s' "$url" | sed 's/\\/\\\\/g; s/"/\\"/g')
  if curl -fsSk -o /dev/null --max-time 4 \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"url\":\"$json\"}" \
    "$post/api/terminals/$PICODE_TERM_ID/open-url" 2>/dev/null; then
    exit 0
  fi
fi

[ -n "$real" ] && exec "$real" "$@"
exit 0
`

func writeOpenURLWrapper(dataDir, name string) error {
	body := strings.NewReplacer("__NAME__", name, "__DATADIR__", dataDir).Replace(openURLWrapper)
	return writeExecutable(wrapperPath(dataDir, name), body)
}

func removeOpenURLWrappers(dataDir string) {
	for _, name := range openURLWrappers {
		_ = os.Remove(wrapperPath(dataDir, name))
	}
}

// installOpenURL/uninstallOpenURL are the wiring verbs for the row; the
// explicit-false write matters as much as guard's, since the default is on.
func installOpenURL(dataDir string) error {
	for _, name := range openURLWrappers {
		if err := writeOpenURLWrapper(dataDir, name); err != nil {
			return err
		}
	}
	m := loadInterceptEnabled(dataDir)
	if m[OpenURLID] {
		return nil
	}
	m[OpenURLID] = true
	return saveInterceptEnabled(dataDir, m)
}

func uninstallOpenURL(dataDir string) error {
	removeOpenURLWrappers(dataDir)
	m := loadInterceptEnabled(dataDir)
	m[OpenURLID] = false
	return saveInterceptEnabled(dataDir, m)
}

// ensureOpenURLWrappers (re)writes the hand-off wrappers when the wiring is
// on — the hook/reply scripts refresh the same way every boot, so a wrapper
// body can never fall behind the binary. Opt-out: enabled.json
// open-url=false removes the files instead.
func ensureOpenURLWrappers(dataDir string) {
	if strings.TrimSpace(dataDir) == "" {
		return
	}
	if !interceptOn(dataDir, OpenURLID) {
		removeOpenURLWrappers(dataDir)
		return
	}
	for _, name := range openURLWrappers {
		_ = writeOpenURLWrapper(dataDir, name)
	}
}

// openURLEnv is the BROWSER=… entry for session environments, empty when
// the hand-off is off. Only sessions PiCode creates see it.
func openURLEnv(dataDir string) []string {
	if !interceptOn(dataDir, OpenURLID) {
		return nil
	}
	path := wrapperPath(dataDir, "picode-open")
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return []string{"BROWSER=" + path}
}

// openURLFn is the host-side opener, swapped in tests (pattern: revealFn,
// mcp.go). CI has no browser.
var openURLFn = osopen.OpenURL

// handleTerminalOpenURL routes one terminal's browser-open request:
// POST /api/terminals/{id}/open-url {"url":"https://…"}.
// A visible client wins (the desktop app turns the event into an integrated
// tab; a plain browser opens a tab in itself); with no client subscribed the
// daemon opens the host's default browser. The event is ephemeral on the
// feed: a reconnect must not replay a stale login page.
func handleTerminalOpenURL(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		u, err := osopen.ValidOpenURL(req.URL)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if deps.Feed != nil && deps.Feed.Subscribers() > 0 {
			deps.Feed.Ephemeral("terminal.open_url", map[string]any{"termId": id, "url": u})
			writeJSON(w, http.StatusOK, map[string]any{"delivered": "client"})
			return
		}
		if err := openURLFn(u); err != nil {
			writeErr(w, http.StatusBadGateway, "the URL is valid but no client is listening and the host browser did not open: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"delivered": "host"})
	}
}
