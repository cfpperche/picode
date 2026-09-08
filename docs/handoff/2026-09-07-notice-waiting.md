# 2026-09-07 — feat/notice-waiting: needs-you cards, announce switches

Shipped, both phases the owner approved after the Superset study.
**Needs-you** covers the whole fleet, not just the agent whose socket is open:
the runtime already publishes every dialog edge as `agent.state` (ADR-0048),
`applyFleet` patches `waiting`/`dialog`, so `needsYou` — mobile's Now queue
since ADR-0044 — is live everywhere. `needsYouPlan` diffs it against what was
announced: arrivals become cards, answers withdraw them, and the first pass
after a page load only records, so a reload never replays a backlog. Sticky is
safe here (bounded by agent count, keyed `ask:<agent>:<dialog>`, always removed
by an event), and `asksOnSurface` withdraws a standing card when the user opens
the conversation that answers it; mobile stays quiet on Now, which *is* the
queue. **Preferences**: `closePlace` and `richColors` retired — the card draws
its own close control, a level reads from glyph + border — replaced by *When an
agent needs me* / *When a run finishes*, worded like the push switches. A
notice's `channel` names the switch that may mute it; feedback with no channel
never can. Only `error` is now exempt from suppression.

Verified: `make ci-scoped` PASS. Scratch (8471, isolated `--session`), waiting
staged by stubbing `/api/workspaces` + a feed reopen: card appears, goes when
the stub goes, goes on opening the conversation; `announceNeedsYou: false`
keeps the sidebar badge and shows no card; Preview muted and unmuted by its
switch. Mobile 390×844, Now badge 1.
visual-review: PASS (prefs, desktop card, mobile card; `overlayAudit ok`; 5/5)
Debt: the *finish* card still only fires for the agent whose socket is open;
neither card has met a real pi dialog — both staged at the HTTP boundary.
Merge: `git merge --ff-only feat/notice-waiting && make ci` from main.
