# File preview: Markdown and HTML (ADR-0136, ADR-0137)

> Part of PiCode's architecture (ADR-0105: one file per subsystem). Edit here; the index only links.

The file pane has one kind registry (`web/shared/domain/filePreview.js`):
`svg`, `mermaid`, `markdown` and `html` (text) plus `image`, `pdf`, `audio`,
`video` and `model3d` (blob). Every other kind renders in the app as a
component; HTML is a page, so it gets an origin — a sandboxed one, or its
own.

## Markdown as a document

A `.md` file renders like GitHub renders it, in both apps, through
`MarkdownDoc` (one copy per app, ADR-0072) on one shared pipeline:

| Piece | Where | Rule |
|---|---|---|
| Pipeline | `web/shared/domain/mdPipeline.js` | `remark-gfm` + `remark-math` → `rehype-raw` → `rehype-sanitize` (hast-util-sanitize's default schema, which is GitHub's allow-list) → `rehype-katex` → alerts → heading ids. Only the steps **after** the sanitizer add classes; the author's HTML never reaches the DOM unsanitized |
| Document rules | `web/shared/domain/mdDocument.js` | frontmatter split (flat `key: value` → table, else raw YAML), github-slugger ids, `> [!NOTE]`-style alerts at the top level, link and image resolution |
| Ids | `DOC_ID_PREFIX` | every id is `user-content-…`, as on GitHub, so a heading can never shadow an id the app looks up; `#frag` links try the prefixed id first |
| Links | `resolveDocLink` | `#frag` scrolls inside the pane (the app routes by hash, so the page hash never moves); `http(s)`/`mailto` open a new tab; a relative path opens that file where the mount passes `onOpenPath` — the same pane in the file tree and mobile Files, a new file tab from a file tab — and is inert text elsewhere (canvas file panels, chat file cards); any other scheme or a path above the root is inert |
| Images | `resolveDocImage` | web and `data:image/` URLs as before (`safeImgSrc`); a relative path loads through the owner's `blob` file route — the same API the pane reads with, never a filesystem path in the page. An SVG is read through the `text` route and shown as a `data:` image (an `<img>` never runs its scripts), so the file API never serves SVG as a document on the app's origin |
| Code | `mdComponents` | the chat's SourceBlock (highlight, copy), MermaidBlock and diff fences; no Run button in a file |
| Outline | `MarkdownDoc` | read back from the rendered `h1`–`h3`, shown as a second column only when the pane is ≥1040px wide (container query) and there are ≥3 headings |
| Scrolling | `scrollToHeading` | jumps scroll the pane's own scroll box, never `scrollIntoView` — that also moves clipped ancestors and pushed the pane header out of reach; the outline's current entry follows a scroll listener on the same box |

### Split (source | preview)

For a markdown file the pane offers **Preview · Live · Split · Raw**. Split puts the
CodeMirror source beside the rendered document (after VS Code's side-by-side
preview) and is offered only while the pane body is ≥880px wide; a remembered
Split on a narrower pane reads as Preview until it widens. The last choice for
markdown is kept in `localStorage` (`picode-md-view`), per browser.

| Piece | Where | Rule |
|---|---|---|
| Source lines | `rehypeSourceLines` via `withSourceLines(offset)` | runs last in the pipeline and stamps `data-line` on blocks (paragraphs, headings, list items, table rows, fences, alerts…) from parser positions; `offset` puts the frontmatter's lines back so numbers are the file's own. Only Split asks for it |
| Mapping | `web/shared/domain/scrollSync.js` | `syncBlocks` keeps blocks whose line moves forward; `previewTopFor` / `lineFor` interpolate between the two blocks around a position (fractional lines); top and bottom of either pane pin the other to its top and bottom |
| Wiring | `web/browser/src/lib/useSplitSync.js` | editor scroll → preview, preview scroll → editor, each programmatic scroll ignoring its own echo for 120ms; the side the reader moved last leads, and a preview re-render (typing, late images, the lazy chunk) re-syncs to the editor; double-click on a preview block puts the cursor on its line and scrolls the editor so that line sits level with the block, which does not move |

The preview renders a deferred copy of the text (`useDeferredValue`), so a
keystroke never waits for the markdown render. Split is desktop and browser
only; the phone keeps Preview and Edit.

### Live (markdown rendered in the editor)

**Live** is the Raw editor with Obsidian-style live preview: the buffer is
still the file's own text (nothing is parsed into a model and written back, so
saving never reformats), but headings, emphasis, links, lists, task
checkboxes, quotes and GitHub alerts, code blocks, rules and images render in
place, and their syntax shows only on the lines a selection touches while the
editor has focus. It is one `Compartment` of the same editor, so Raw ↔ Live
keeps the cursor and the undo history.

