# ADR-0147: user-installed webapps

- **Status**: accepted
- **Date**: 2026-09-17
- **Boundary**: persistence (first user-persisted app-like entity, new `webapps` table), protocol (new `/api/webapps` namespace + feed events), security model (third-party web content launched from a persistent shortcut inside the shell), process (the Tauri shell spawns the webview that serves it).

## Context

The Apps grid (ADR-0036) is first-party only: manifests are compiled Go values, the header has no add action, and free-form third-party surfaces are refused in v1. The user asks for the opposite door: type a URL, get a tile, click it, and the webapp opens inside the desktop shell.

What exists already shapes the answer. The Tauri shell (ADR-0120) renders any http(s) URL as a native WebView2 child — the work browser (`btab`), which shares one persistent profile under `%LOCALAPPDATA%\picode-shell\`, so logins survive restarts. Work-browser tabs are session-scoped and unnamed. Meanwhile an iframe in the SPA is closed twice over: the daemon CSP allows `frame-src 'self'` and loopback only (`internal/server/csp.go`), and most SaaS pages send `X-Frame-Options`/`frame-ancestors` that forbid framing. ADR-0036's marketplace-era amendment fixes the sandboxed iframe as the third-party *app* body — a different product from a user's shortcut to a site they already use.

Precedents studied: Chrome/Edge "Install as app" (manifest supplies name/icons; refuses an unreachable site), Rambox/Ferdium custom services (URL alone is enough; per-service scripts are the integration era, not v1), Cursor's browser pane (persistent cookies/localStorage per surface is the expected bar).

Owner decisions (2026-09-17): installation **refuses** when the site does not answer; v1 ships a **title-derived badge** (`(N)` prefix); plain-browser clicks show the honest "requires desktop" message only (no system-browser handoff); loopback webapps are first-class citizens.

## Decision

Installed webapps are a **persisted shortcut entity + launcher**, not a new app kind. One store table (`webapps`: id, name, url UNIQUE, icon, icon_mime, created_at), mutations announce `webapp.installed/updated/removed` in the same transaction (ADR-0048), served under a new `/api/webapps` namespace (list, resolve, install, rename, delete, icon). `POST /api/webapps/resolve` and the install itself fetch the page server-side — PWA manifest first (same-origin only: `name`, then an icon resolved against the manifest URL), then same-origin `<link rel=icon>`, then `/favicon.ico`; the site must answer or the install is refused. The row stores **the URL the user typed**, never the redirect-landing URL, so a login redirect cannot hijack the shortcut. The fetch is bounded (timeout, body caps, redirect cap, no proxy, no credentials) and DNS-rebind-guarded: a public hostname resolving to a private address is refused at dial, and redirects may not cross from public to private — explicit loopback URLs stay first-class.

In the UI, `AppsGrid` gains the add action; installed webapps render as tiles beside the first-party ones (icon endpoint → letter-tile fallback) with Open/Rename/Remove. Clicking focuses or opens a work-browser tab with the stable id `w:app-<webappId>`; the shell's existing `btab_meta` title poll drives the `(N)` badge — no new polling. In a plain browser the click announces that webapps require PiCode Desktop and does nothing else. The webapp page itself gains no ADR-0109 door: no `host` object, no tokens, no approval UI, no badge API.

## Consequences

- The Apps grid stops being "drawn entirely from manifests": it now mixes compile-time first-party tiles with store-backed user tiles. `GET /api/apps` stays untouched; the UI merges the two lists.
- Third-party content runs inside the shell under a shortcut the user chose. It is engine-isolated (per-origin cookies), CDP deny-by-default (ADR-0128), and gets no IPC bridge — but it shares the work profile, so "Clear browsing data" wipes webapp logins together with work-browser sites. Accepted and to be documented.
- If wrong: the entity is one table and one namespace — removable without touching the first-party app contract; the launcher degrades to the existing unnamed work browser.
- URL-with-fragment rows mean `example.com` and `example.com/#/inbox` are distinct shortcuts; duplicates are judged on the normalized address.

## Alternatives considered

- **iframe in the SPA** — refused: daemon CSP, `X-Frame-Options` on most SaaS, and ADR-0036's v1 refusal; the sandboxed-iframe amendment is the marketplace-era *app* surface, not a user shortcut.
- **Register installed webapps as first-party `App`s in `GET /api/apps`** — refused: that contract is compile-time, `apiVersion`-gated, first-party; user rows would corrupt its semantics.
- **Separate Tauri window per webapp (`WebviewWindow`)** — deferred: escapes the shell chrome (ADR-0122), loses btab bounds/preview/find plumbing; revisit for PWAs declaring `display: standalone`.
- **`chrome --app=URL`** — refused: depends on an installed Chrome, sits on the deprecated extension path (ADR-0120), and leaves the shell.
- **Per-webapp WebView2 partitions** — deferred to v2 (two accounts of one service); v1 shares the work profile.

## Amendment 2026-09-18 — the manifest's own identity, not just its name

Dogfooding (GitHub installed live, opened in the shell with its login)
confirmed the shortcut model; the owner clarified the product intent was
PWA-first. The manifest is now read beyond name/icon: `start_url`, `scope`,
`display` and `theme_color` persist with the shortcut (migration 054,
same-origin resolved against the manifest URL, anything malformed dropped).
Launch rule: a **bare** typed address defers to the manifest's `start_url`
(the app's own entry point); a deep link the user typed wins — explicit
intent beats the app default. The add dialog announces a detected web app.
The standalone-window door stays exactly where it was — deferred, owner's
call — so a PWA today installs with its full identity and launches at its
own start page inside the work browser; a Chrome-style app window is the
next boundary decision, not this one.

## Amendment 2026-09-18 — app mode inside the shell (the standalone question, settled)

Asked to choose between a Chrome-style standalone window (v2's deferred
door) and staying in the shell, the owner chose the shell: an installed
web app whose manifest declares an app-like `display` (`standalone`,
`fullscreen`, `minimal-ui`) opens **without the browser toolbar** — a
minimal titlebar (the app's own name left; ⋮ menu right, carrying Reload,
Copy address and the usual browser actions) and the page filling the
surface. `display: browser` (and plain shortcuts) keep the full toolbar.
The separate-window alternative stays refused for now: the user works
inside PiCode, not across windows; revisiting it costs an ADR that
re-opens the btab plumbing question. Login persistence, the shared
profile and the CDP policy are unchanged.
