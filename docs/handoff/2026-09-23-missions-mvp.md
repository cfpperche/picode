# 2026-09-23 — feat/missions-mvp: persistent missions through execution and review

Implemented: approved M0–M3 scope, ADRs 0199–0200, store/API/feed, CLI/MCP reporting, browser/mobile Missions, Inbox decisions, executor transfer and evidence-bound owner acceptance.
Verified: after syncing main, `make close` passed formatting, vet, hook policy, Go tests across 27 packages, JS tests/builds, living docs and the fast-forward check; the final mobile control-height correction was rebuilt.
Native pilot: scratch :8471, Codex 0.156.1 → Claude Code 2.1.280; mobile Inbox answer, transfer without repeating the decision, stale-review refusal, old-session HTTP 403, explicit generation-3 rebind, fresh review, mobile acceptance at version 19 and daemon-restart persistence.
Decision table: D01–D17/D20–D22 map to fixtures and measured evidence in `docs/plans/missions-verification.md`; D18/D19 belong to future M4/M5.
visual-review: PASS from screenshot review of browser navigation/list/empty/blocked/conflict and mobile completed/error/empty/blocked states; mobile controls corrected to 36px, overlay audit ok.
Visual card: overlay inside yes; items readable yes; trigger usable yes; clipping, double scroll or dead hover no; next click obvious yes.
Limits: one controlled native task, browser phone emulation only; no physical phone or Windows-shell acceptance, no live proof for other providers, no exhaustive native-send crash injection.
Production was not deployed. Durable follow-ups are in `docs/handoff/open/missions.md`.
Integration: feature commit `dbbfde2c4`, landed on main at `f4cf3b514`; full `make ci` passed after the fast-forward.
