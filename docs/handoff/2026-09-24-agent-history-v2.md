# 2026-09-24 — feat/agent-history-v2: fast history lookup; a restore keeps the agent's id

Shipped:
- **Lookup.** `internal/clisession` finds Claude Code, Codex and Omp transcripts by the session id in the file name, with no listing of the store; builds on the other session's recorded-path fast path (5d780657d). Measured on the owner's real 13 exits: 8.0 s → 0.24 s, the same 8 found.
- **Same agent.** `Store.AddAgentAs` lets a restore keep the agent's id (ADR-0211, amends ADR-0205/0194), so automations, pins and edges on that id work again.
- **Undo.** Calls `POST /api/agent-history/{id}/restore` with `undo:true`: no transcript required, and a purged folder under the data dir's `work/` is made again. The client's snapshot path stays only as a fallback. `adoptPiAgentDir` removed (no longer needed).

Verified: go tests, including `TestAgentHistoryUndoWithoutTranscript`, which asserts the purge really happened; `make ci-scoped` PASS (first run failed on a transient DNS error fetching vale); `make close` green.
visual-review: PASS (`var/screenshots/agent-history-v2-*.png`, not committed): Undo and Bring back kept the ids and showed the conversation.
Blind spot: one scratch instance and the owner's WSL dataset; not run inside the Windows desktop shell or on the Windows dataset.
Notes for the owner, not defects: Undo restores the agent stopped while Bring back starts it; Undo switches the view to the restored agent.
Not done: phone screen and Windows latency stay open in `docs/handoff/open/agent-exits.md`. Not deployed (owner's call).
Merge: fast-forward ready.
Confirmed live 2026-09-24 after the owner's deploy: `GET /api/agent-history` answered in 0.02 s, and the owner restored a real removed agent from the history and it worked.
