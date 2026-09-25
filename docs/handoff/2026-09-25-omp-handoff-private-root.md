# 2026-09-25 — omp-handoff-private-root: Continue in… works for workspace Omp agents

Bug: "Continue in…" on a workspace Omp agent answered 400 "That session is not on this machine." Workspace Omp
agents launch with `--session-dir <dataDir>/omp-sessions/<agentID>` and the pin names that file, but
`OmpSource.Read` (`internal/clisession/omp_io.go`) accepted only paths under `~/.omp/agent/sessions`; the
agent-history locator knew the private root, the handoff reader did not.
Shipped (a061cf842): `clisession.Ref.Roots` (extra roots the caller vouches for); `clisession.OmpAgentSessionsRoot(dataDir)`
is the one definition used by the launch (`ompAgentSessionDir`), `NewAgentHistoryLocator` and the handoff route
(`cli_handoff.go` sets Roots for src omp). Paths outside both roots are still refused.
Verified: `TestOmpReadAgentPrivateRoot` (clisession); `TestHandoffOmpAgentPrivateSession` (server: preview 200, handoff
201 Omp→Codex with a fake codex, stray path 400) — fails without the fix with the exact live error. Live on scratch
`omph` with a copy of a real 13.6 MB Omp agent transcript: preview 200 (native, 33-message recent window), handoff
201 to a Claude Code terminal, native Claude session written, terminal running with `--resume`; copy deleted.
Blind spot: driven through the HTTP routes the dialog calls, not by clicking the dialog; Claude Code on the scratch's
isolated HOME stopped at its first-run theme screen, so the resumed TUI itself was not seen.
visual-review: n/a (no UI change)
Not done / debts: Omp Sessions listing still ignores the per-agent root (in `open/sessions.md`; paid by feat/omp-sessions-workspace).
Deployed 0.7.0+e6af921; the owner ran Continue in… on the live delivery agent from the UI and it worked (2026-09-25).
Merge: fast-forward ready.
