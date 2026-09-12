# 2026-09-10 — Matrix phase 4: the conversation panel

**Branch** `feat/matrix-chat-panel` (from `83cacc4e`). Plan
`docs/plans/matrix-app.md` §2.4/§4.4/§4.5; its phase 4 row is now Done.

**Shipped.** `web/desktop/src/hooks/useAgentSocket.js`: one
`/ws/agent?agent=` per mount over the desktop's own `lib/agentEvents.js`
(no third copy), transcript reconciled on snapshot and settle. The composer
verbs are **not** ported — phase 5 lands them verbatim from the mobile hook
— and only the reducer's `scroll` effect runs, so a reader never writes.
`AgentChatPanel.jsx` renders the tab's `Conversation` in a new `readOnly`
mode (no composer, queue controls, ask form or snippet Run) in the panel's
one scroller, sticky to the bottom. Needs you: the chip stays the fleet's,
the open question renders read-only, and a bar on the body's floor carries
it with **Open**.

**Cost rules** (`hasChat`, `loadPolicy`, one test per row): live in band at
zoom ≥ 0.4; name-plate with the socket **closed** below 0.4; unmounted
outside the band after the same 5 s; quiet past `CHAT_LIVE_MAX` = **12**,
LRU on when a panel entered the band. Measured with 20 live managed agents
on one matrix: 9 sockets at rest, 12 at the cap with 6 quiet mid-scroll,
never 13, 8 after the hysteresis.

**Debts.** A live chat panel counts as a watcher for `Hub.Len()`, so it
suppresses the unobserved-result item and the needs-you push as an open tab
does (same rule, written down in `docs/architecture/matrix.md`). The App
socket and a panel's hand off rather than coexist, so two sockets for one
agent were proven with a raw second connection. No reply from the panel —
phase 5.
