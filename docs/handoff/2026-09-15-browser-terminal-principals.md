# 2026-09-15 — browser-terminal-principals

Steps 1–2 of ADR-0143 (the owner picked this after the new-tab recheck).

## Done

- `packages/pi-browser/extensions/browser.ts` sends `term`
  (`PICODE_TERM_ID`) beside `agent`, the same rule as pi-inbox's
  `agentIdentity`.
- `browser.ResolveCaller` (ADR-0143 branch order: managed id → term id →
  unmanaged) and the tool handler resolves through it.
- `TestResolveCallerIsTheHouseIdentity`: six rows (agent wins over terminal,
  terminal grant, no grant on either side, unmanaged, ungranted agent next to
  a granted terminal) — the decision table in tests.
- Green: `internal/browser`, `internal/server`, `packages/pi-browser` tests.

## Not built yet (steps 3–4 of the ADR)

- The listing still returns managed agents only and the POST still requires a
  known agent id, so **a human cannot yet see or grant a terminal row**. The
  resolver half is in place and inert: every terminal still resolves to
  `Default()` (read on screen) until the next slice wires the visible half.
- Then the UI row per principal and the line about an outside-PiCode `pi`.
