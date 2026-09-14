# File preview: HTML (ADR-0136)

> Part of PiCode's architecture (ADR-0105: one file per subsystem). Edit here; the index only links.

The file pane has one kind registry (`web/shared/domain/filePreview.js`):
`svg`, `mermaid`, `markdown` and `html` (text) plus `image`, `pdf`, `audio`,
`video` and `model3d` (blob). Every other kind renders in the app as a
component; HTML is a page, so it gets an origin instead.

## Route family

| Route | Answers |
|---|---|
| `POST /api/previews {kind, id, path, root}` | 200 `{url, path, expiresAt}` — an ordinary authed API (cookie/bearer, origin check); `.html`/`.htm` under the owner's resolved folder only, `root` precondition honored (ADR-0074) |
| `GET`/`HEAD` `/preview/{token}/{path…}` | the document or one allowlisted asset; everything else 404 "This preview is not available." (method-less mux pattern, so a POST answers 405 here instead of falling through to the app shell) |

## Policy

- **Ticket** (`internal/preview`): 32 random bytes, in-memory, one hour,
  bound to the minting session (re-checked against `auth_sessions` on every
  request — revoking a device kills its previews) and to the canonical root
  at mint time. The token is a path prefix so relative assets inherit it, and
  it is a credential: never logged, never fed, `no-referrer` on every answer.
- **Paths**: `relUnderCwd` then `filepath.EvalSymlinks`; anything resolving
  outside the root, any dot-prefixed segment, and any non-regular file is
  refused. A directory serves its own `index.html`, never a listing.
- **MIME**: the closed list in `preview.MIMEType` (web assets, fonts, media,
  wasm, pdf, 3d). `.env`, keys and extension-less files are structurally
  unservable.
- **Headers**: `CSP: sandbox allow-scripts allow-forms allow-modals
  allow-popups allow-popups-to-escape-sandbox allow-pointer-lock
  allow-downloads; frame-ancestors 'self'`, `Referrer-Policy: no-referrer`,
  `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, a
  Permissions-Policy that denies camera/microphone/geolocation/usb, and
  `Access-Control-Allow-Origin: *` + `Cross-Origin-Resource-Policy:
  cross-origin` so ES modules and `fetch` work from the opaque origin.
- **Pane**: one `usePreviewTicket` per app (browser and mobile own their
  own, ADR-0072) mints on open and Reload, HEAD-preflights the new URL, and
  pauses while the editor is dirty — the pane says "Unsaved changes aren't in
  the preview" instead of showing the previous version. The `iframe` mirrors
  the sandbox flags, `allow="fullscreen; clipboard-write"`.

## Why not `srcdoc`/`blob:`

Measured 2026-09-14 (Chromium): both inherit the embedding document's CSP, so
the page's inline scripts die under the app's `script-src 'self'`, and
relative URLs do not resolve into the project. A served response carries its
own policy — which is why the ticket route exists.

## Live reload

v1 is manual: **Reload** re-mints and re-loads, and opening the preview or
saving the file mints fresh. v1.5 (`docs/plans/html-preview.md`) records the
assets each ticket served, stats them from one ticker and reloads the parent
frame over SSE — no script injection into the page. A real second origin with
localStorage and workers is the v2 boundary move.
