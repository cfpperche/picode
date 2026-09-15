# HTML file preview — study and spec

- **Status:** v1 and v1.5 implemented on 2026-09-14 (`feat/html-preview`,
  `feat/html-preview-live`, `feat/html-preview-unsaved`): ADR-0136 accepted
  with the eight decisions below as recommended; the route family and policy
  live in `docs/architecture/file-preview.md`. v1.5 is complete — live reload
  (served-asset watch + `__events` SSE), preview-only documents over the text
  cap, and unsaved editor text as the ticket's overlay. Still open: v2 (a
  real second origin), which needs its own ADR.
- **Boundary:** security model — a cookie-less, sandboxed origin for project
  HTML and a capability route (ADR-0136).
- **Scope:** render `.html` / `.htm` files in PiCode's existing file surfaces
  the way a browser shows them — scripts run, relative assets resolve —
  without giving the page any reach into PiCode, its API or its session.

## What exists today

The file pane already has a kind registry and an inline renderer per kind.
HTML is the one text format with no renderer, so it opens in the raw editor.

| Layer | File | Today |
|---|---|---|
| Kind map | `web/shared/domain/filePreview.js` | `svg`, `mermaid`, `markdown` (text) + `image`, `pdf`, `audio`, `video`, `model3d` (blob). No `.html` |
| Renderer | `web/browser/src/components/FilePreview.jsx`, `web/mobile/src/components/FilePreview.jsx` | one `FilePreview` per app; PDF already uses an `<iframe src=blob:>` |
| Pane | `web/browser/src/components/FilePane.jsx`, `web/mobile/src/components/FileDocument.jsx` | default mode `preview` when `previewKind(path)` is set, else `raw`/`edit`; Preview \| Raw chips |
| Chat card | `web/browser/src/components/FileCard.jsx` (+ mobile) | text kinds render inline; blob kinds render players |
| Owners | `/api/{agents,terminals,workspaces}/{id}/{text,blob}` (`internal/server/agent_files.go`, `terminals.go`, `workspace_files.go`) | `relUnderCwd` + `root` precondition (ADR-0074); blob MIME is a fixed allowlist |
| Gate | `internal/auth` | `guarded()` covers `/api/` and `/ws/` only; `internal/server/csp.go` sets the app CSP on `.html` UI responses |

## Research — the references, and what each one decides

