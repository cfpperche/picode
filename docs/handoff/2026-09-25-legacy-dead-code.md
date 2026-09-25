# 2026-09-25 — feat/legacy-dead-code: legacy dead code and group-2 fallbacks retired
Owner asked for "grupo 1 e grupo 2". Commits: 94509f8b0, 0903dd306, 05f67bed1, d0decb9d7, e81e058e7.
Group 1 (dead code) removed: stripLegacyUserClaudeHooks/claudeSetWiring/groupHasMarker (owner's
~/.claude/settings.json had no PiCode hook); inbox-burst reconciliation (prod DB held 0 such tasks;
ADR-0059 amended); #/clis/<cli>/launch alias rewrite. Reclassified and kept, comments corrected:
termGroups/workBack null guards, terminalCli top-level cli (term_state.go still copies it), Outcomes
skills .catch, Codex `agent-turn-complete` (third-party notify), empty-runID hook reports.
Group 2 (measured first): /api/apps/inbox/{view,action} aliases removed (UI uses /api/inbox; ADR-0208
amendment extended). Pre-stamp port fallback in apps/tmux.go tmuxElsewhere + Host.LoopbackURL removed
(all 20 live prod sessions carry PICODE_INSTANCE; ADR-0141 amended — code cites it as "ADR-0140" by a
number collision; 0140 verified unchanged). Safety rule changed: an unstamped session is no longer
another instance's, so reap may remove it (only pre-ADR-0140 binaries left sessions unstamped).
WebTab pre-ADR-0161 hide-and-freeze path removed (coverDecision, verifyPreviewUrl,
subscribeFloatingLayers, still backdrop, data-covered; overlayAudit no longer accepts a parked page);
ADR-0161 amended; installed picode-shell.exe (2026-09-25) has __PICODE_LIVE_LAYERS__. btab_preview kept
(annotation captures; preview.rs comment left stale, no shell rebuild). Pi package mutations answer
the unified report (writePiPackageReport); loadPackageReport stays for roles/config pages.
Verified: `make ci-scoped` PASS, `make close` green. Scratch: /api/apps/inbox/view 404, /api/inbox/view
200; agent-layer POST/DELETE /api/packages answer rows/scopes/caps (row appears, then goes). Blind
spot: the WebTab change runs only inside the desktop shell and was not exercised live.
visual-review: PASS (ldc-inbox.png, ldc-packages.png; overlayAudit ok). Merge: fast-forward ready.

## Debts

- Work-browser first live run and remaining legacy items: docs/handoff/open/legacy-compat.md
