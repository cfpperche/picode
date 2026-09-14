# 2026-09-14 — feat/html-preview: HTML file preview study and spec

Shipped: `docs/plans/html-preview.md` — references (VS Code Live Preview,
JetBrains built-in server, MDN sandbox/CSP, the repo's own ADR-0036 stance),
the probe results, the ticket + sandboxed-origin serving design, decision
tables, phases and the eight owner decisions still open. No production code.
Verified: probe on Chromium via `agent_browser` (2026-09-14; script in /tmp):
a same-origin `sandbox="allow-scripts"` frame sends no `SameSite=Strict`
cookie, carries `Origin: null`, cannot read same-origin JSON until the route
adds `Access-Control-Allow-Origin: *`; `srcdoc` and `blob:` inline scripts are
blocked by the embedding CSP, a served response's are not. `make ci-scoped`
PASS (docs-only scope); no `make ci` on this branch.
visual-review: n/a (no UI change)
Not done / debts: nothing implemented; the spec waits on the owner's answers.
Merge: not for merge — review branch.

## Next up

- Owner answers *Decisions needed* in `docs/plans/html-preview.md` (origin,
  sandbox flags, ticket scope/TTL, network egress, unsaved edits, cards, mobile).
- Implementation starts with `make adr NAME=html-file-preview` (Boundary:
  security model), then the ticket store, the serve route and the pane wiring.
