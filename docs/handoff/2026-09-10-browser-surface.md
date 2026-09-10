# 2026-09-10 — browser-surface: watch-only browser view over an authenticated stream proxy

Shipped: ADR-0114 + benchmark study (Lovable/Replit/bolt.diy/Orca/engine docs);
`GET /ws/browser` in `internal/server/browser_ws.go` (auth-gated proxy of the
engine's loopback stream; frames/status/url out, config/ack in, `input_*`
refused with a read-only envelope; engine death closes the client);
rendezvous discovery in `internal/rpc/browser_stream*.go` (implicit session
name + uid/0700/O_NOFOLLOW/liveness checks mirroring the ADR-0082 sidecar;
capture cache now carries sessionId too); `BrowserSurface.jsx` — the agent
tab's third view (Chat | TUI | Browser) with canvas frames, URL bar,
Watch-only chip and honest empty/ended states.

Verified: `make ci-scoped` green (fmt, vet, hooks, go tests, test-js, build);
scratch-instance E2E with a synthetic rendezvous + fake stream engine —
empty (idle), live (frames + URL) and ended states screenshotted and read;
overlay audit ok; Back-to-chat verified by click. Fixed in-session: the proxy
kept client connections open after engine death (regression test added).

visual-review: PASS (browser-empty/browser-live/browser-ended.png, card 5/5)
Not done / debts: input injection (interactive phase) needs its own consent
decision; mobile keeps the ADR-0082 capture pill (stream view is desktop);
multi-viewer pacing policy (per-client vs fanned-out acks) undecided; real
`agent-browser`-driven E2E (fake engine used for determinism).
Merge: fast-forward ready.
