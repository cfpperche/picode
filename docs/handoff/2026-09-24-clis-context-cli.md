# 2026-09-24 — feat/clis-context-cli

Shipped: the desktop rail's Agent CLIs button now follows the same context
rule as the other CLI panes (ADR-0179): with an agent selected in the sidebar
(or a terminal owned by one) the view opens directly at that agent's CLI —
`go("clis", …, {cli})` → `#/clis/<cli>` (`web/browser/src/lib/routes.js`),
rail handlers pass `ctxAgent?.cli` (`App.jsx`, both mounts). Without an agent
in context it keeps the legacy `#/clis`, whose first row is unchanged. Pi
agents land on `#/clis/pi`; bookmarks and deep links (`#/clis/pi`) are not
hijacked. Mobile More is unchanged by design — the phone has no persistent
sidebar selection to read.

Verified: `make ci-scoped` and `make close` green (fmt, vet, hooks, test-js,
build, living-docs). `routes.test.js` pins the new navigation: context CLI →
`#/clis/claude-code`; no cli → `#/clis`. On qa-scratch `clis-ctx`: a stopped
Claude Code agent (proint) selected → rail click landed `#/clis/claude-code`
with the Claude Code pane; Pi agent → `#/clis/pi`; workspace-only and fresh
`#/clis` loads unchanged. `overlayAudit` ok.

visual-review: PASS (subagent read of
`var/screenshots/clis-opens-on-context-cli.png`; card 5/5; catalog highlights
Claude Code, heading "Claude Code").

## Debts
- (none new — the change is a single-path navigation rule; covered by the
  pinned test above)
