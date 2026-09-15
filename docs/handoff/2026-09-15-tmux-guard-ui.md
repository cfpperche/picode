# 2026-09-15 · feat/tmux-guard-ui — phase 3/4 of tmux-resilience

**What shipped.** The guard toggle, on **Preferences ▸ Terminal (Terminal
defaults)** as a **Safety** section: state line (On/Off) with the exact
behaviour, a Radix switch, the "applies to terminals opened from now on"
line and a docs link. Optimistic switch with revert; a failed refresh shows
a notice with **Try again** instead of silently keeping the old state (that
silent case was found by the visual pass and fixed). The wiring status
endpoint now ensures the guard on first read, so a fresh instance reports
the default-on truth.

**Verified on a scratch instance** (`qa-scratch`, :8475), screenshots read:
on, off, failed-refresh notice, failed-toggle toast + revert; overlay audit
`ok:true`; state survives reload. **End to end:** a seeded managed terminal
ran `tmux kill-server` and got the refusal copy + `EXIT=1`, the server
stayed up, and the line landed in the scratch's `tmux-guard.log`. The
pre-existing test `TestInterceptDoesNotWriteUserClaudeSettings` was updated
for the new truth (the guard is the one wrapper left after disabling every
CLI; disabling it too restores the no-interception state).

**Deviations / debts.** Mobile has no terminal-settings page, so the toggle
is desktop-only for now (the guard stays default-on there) — recorded in
`docs/handoff/open/terminal.md`. Screenshots live in `var/screenshots/`
(`tmux-guard-ui-on.png`); three other state captures were read and then
deleted by mistake, evidence kept in this note.

**Gates.** `make ci-scoped: PASS`.
