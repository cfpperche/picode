# 2026-09-15 · feat/tmux-guard

**What shipped.** A `tmux` guard wrapper on the ADR-0056 session PATH
(ADR-0138): refuses `kill-server`/`kill-window`/`kill-pane`, `kill-session
-a`, pattern kills, and exact-name kills whose target lacks this terminal's
`PICODE_TERM_ID` marker; blocks `send-keys` payloads carrying
`kill-server`/`pkill`/`killall`; stamps `new-session` with the creator's
marker so agents can still clean up their own sessions. Default on,
`enabled.json` key `tmux-guard`, refusals logged to
`<dataDir>/tmux-guard.log`, copy printed on the pane. Wiring enable/disable
endpoints work; **no UI toggle yet** (recorded in `docs/handoff/open/terminal.md`).

**Measured root cause of the week's incidents.** `$TMUX` outranks
`TMUX_TMPDIR`: a client started inside a session talks to the server in
`$TMUX` no matter what `TMUX_TMPDIR` says, so "isolated" scratch work hits
production. Yesterday's `kill-server` and **this branch's own integration
test (twice today — my fault)** both came from that trap. The fixture now
scrubs `TMUX`/`TMUX_PANE`, asserts `#{socket_path}` under its temp dir
before acting, cleans up by exact names, never runs `kill-server`, and every
exec has a deadline.

**Also fixed by the guard's own test.** The shared `wrapperFindReal` shells
out to `dirname`; under a minimal PATH the guard found itself in its bin dir
and exec'd in an endless loop. The guard now resolves with `${0%/*}`
(`guardFindReal`); `TestTmuxGuardMinimalPathDoesNotSelfExec` pins it.

**Gates.** `make ci-scoped: PASS` (fmt, vet, hooks, go×4, docs, vale) —
run while the owner held other agents. No UI change, so no visual-review
scope; the UI row is a named debt, not claimed.

## Next up

- UI toggle for the guard (needs a UI home).