| Piece | Where | Rule |
|---|---|---|
| Plan | `web/browser/src/lib/mdLivePlan.js` | `planLive(state, ranges, activeLines)` walks the lezer tree (`markdownLanguage`, GFM) over the visible ranges and returns data: line classes, marks, hides, labels, bullet/task/image widgets. Tested without a DOM |
| View | `web/browser/src/lib/mdLive.js` | a `ViewPlugin` turns the plan into decorations on doc, viewport, selection, focus and tree changes; replacements never cross a line break; the task checkbox edits `[ ]`↔`[x]` as one undoable change; images resolve like the preview (repo SVG as a `data:` image) |
| Links | `openLink` in `FilePane` | Ctrl/⌘+click: `#frag` moves the cursor to the heading with the preview's slug (`headingPos`), a relative path opens the file (`onOpenPath`), the web opens a new tab; a plain click only places the cursor |

| Tables | `web/browser/src/lib/mdLiveTables.js` + `tableField` in `mdLive.js` | a table spans lines and only a state field may replace across line breaks, so tables are a separate `StateField` (told about focus by `focusChangeEffect`): every table no selection touches is one block widget — header, alignment from the delimiter row, inline code/bold/italic/strike/links built as DOM nodes (never parsed HTML). A click puts the cursor in the cell it hit (positions re-read at click time), which turns the table back into source; Ctrl/⌘+click on a cell link follows it. Up/Down stop on a table's near row (`tableSkip`) instead of stepping over the widget. The widget is `contain: inline-size`, so a wide table scrolls inside itself and never widens the editor |

Raw HTML, math and frontmatter stay as monospace source in Live; Preview
renders them.

Styles live in `web/shared/styles/markdown-doc.css`, loaded after each app's
sheet and scoped under `.md-doc`, so the chat's `.md` message rhythm is
untouched. `MarkdownDoc` is a lazy chunk: previews of other kinds do not load
the highlighter, KaTeX or the sanitizer.

## Route family

One ticket, two shapes (ADR-0137). The **path form** keeps the opaque-origin
sandbox; the **host form** serves the same ticket from
`http://<label>.localhost:<daemon port>`, where the page is an ordinary
cross-origin page with its own storage. `preview.Store` owns the token *and*
the label; the label is a credential exactly like the token.

| Route | Answers |
|---|---|
| `POST /api/previews {kind, id, path, root}` | 200 `{path, expiresAt, sandbox:{url, events}, origin?:{url, events}}` — an ordinary authed API (cookie/bearer, origin check); `.html`/`.htm` under the owner's resolved folder only, `root` precondition honored (ADR-0074). `origin` is present only when the minting request's Host is this machine's loopback, and names the daemon's live port |
| `GET`/`HEAD` `/preview/{token}/{path…}` | the document or one allowlisted asset; everything else 404 "This preview is not available." (method-less mux pattern, so a POST answers 405 here instead of falling through to the app shell) |
| `PUT` `/preview/{token}/{document}` | the pane's unsaved editor buffer becomes the ticket's **overlay** (204); the ticket is the gate, only the ticket's own document can be overlaid, UTF-8 and ≤1 MiB (the editor's cap) |
| `GET` `/preview/{token}/__events` | the live-reload stream for that ticket: `hello`, then one `change` frame when a file the ticket served moved on disk |
| `GET`/`HEAD` `http://<label>.localhost:<port>/<path>` | the same document and assets on the ticket's own origin, `/` serving the ticket's document; `/api`, the app shell and every other route answer 404 here |
| `PUT`/`DELETE` `http://<label>.localhost:<port>/<document>` | the overlay (204) and its removal on Save (204); `OPTIONS` answers the pane's preflight |
| `GET` `http://<label>.localhost:<port>/__events` | the same live-reload stream, with the minting origin's CORS allowance (the pane's `EventSource` is cross-origin there) |

The host form is routed by `Host` in `previewHostHandler`, which sits
**outside** `auth.Wrap` on purpose: that origin is not the app and the ticket
is its only gate. A `Host` that is `<label>.localhost` but not a live ticket
answers 404 — never the UI. Everything else falls through to the gate
unchanged.

## Policy

- **Ticket** (`internal/preview`): 32 random bytes plus a 26-character
  lowercase base32 **label** (the origin name), in-memory, one hour, bound to
  the minting session (re-checked against `auth_sessions` on every request —
  revoking a device kills its previews) and to the canonical root at mint
time. The token is a path prefix so relative assets inherit it, and it is a
  credential: never logged, never fed, `no-referrer` on every answer.
