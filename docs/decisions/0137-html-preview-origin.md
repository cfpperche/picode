# ADR-0137: html-preview-origin

- **Status**: accepted
- **Date**: 2026-09-14
- **Boundary**: security model — the HTML preview moves from an opaque-origin
  sandbox to a **real per-ticket origin on the same loopback host**
  (`<label>.localhost`), reached through the running listener before the auth
  gate. What crosses it: the previewed page gains storage, workers and its own
  cookie, and becomes a normal cross-origin client of PiCode's host.

## Context

ADR-0136 ships the preview as an opaque-origin sandbox: no cookie, no storage,
no same-origin reads — and, as a direct consequence, no `localStorage`,
workers, or service workers. The v2 study (`docs/plans/html-preview-v2.md`)
measured what a real origin would cost:

- `*.localhost` resolves to loopback in Chromium on its own (no DNS, no hosts
  file), so a per-ticket hostname needs no infrastructure; `getent`, `curl`
  and Node do not resolve it, so tooling must keep using `127.0.0.1` + `Host`.
- A page on `<label>.localhost:P` calling `localhost:P` is **cross-site**
  (`Sec-Fetch-Site: cross-site`) and the host-only `SameSite=Strict`
  `picode_session` cookie is **not sent**; the same call from
  `localhost:P2` is same-site and the cookie **is** sent. A different host
  therefore protects more than a different port.
- PiCode's own gate already refuses a foreign origin: `auth.originAllowed`
  requires `Origin == r.Host` (host **and** port) for every mutating request,
  every WS upgrade and `/api/events`, and refuses `Sec-Fetch-Site: cross-site`
  outright; no route sends CORS headers except the deliberate `/api/health`
  probe (`{status, bootId, time}`, readable by anything on purpose), so
  cross-origin reads of real data are impossible; and `/preview` sits outside
  the gate because the capability is the ticket.

The owner approved the study's five recommendations (D1–D5) on 2026-09-14.

## Decision

A preview ticket gains a DNS-safe **label**, and the ticketed page is served
from `http://<label>.localhost:<daemon port>` — the same listener, routed by
`Host` **before** the auth gate: a live label serves the preview namespace and
nothing else (no `/api`, no app shell), any other host is PiCode exactly as
today. The ticket's path form (`/preview/<token>/<path>`) stays as the
sandboxed route, so both modes coexist.

The origin form drops the `sandbox` directive from its CSP (it would re-impose
an opaque origin) and drops the iframe's `sandbox` attribute — the separate
origin *is* the isolation. It keeps `no-referrer`, `no-store`, `nosniff`,
`Permissions-Policy` and `CORP`, sets `frame-ancestors` to the origin that
minted the ticket, narrows CORS to that origin (with the `OPTIONS` preflight
the pane's `PUT` now needs), and leaves the sandboxed path form's headers
(`CSP: sandbox`, `ACAO: *` — required by an opaque-origin page's own `fetch`
and ES modules) untouched. The pane prefers the origin form, falls back to the
sandbox when the browser cannot reach the label (Safari and other hosts,
stated in one line), and keeps the ticket across a Save so the preview's
storage survives. v2 is local-only: a non-loopback minting host is never
offered the origin form.

## Consequences

`localStorage`/`IndexedDB`/Cache, workers and service workers and the page's
own cookie work. Storage is **fresh per preview** — a new ticket is a new
origin, so closing the pane and reopening starts clean; a service-worker
registration from a dead origin lingers in the browser profile until the
browser collects it. Remote access (Tailscale, gateway) keeps the sandboxed
preview until a wildcard-subdomain option is decided.

The previewed page is now an ordinary cross-origin client of the machine: it
can read anything a local service exposes to it and send requests to PiCode's
host, where the cookie does not travel and the origin gate refuses. That is
the accepted price of a real origin. If we are wrong about the origin
comparison or the cookie scoping, the blast radius is a project HTML page
reading PiCode's API as the signed-in user; the measurement above, the
`SameSite=Strict` host-only cookie, the host/port-exact `Origin` check and the
absence of CORS are four independent locks, and the fallback path (sandbox)
never leaves the original v1 posture.

## Alternatives considered

- **Second loopback listener on its own port** (`127.0.0.1:P2`): simpler, but
  measured same-site, so the session cookie travels with the page's requests
  and only `Origin` + no-CORS stand; and every preview would share one storage
  bucket across projects.
- **Wildcard subdomain with TLS** (`https://<label>.<public host>`): the only
  option that works through a tunnel, but it needs wildcard DNS and a
  certificate the daemon does not have; deferred, not refused.
- **Staying sandboxed** (no v2): keeps the smaller surface, loses the reason
  the pane exists for real project pages (apps that need storage or a worker).