| Source | What it establishes |
|---|---|
| [VS Code Live Preview](https://github.com/microsoft/vscode-livepreview) | Previewing a *project* needs a **served origin**, not a string: an embedded server, relative assets, live refresh (default), `data-server-no-reload` opt-out, external-browser preview, a console channel |
| [JetBrains built-in server](https://www.jetbrains.com/help/webstorm/editing-html-files.html) | Same shape since forever: a built-in HTTP server on port 63342, an in-IDE preview tab, auto-reload on save or on type |
| [MDN `<iframe>` sandbox](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/iframe) | Flag semantics; *"When the embedded document has the same origin as the embedding page, it is strongly discouraged to use both allow-scripts and allow-same-origin, as that lets the embedded document remove the sandbox attribute"* |
| [MDN CSP `sandbox`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Security-Policy/sandbox) | The same flags can be enforced **by response header** (so a direct hit on the URL is sandboxed too); without `allow-same-origin` the document is an **opaque origin**, `Origin: null`, no localStorage, no `document.cookie` |
| [VS Code webview security](https://code.visualstudio.com/api/extension-guides/webview) | Minimum capabilities, a CSP, never leak the host object |
| `docs/decisions/0036-extensions-host-and-apps-tab.md` | PiCode's own stance: sandboxed iframes are the industry's only trusted escape hatch for free-form UI; they demand a separate origin and CSP discipline |
| `docs/benchmarks/2026-09-02-live-browser-preview.md` | A live-view URL is a credential (Browserbase/browser-use); the Preview panel is a separate approved plan — this feature is a *file* renderer, not a stream |

### Verified behaviour (probe, Chromium via `agent_browser`, 2026-09-14)

Probe: a page sets `SameSite=Strict` (PiCode's cookie) and embeds a
`sandbox="allow-scripts allow-forms"` iframe on the same origin; the frame
beacons what really happened. Script and method kept in `/tmp` during the
study; the feature's own tests will pin the conclusions.

| Probe | Result |
|---|---|
| `<img>`, classic `<script>`, `fetch()` with `credentials:include`, POST | **No `Cookie` header; `Sec-Fetch-Site: cross-site`** — the sandbox does not carry the session at all |
| fetch / form POST | `Origin: null` — `originAllowed` already refuses it on purpose (`internal/auth/auth.go`: *"null" is a sandboxed or file: page: refuse*), and `Sec-Fetch-Site: cross-site` is refused first |
| `fetch('/whoami')` JSON from the sandbox | Not readable (no CORS header) |
| After the endpoint adds `Access-Control-Allow-Origin: *` | JSON readable, ES module executes — a token route can feed the sandbox without cookies |
| `window.top.location = …` | Blocked (SecurityError) |
| Inline `<script>` in `srcdoc` and in a `blob:` iframe, host CSP `script-src 'self'` | **Did not run** — both inherit the embedding document's CSP |
| Inline `<script>` in a served iframe (own HTTP response) | Runs — response CSP is the child's, not the parent's |

Consequences: (1) the preview **must be served from a real route**, not
`srcdoc`/`blob:`; (2) that route cannot use the cookie — the sandbox never
sends it — so it needs its own capability; (3) `allow-same-origin` is never
an option.

## The shape

```
FilePane / FileDocument
  │  POST /api/previews {owner, id, path, root}        (cookie-authed, origin-checked)
  ▼
internal/preview ticket store   token → {owner, root, path, session, expiry}
  │  { url: /preview/<token>/<path>, expiresAt }
  ▼
<iframe sandbox="allow-scripts allow-forms …" src="/preview/<token>/<path>">
  │  GET, no cookie; the token is the gate
  ▼
GET /preview/{token}/{path…}
  ├─ document: CSP sandbox + Referrer-Policy: no-referrer + ETag
  └─ assets:   allowlisted MIME + Access-Control-Allow-Origin: * + no-store
```

### Ticket (new `internal/preview` package)

| Rule | Value |
|---|---|
| Minted by | `POST /api/previews` — ordinary authed API (cookie/bearer, origin check) |
| Mintable | `.html` / `.htm` under the owner's resolved cwd (`resolveGitWorktree` like `…/blob`), `root` precondition honored |
| Token | 32 random bytes, base64url; in-memory map; 404 on unknown |
| Binding | session id of the minter (re-checked against `auth_sessions`; revoke a device ⇒ its previews die), plus the canonical root at mint time (stable tree for the token's life, unlike live-cwd reads) |
| TTL | 1 h (decision below); dies with the daemon; new mint on pane open / Reload |
| URL | `/preview/<token>/<relpath>` — the token is a path prefix so **relative assets inherit it** |
| Retention | never logged, never in the feed; responses carry `Referrer-Policy: no-referrer` and `Cache-Control: no-store` |

### Serve route (`internal/server/preview.go`)

| Rule | Value |
|---|---|
| Methods | `GET`, `HEAD` only |
| Gate | none from `internal/auth` — `/preview/` is not `/api`; the token is the gate (and the sandbox sends no cookie to authenticate anyway) |
| Paths | `relUnderCwd(root, path)`, then `filepath.EvalSymlinks`; anything resolving outside the root or into a dot-prefixed component is 404 |
| MIME | allowlist: `html htm css js mjs cjs json map svg png jpg jpeg gif webp avif ico bmp woff woff2 ttf otf wasm txt csv xml mp3 wav ogg m4a mp4 webm pdf glb gltf`. Everything else 404 (`.env`, key files and extension-less files are structurally excluded) |
| Directory | `…/dir/` → `dir/index.html` when it exists; never a listing |
| Document headers | `Content-Security-Policy: sandbox allow-scripts allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-pointer-lock allow-downloads; frame-ancestors 'self'`; nosniff; no-store; ETag `W/"<mtime>-<size>"`; Referrer-Policy; Permissions-Policy denying camera/microphone/geolocation/usb/serial/hid/payment |
| Asset headers | the same minus the CSP, plus `Access-Control-Allow-Origin: *` and `Cross-Origin-Resource-Policy: cross-origin` |
| Size | assets streamed (`http.ServeContent`, Range/HEAD for free) up to `maxAgentBlob` (32 MiB). The document is served the same way; the v1 pane mints only after its ≤1 MiB text read succeeded (Raw/dirty tracking), so a heavier document is a v1.5 item, not a silent failure |
| Not wrapped by | the app-shell `securityHeaders`/`appCSP` (registered mux pattern wins; `csp.go` additionally skips `/preview/` as defense) |

The app shell's `frame-src 'self'` already allows this URL; no change.

### Decision table — serve request

| Conditions | Action |
|---|---|
| Unknown / expired token, dead session | 404 (no oracle), pane shows one line + Retry |
| Path escapes root, symlink escapes, dot component | 404 |
| Extension not in the MIME allowlist | 404 |
| Directory without `index.html` | 404 |
| Method other than GET/HEAD | 405 |
| Document, allowed, no CORS request | serve with CSP sandbox + no-referrer |
| Asset, allowed | serve with ACAO `*`; module scripts and `fetch` work under the opaque origin |
| Previewed page calls `/api/*` | CORS blocks reads; `Origin: null` is refused for writes by the existing origin check |
| Previewed page top-navigates | sandbox blocks |

## UI

- `previewKind()` gains `.html`/`.htm` → `"html"`; the pane's existing rule
  already opens it in **Preview** with a **Raw** chip.
- Browser: `FilePreview` gains an `html` branch. Because minting needs the
  owner, a shared `usePreviewTicket({owner, path, root, enabled})` hook in
  `web/shared/client/` mints on open/reload, HEAD-preflights the URL and
  exposes `{url, state, reload}`; `FilePane`, `FileDocument` and `FileCard`
  call it. The iframe mirrors the sandbox flags and keeps the
  `file-preview-frame` look.
- Controls: existing Preview \| Raw chips + **Reload** + **Open in browser**
  (token URL). **Save** while Preview is showing reloads the frame.
  Empty file → *Nothing to preview.*; load failure → one line + Reload;
  too large / gone reuse the existing `fileMessage` copy.
- Honest limits in copy: the preview runs sandboxed — no PiCode session, no
  storage, no cookies; unsaved editor text is not in the preview and links
  open inside the frame (v1).
- Chat `FileCard`: v1 keeps the source excerpt and adds **Open preview**
  (one click to the file tab) instead of minting an iframe per card in a
  long transcript — inline artifact cards are v2.
- Mobile `FileDocument` gets the same Preview \| Edit toggle, Reload and
  Open; the sandbox works unchanged in mobile browsers.
- Not in reach (cross-origin): console output and error capture; an
  "Open in browser" tab is the DevTools path in v1.

## Phases

1. **v1 — the renderer.** ADR + ticket store + route + headers + kind map +
   pane/card/mobile wiring + tests + docs.
2. **v1.5 — the editing loop.** *Shipped 2026-09-14*
   (`feat/html-preview-live`, then `feat/html-preview-unsaved`).
   Preview-only documents over the text cap; live reload — the route records
   served asset paths (`preview.Store.Touch`, cap `MaxWatch` = 200/ticket),
   the pane's EventSource stats them every second on
   `/preview/<token>/__events`, and a change reloads the parent frame with a
   nonce, no script injection into the page; and unsaved editor text as the
   ticket's overlay (`PUT` on the document; a clean buffer re-mints so disk
   wins).
3. **v2 — a real preview origin.** Study, options and measurements in
   `docs/plans/html-preview-v2.md`; awaiting the owner's decision on five
   points (origin strategy, remote, storage lifetime, fallback, save). The
   non-boundary parts (inline artifact cards, viewport presets, console
   bridge) can ship alone; the door to preview a running dev server overlaps
   the approved browser-preview panel and is a network capability of its own.

## Acceptance (v1)

- Any `.html`/`.htm` in the tree, a terminal tab, a file tab, a canvas panel
  or mobile shows the rendered page: inline and external scripts, styles,
  images, `fetch('data.json')`, ES modules, a CDN import, forms.
- A page calling `fetch('/api/agents')` reads nothing; a POST to any `/api`
  route is refused; the preview route serves its document and assets with no
  session cookie at all (Go tests pin the headers and the anonymous request).
- The token URL opened directly in a browser is still sandboxed (header CSP).
- Expired/revoked/re-minted tickets, traversal, symlink escape, dotfile and
  off-allowlist requests are covered by table tests (every decision-table
  row, or named as debt in `docs/handoff/open/`).
- Empty, gone, too-large and reload-error states each show one line + one
  action; screenshots read in both apps, light and dark; overlay audit ok.
- `make ci` green; `docs-site/guide/files.md`, the `Preview | Raw` row in
  `docs/architecture/routes.md`, `security-model.md`, a changelog fragment
  and the handoff note travel with the change.

## Decisions (accepted as recommended, 2026-09-14)

| # | Question | Recommendation |
|---|---|---|
| 1 | Preview origin | **v1:** opaque-origin sandbox on the app origin. **v2:** a real second origin (storage/workers/dev servers). Do not put `allow-same-origin` on the app origin, ever |
| 2 | Sandbox flags | `allow-scripts allow-forms allow-modals allow-popups allow-popups-to-escape-sandbox allow-pointer-lock allow-downloads`; never `allow-same-origin`, never top-navigation |
| 3 | Ticket scope | owner's project root (VS Code/JetBrains parity; `../shared/style.css` works) + allowlist + dotfile denial. Tighter alternative: the document's directory subtree |
| 4 | Network egress | allow, like a normal browser tab. Residual risk named: the page can read *allowlisted* sibling files and post them out; only a per-preview "no network" toggle fully closes it |
| 5 | Ticket lifetime | 1 h, bound to the minting session; dies with daemon/revoke |
| 6 | Unsaved-edits preview | shipped with v1.5 (`feat/html-preview-unsaved`): the parent PUTs the editor buffer as the ticket's overlay |
| 7 | Inline chat card | v2 (v1: source + **Open preview**) |
| 8 | Mobile | v1 (shared hook + `FileDocument` toggle) |

## Alternatives rejected

| Alternative | Why not |
|---|---|
| `srcdoc` / `blob:` | Verified: both inherit the app CSP, so inline scripts die; relative URLs do not resolve to the project |
| Serve with the session cookie instead of a token | Verified: the sandbox sends no cookie; the pane would load only error pages |
| `allow-same-origin` (same origin as the app) | The page could read `/api` answers and mutate state as the user; MDN's own warning; ADR-0036's stance |
| Separate listener/host for previews in v1 | Best isolation but a second HTTPS surface and reachability problem across LAN/Tailscale/gateway/phone; v2 with its own ADR |
| Server-side render (Go HTML → image) | No scripts, no interaction; a different feature |
| Inline every asset into one document | Breaks sibling references, `fetch`, modules, and huge files |
