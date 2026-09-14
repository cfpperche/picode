# 2026-09-13 — feat/browser-policy: the browser tool, and read as the default

Shipped: ADR-0134 (accepted: an agent with no grant reads the work-browser tab
the human has on screen; `act`/`full` and any other origin need an explicit
grant) and its implementation. `packages/pi-browser` registers the `browser`
tool — `snapshot` (accessibility tree), `screenshot` (PNG to a temp path),
`events` (the tab's ring) — naming a **verb**, never a CDP method; the daemon
maps it (`internal/browser.VerbFor`), resolves the grant
(`browser.policy.<agent>`, default read), and the shell re-checks the method
catalog. ADR-0132 is accepted as well; 0128's index row finally matches its
own file.

Verified: `go test ./internal/browser/` (policy default, fallbacks, save/
resolve, tier matrix, closed verb set) and `./internal/server -run TestBrowser`
(the tool route drives the stream, refuses an unknown verb); 8 rows of
`packages/pi-browser/test`; `make close` → ci-scoped PASS.
visual-review: n/a (no UI surface changed; the grants editor is next).
Not done / debts: slice 4 — the grants editor, the `act` verbs a grant
unlocks, and the navigation gate; the extension's registration is unverified
until it runs inside Pi (no tsc and no peer dep in this tree, same as the
other packages); the bridge, channel and tool still have no Windows runtime
acceptance.
Merge: **not merged** — `make close` runs the full matrix and two flaky
`internal/server` process tests fail it under load (see the debt in
communication.md): each passed 2 of 5 isolated reruns, and both attempts
failed on them. Everything this branch owns is green (the browser packages'
tests, the Go tests for the policy, verb table and routes, `npm run
build:browser`); the worktree is committed and clean, waiting for a gate that
is not red for reasons of its own.

## Next up

- Slice 4: the per-agent grants editor (Settings  Browser), the `act` verbs, and the navigation gate.

## Debts

- The `browser` extension has never run inside Pi here: its tool registration is unverified (the peer dep is not in this tree).