- **Paths**: `relUnderCwd` then `filepath.EvalSymlinks`; anything resolving
  outside the root, any dot-prefixed segment, and any non-regular file is
  refused. A directory serves its own `index.html`, never a listing.
- **MIME**: the closed list in `preview.MIMEType` (web assets, fonts, media,
  wasm, pdf, 3d). `.env`, keys and extension-less files are structurally
  unservable.
- **Headers, path form**: `CSP: sandbox allow-scripts allow-forms allow-modals
  allow-popups allow-popups-to-escape-sandbox allow-pointer-lock
  allow-downloads; frame-ancestors 'self'`, `Referrer-Policy: no-referrer`,
  `Cache-Control: no-store`, `X-Content-Type-Options: nosniff`, a
  Permissions-Policy that denies camera/microphone/geolocation/usb, and
  `Access-Control-Allow-Origin: *` + `Cross-Origin-Resource-Policy:
  cross-origin` so ES modules and `fetch` work from the opaque origin.
- **Headers, host form**: the same `no-referrer`/`no-store`/`nosniff`/
  Permissions-Policy/CORP set, **no `sandbox`** (it would re-impose an opaque
  origin and defeat the whole point), `frame-ancestors
  <minting origin>` and CORS narrowed to that origin — the pane's own
  `HEAD`/`PUT`/`DELETE`/`EventSource` are cross-origin once frame and pane
  differ, so the form answers `OPTIONS` with `GET, HEAD, PUT, DELETE`.
- **Why a different host and not a second port**: measured 2026-09-14 in
  Chromium. `<label>.localhost:P` → `Sec-Fetch-Site: cross-site`, the
  host-only `SameSite=Strict` session cookie is **not sent**; `localhost:P2`
  → same-site, cookie sent. A different host keeps `Origin == r.Host`, the
  `SameSite=Strict` cookie and the missing CORS as three independent locks.
- **Pane**: one `usePreviewTicket` per app (browser and mobile own their
  own, ADR-0072) mints on open and Reload, HEAD-preflights the offered forms
  (origin first), and **PUTs the editor buffer as the ticket's overlay while
  it is dirty** — the preview shows what the pane holds; when the buffer is
  clean again the sandbox form re-mints (disk wins) and the host form clears
  the overlay (same origin, so the page's storage survives the save). The
  toolbar's **Reload** refreshes the frame on the same origin (`refresh`); the
  error state's **Retry** mints again, which is what a dead ticket needs. The
  `iframe` carries the sandbox flags **only in the fallback**, and
  `allow="fullscreen; clipboard-write"` in both. A document too large for the
  pane's text read (`>1 MiB`) renders **preview-only** from the ticket route:
  no Raw, no "too large" banner, because the page itself is there.
- **Fallback and remote**: a browser that cannot reach `<label>.localhost`
  (Safari unverified) gets the sandboxed path form and one line above the
  frame; a mint from a non-loopback Host (Tailscale, gateway, LAN) is never
  offered an origin at all, so remote viewers keep the sandbox by
  construction.

## Storage and what the page can reach

The host form is a real origin: `localStorage`, IndexedDB, Cache, Web
Workers, service workers and the page's own `document.cookie` work. Storage
is **fresh per preview** — a new ticket is a new origin, and closing and
reopening the pane starts clean; a service-worker registration from a dead
origin lingers in the browser profile until the browser collects it. The page
is also an ordinary cross-origin client of the machine: it can read whatever
a local service exposes to it (its price, documented in ADR-0137), while
PiCode itself stays out of reach — no CORS answer from `/api` (the deliberate
`/api/health` probe aside), an `Origin` that never matches `r.Host`, and a
cookie that never travels.

## Why not `srcdoc`/`blob:`

Measured 2026-09-14 (Chromium): both inherit the embedding document's CSP, so
the page's inline scripts die under the app's `script-src 'self'`, and
relative URLs do not resolve into the project. A served response carries its
own policy — which is why the ticket route exists.

## Live reload

The route records every file a ticket serves (`preview.Store.Touch`, capped
at `MaxWatch` = 200 paths per ticket; the document is served first and always
fits). The pane holds one `EventSource` per open preview on
`/preview/<token>/__events`; that connection stats the watched paths every
second and emits one `change` frame when an mtime or size moved (a path seen
for the first time is seeded silently, a vanished one reads `gone`). The hook
bumps a `v=` nonce on the iframe `src`, so the **parent reloads the frame** —
no script is injected into the page and the sandbox is untouched. Five
consecutive stream failures close it (an expired ticket 404s forever) and the
manual **Reload** remains.

Still open from the v2 plan (`docs/plans/html-preview-v2.md`): nothing in the
approved scope. A wildcard-subdomain origin with TLS (remote access to a real
preview origin) stays a separate future decision.
