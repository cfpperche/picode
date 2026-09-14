# ADR-0136: html-file-preview

- **Status**: accepted (owner, 2026-09-14 — "aprovado pode executar", the five
  recommendations in `docs/plans/html-preview.md`)
- **Date**: 2026-09-14
- **Boundary**: security model — project HTML gains an executable rendering
  surface of its own: a cookie-less capability route on the app origin, an
  opaque-origin sandbox, and the exact reach a previewed page has.

## Context

The file pane previews SVG, mermaid, markdown, images, PDF, audio, video and
3D models (ADR-0019's "no MIME zoo" was overtaken by the pane), but an HTML
file opens in the raw editor. Agents write `.html` constantly — reports,
dashboards, demos — and the browser is the only renderer that means anything
for them.

An HTML preview runs project code inside the user's authenticated session.
The obvious door — an iframe on the app origin with `allow-same-origin` —
would hand the page the user's PiCode cookie, the API and the whole fleet.
MDN warns about `allow-scripts` + `allow-same-origin` for same-origin
embeds, and ADR-0036 already records the stance: a sandboxed iframe is the
industry's only trusted escape hatch, and it demands origin and CSP
discipline.

Two facts were measured first (Chromium via `agent_browser`, 2026-09-14;
probe script disposable, conclusions kept):

- A same-origin `sandbox="allow-scripts"` frame carries **no cookie at all**
  (`Sec-Fetch-Site: cross-site`), sends `Origin: null`, cannot read a
  same-origin JSON answer without CORS, and cannot navigate the top page —
  and therefore cannot authenticate to a cookie-gated route.
- `srcdoc` and `blob:` iframes **inherit the embedding document's CSP**, so
  a previewed page's inline scripts die under the app's `script-src 'self'`;
  a served HTTP response carries its own policy instead.

## Decision

HTML files preview through a **capability route**. An authenticated
`POST /api/previews` mints a short-lived, session-bound ticket for one
`.html`/`.htm` file; the pane embeds `GET /preview/<token>/<path>` in a frame
sandboxed by **response header** (CSP `sandbox`) *and* by the `sandbox`
attribute — never `allow-same-origin`, never top-navigation. The token sits
in the path prefix so relative assets inherit it; the route takes no cookie
(the sandbox sends none) and serves only allowlisted web asset types under
the owner's resolved directory, with dotfile and symlink containment,
`Referrer-Policy: no-referrer`, `Cache-Control: no-store`, and
`Access-Control-Allow-Origin: *` on assets so ES modules and `fetch` work
from the opaque origin. Isolation rests on the browser: reads are
CORS-blocked, writes are already refused by the `Origin: null` rule in
`internal/auth`, and the top page cannot be moved.

Fidelity flags kept: scripts, forms, modals, popups
(`allow-popups-to-escape-sandbox`, so a link opens a normal tab), pointer
lock, downloads, fullscreen and clipboard-write on the element. Refused:
`allow-same-origin` (the entire boundary) and `allow-top-navigation*`.

## Consequences

- **Easier**: agents that write HTML ship something the user can see; the
  file pane stays the one preview surface; no new dependency, listener or
  certificate.
- **Harder**: a previewed page is an opaque origin — no localStorage,
  sessionStorage, cookies, workers or service workers, and a page that needs
  them fails visibly instead of silently. Unsaved editor text is not in the
  preview (checked in the pane, not pretended); console capture is out of
  reach; a heavier page's DevTools path is the token URL in a real tab.
- **If wrong** (a browser sandbox escape): a project HTML file could read
  PiCode API answers or act as the user. The blast radius is a browser bug,
  not a PiCode shortcut; a second origin (v2) remains the next boundary
  move.
- Accepted residual: the page can read *allowlisted* sibling assets
  (`js`, `json`, `csv`…) and post them out. `.env`, key extensions and
  extension-less files are structurally out of the allowlist; secrets with
  web extensions inside the project are not defended against.

## Alternatives considered

| Alternative | Why it lost |
|---|---|
| Same-origin iframe without sandbox | Full API access as the user; MDN's own warning on the same-origin pair |
| `srcdoc` / `blob:` | Inherit the app CSP (inline scripts blocked, measured) and do not resolve relative URLs into the project |
| Separate listener or host in v1 | A second HTTPS surface that must reach through LAN, Tailscale, gateway and phone; its own decision as v2 |
| Server-side render (Go HTML → image) | No scripts, no interaction — a different product |
