# 2026-09-24 — feat/session-browser-no-focus: agent browser stops stealing the tab
Shipped: `ensureSession` (web/browser/src/App.jsx) reselected the agent's
tab before EVERY session verb, so an agent browsing pulled the owner off
whatever tab they used (owner report with screenshots). Now the pure
`sessionReveal` (lib/sessionBrowser.js) decides: only `shell.open`, a missing
host tab or a missing split select the session; every other verb runs in the
split where it is — what ADR-0172 already said. A screenshot of a split off
screen waits 4 s, then reveals it once and retries (lib/browserChannel.js).
Verified: node tests for the decision table and the fallback (16/16);
`make ci-scoped` / `make close` green. Blind spot: not exercised in the
Windows shell (no scratch path for the shell; needs a deploy) — no visual
change, behavior only.
## Debts
- Hidden WebView2 screenshot/click behavior unmeasured → docs/handoff/open/work-browser-tabs.md
- Background mode (never reveal) awaits a superseding ADR → docs/handoff/open/work-browser-tabs.md
Merge: fast-forward ready.
