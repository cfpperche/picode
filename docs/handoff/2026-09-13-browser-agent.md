# 2026-09-13 — feat/browser-agent: the work browser gets its command channel

Shipped: ADR-0132 (`proposed` — needs your read) and its first half. The
shell's page opens one stream (`GET /api/browser/stream`, authenticated by the
session it already has: no new port, no new credential); the daemon pushes a
command down it (`internal/browser` hub — pending requests keyed by id, the
newest stream is the active one, with timeout, busy and cancel answers); the
shell runs it against the work-browser tab on screen and posts the answer to
`POST /api/browser/result`. One verb is shell-local: `shell.events` returns the
tab's recorded ring and never reaches the page.

Verified: `go test ./internal/browser/` 9 rows (no shell, round trip, newest
stream wins, detach, shell error, timeout forgets, cancel, unknown result,
busy); `go test ./internal/server -run TestBrowser` round trip plus the 404 and
400 edges; `web/browser` 8 rows for `browserChannel` (tier passthrough, the
events verb, no tab on screen, a refusal, bad frames, a failed post, close);
`npm run build:browser` green; `make close` → ci-scoped PASS.
visual-review: n/a (no UI surface changed; the channel is inert until the tool).
Not done / debts: the `browser` Pi tool and its per-agent policy source (the
default is the owner's call — ADR-0128 says deny by default, slice 4 owns the
UI), the navigation gate, and a cancel frame (a timeout ends the wait, not a
command already on the stream).
Merge: fast-forward ready.

## Next up

- The `browser` Pi tool (`internal/pipkg`) plus the per-agent policy source, then the navigation gate, then the policy UI.

## Debts

- Nothing calls `Dispatch` in production yet: the channel is inert until the tool lands.