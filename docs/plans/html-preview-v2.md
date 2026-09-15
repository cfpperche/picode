# HTML preview v2 — a real second origin (study)

- **Status:** study, measured 2026-09-14; **implemented** the same day on
  `feat/html-preview-origin` (ADR-0137, accepted) with D1–D5 as recommended.
  The measurements below were confirmed by the Go tests (host routing, header
  and CORS split, off-loopback mint) and by the visual QA on a scratch
  instance (storage and its lifetime, the app out of reach, the fallback
  line).
- **Boundary:** security model + process (a second origin and how it is
  reached) — hence the ADR; the parts of v2 that cross no boundary are listed
  at the end and can ship on their own.
- **Predecessor:** `docs/plans/html-preview.md` (v1 + v1.5, ADR-0136 accepted).

## What v2 buys

The previewed page stops being an opaque sandbox and becomes an ordinary page
on its own loopback origin. Everything a "real" page expects starts working:

| Capability | v1 (opaque sandbox) | v2 (real origin) |
|---|---|---|
| `localStorage`, `sessionStorage`, IndexedDB, Cache API | ✗ (throws/absent) | ✓ |
| Web Workers, Service Workers (offline/PWA work) | ✗ | ✓ |
| `document.cookie` — the page's *own* | ✗ | ✓ (never PiCode's) |
| `crypto.subtle`, `navigator.clipboard` | mostly ✓ (localhost is a secure context, sandbox does not strip it) | ✓ |
| Third-party embeds that need storage (maps, auth widgets) | ✗ | ✓ |

Not in v2: PiCode data, PiCode's own storage, and a *stable* identity for the
preview across sessions (each preview is its own origin — see D3).

## What changes in the threat model

The frame is no longer opaque, so it becomes a page that can send requests —
including credentialed ones — to PiCode's host, open WebSockets and register
service workers. What already stops it (all in `internal/auth`, verified by
reading the code and by measurement):

| Defence | Where | Effect on a preview page |
|---|---|---|
| `picode_session` is **host-only** (no `Domain=`) and `SameSite=Strict` | `auth.setCookie` | cross-site requests carry **no cookie at all** |
| `Origin == r.Host` (host **and port**) unless `Origin` is absent | `auth.originAllowed` | a preview origin is a foreign origin: any mutating request, any WS upgrade and `/api/events` are refused (`403 cross-site request refused`) |
| `Sec-Fetch-Site: cross-site` refused outright | `auth.originAllowed` | the browser's own header closes the door first |
| **No CORS headers anywhere** in the server except the deliberate `/api/health` probe | routes | a cross-origin `fetch` cannot read a response, even if it is sent |
| The preview host serves **only** the preview namespace | `preview.go` | a same-origin `fetch("/api/…")` from the page is a 404, not the API |

Measured in the PiCode QA Chromium (probe: a page on one loopback origin
calling another, `credentials: "include"`):

| Preview origin → PiCode on `localhost:P` | `Sec-Fetch-Site` | `picode_session` sent? |
|---|---|---|
| `http://<label>.localhost:P` (**same port, different host**) | **cross-site** | **no** (`cookie=""`) |
| `http://localhost:P2` (**same host, different port**) | same-site | **yes** (`cookie=…`) |

That measurement decides the shape: **a different host is worth more than a
different port.** With a subdomain origin the cookie never travels; with a
second port it does, and only the `Origin` check and the missing CORS stand
between the page and the API. The measurement also confirmed that Chromium
resolves `*.localhost` to loopback on its own (no DNS, no hosts file) — while
`getent`, `curl` and Node do **not** (they need `127.0.0.1:P` + a `Host:`
header), which matters for tests and CLI tooling, not for the browser.

## Options

| | Origin | Storage | New port | Works through a Tailscale/gateway URL | Cost |
|---|---|---|---|---|---|
| **A** | `http://127.0.0.1:P2` (second listener) | one bucket shared by every preview | yes, loopback only (no firewall prompt) | no | small |
| **B** *(recommended)* | `http://<label>.localhost:P` (Host routing on the existing listener) | one bucket **per preview**, fresh | no | no | small |
| **C** | `https://<label>.<public host>` (wildcard DNS + TLS) | per preview | no | yes | large — wildcard cert, DNS, gateway mapping |
| **D** | stay sandboxed (v1) | n/a | no | — | zero |

