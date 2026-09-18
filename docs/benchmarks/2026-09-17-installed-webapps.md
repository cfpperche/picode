# Study: user-installed webapps (URL → tile → open in the desktop shell)

- **Date:** 2026-09-17
- **Scope:** the owner's proposal — an **Add** action on the Apps sidebar tab;
  the user types a webapp URL, gets a persistent tile, and clicking it opens
  the webapp inside the desktop shell. Shipped as ADR-0147 with this note as
  its benchmark base.
- **Sources:** Chrome "Install as app" / PWA install docs
  (developer.chrome.com), MDN Progressive web apps (manifest identity:
  `name`/`short_name`/`icons`), Ferdium recipes docs (custom service =
  URL + optional per-service script), Rambox "custom service" model,
  [2026-09-02 live browser preview](2026-09-02-live-browser-preview.md)
  (Cursor's per-surface persistent sessions), and PiCode's own work browser
  (btab) as the runtime.

## The comparison

| Product | Install unit | Identity source | Runtime | Sessions |
|---|---|---|---|---|
| **Chrome / Edge "Install as app"** | a URL (manifest optional) | web app manifest, else page title + favicon; **install refused when the site does not answer** | browser app window, own profile | cookies persist per app |
| **Rambox / Ferdium custom service** | a URL, no recipe needed | service name + icon (recipe only adds badge/dark-mode scripts) | Electron `<webview>` with per-service partitions | persist per service |
| **Workona / Arc pinned apps** | a named shortcut | page title + favicon | the browser's own tabs | browser profile |
| **Cursor browser pane** (2026-09-02 study) | — (agent surface) | — | pane over a managed browser | **persist per workspace** (cookies, localStorage) — the bar users expect |
| **iframe in the SPA** (the "obvious" v0) | a URL | — | iframe | blocked: daemon CSP (`frame-src` loopback only), `X-Frame-Options` on most SaaS, ADR-0036 refusal |

## What PiCode adopted

- **Chrome's install contract, minus the OS install**: the daemon resolves
  the URL server-side (bounded fetch, DNS-rebind guard, loopback
  first-class), identity from the PWA manifest first (same-origin only),
  then favicon; **an unreachable site refuses the install** (owner decision).
- **Rambox's "URL is enough"**: no per-service scripts in v1 — that is the
  marketplace-era surface ADR-0036 reserves, not a user shortcut.
- **Cursor's persistence bar, for free**: the shortcut opens the existing
  work-browser WebView2 on the stable tab `w:app-<id>`, sharing the shell's
  persistent profile — logins survive restarts with zero new runtime.
- **Ferdium's badge, cheaply**: the leading `(N)` page title becomes the
  tile badge — the same signal their recipes scrape, read from data
  `btab_meta` already reports (no new polling).

## What we refused (and why)

| Pattern | Why refused here |
|---|---|
| iframe embedding in the web UI | CSP + `X-Frame-Options` + ADR-0036; the sandboxed-iframe amendment is the marketplace-era app body, a different product |
| per-service WebView2 partitions (Ferdium model) | v2: two accounts of one service; v1 shares the work profile (documented: "Clear browsing data" clears webapp logins too) |
| separate Tauri window per webapp | escapes the shell chrome (ADR-0122), loses btab bounds/preview/find; revisit for `display: standalone` PWAs |
| registering webapps in `GET /api/apps` | that contract is compile-time first-party with `apiVersion`; user rows would corrupt it |

**Adaptation line for the record:** this surface adapts Chrome's install
flow and Rambox's custom-URL service to PiCode's work browser, citing the
persistence bar measured in the Cursor study.