`*.localhost` support: Chromium/Edge/WebView2 (measured here) and Firefox 84+
resolve it; Safari is **unverified**. If the browser cannot reach the origin,
the pane keeps the v1 sandboxed preview and says so in one line (D4). Remote
access (Tailscale, gateway) can never reach `*.localhost` — that is the
client's own machine — so v2 is a local-desktop capability; remote keeps the
sandbox until C is decided.

## Shape if B is chosen (the ADR's decision section)

1. A ticket gains a DNS-safe label (lowercase, ≤63 chars) next to the token;
   the v1 path form (`/preview/<token>/<path>`) stays as the sandboxed route,
   so both modes coexist and the fallback needs no new code path.
2. The listener routes by `Host` **before** the auth gate: a live label serves
   the preview namespace only; every other host is PiCode exactly as today.
3. v2 responses **drop `CSP: sandbox`** (it would re-impose an opaque origin)
   and keep `Referrer-Policy: no-referrer`, `Cache-Control: no-store`,
   `X-Content-Type-Options`, `Permissions-Policy`, `Cross-Origin-Resource-Policy`;
   `frame-ancestors` names the origin that minted the ticket (stored with it),
   not `'self'`. CORS splits by route form: the **path form** (sandbox
   fallback) keeps `Access-Control-Allow-Origin: *` because an opaque-origin
   page's own `fetch('data.json')` and `<script type="module">` are
   cross-origin requests; the **host form** narrows to `ACAO: <minting origin>`
   (which is the same origin the pane itself calls, and the iframe no longer
   needs `*`) and answers the `OPTIONS` preflight the pane's `PUT` now
   triggers, since pane and frame are different origins in v2.
4. The iframe **drops the `sandbox` attribute** in v2 — the separate origin is
   the isolation — and keeps `allow="fullscreen; clipboard-write"`.
5. Overlay and live reload are unchanged (same PUT/SSE), except that in v2 a
   save reloads the frame on the **same** ticket instead of re-minting, so the
   preview's storage survives the save (the buffer and the file are equal at
   that moment anyway).

## Risks that remain (honest)

- The page still reaches the network (same as v1) — only PiCode and the local
  machine's services are out of reach, and only by the table above.
- Preview storage is fresh per preview and dies with the ticket (1 h, session
  bound). Service-worker registrations from a dead origin linger in the
  browser profile until the browser garbage-collects them.
- Anything reachable on the machine that **trusts the origin** (a dev server
  with `Access-Control-Allow-Origin: *`, a service on another loopback port)
  is now reachable by the preview page as an ordinary cross-origin client —
  it always could *send*, but now it can also *read* whatever that service
  allows. This is the price of a real origin and should be said in the docs.
- Safari unverified; CLI/QA/tests must use the loopback + `Host` form.
- "Open in browser" would open the v2 origin, not a sandboxed one — a
  behaviour change for that button to document.

## Decision — owner's call (approved 2026-09-14)

| # | Decision | Recommendation |
|---|---|---|
| D1 | Origin strategy | **B** — `<label>.localhost` on the existing port (measured cross-site; no cookie, no new port, storage per preview) |
| D2 | Remote access | **Local only in v2**; C (wildcard DNS + TLS) stays a separate future decision |
| D3 | Storage lifetime | **Fresh per preview** (consequence of B); a preview is not a session |
| D4 | Browser that cannot resolve `*.localhost` | **Fall back to the v1 sandbox**, one line in the pane; a loopback second listener (A) only if a real user hits it |
| D5 | Save | **Same ticket/origin** — storage survives Save |

Shipped as written: `previewHostHandler` routes the label before the auth
gate, the origin form drops the CSP sandbox and the iframe's `sandbox`
attribute, the pane falls back with a line, and Save issues a `DELETE` that
hands the document back to disk without changing the origin.

**Not part of this decision** (no boundary crossed, each can ship alone and
without the ADR): inline artifact cards in the chat, viewport presets, the
console bridge, and the door to preview a running dev server (which overlaps
the approved browser-preview panel and is a network capability of its own).
