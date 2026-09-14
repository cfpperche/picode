# 2026-09-13 — snips-debts: `open/snippets.md`'s three debts, and what checking found

**Decode cap.** The six `snips` decodes read through `snipDecodeLimit` (256 KB).
`TestSnipDecodeIsCapped` separates the layers by message (past the store's
100 KB `body is too long`, past the cap `invalid JSON body`), since status
alone cannot — both are 400. Decisive: it fails without the cap.

**`snip.ran` and 404.** Kept and pinned: an id that does not exist is not a run
(feed silent); a run that could not deliver publishes `ok:false`.
`TestSnipRanFeedBoundary` covers both rows.

**Shell door and `repoBusy`.** The door refuses (409 `busy`, naming who) when a
PiCode agent or terminal works in the target pane's repository — the rule
`handleTerminalRun` follows. The snippet text is arbitrary, so the repository's
state decides; `preview:true` is never blocked (it types nothing). Decisive:
without the guard the test sees 200 `typed:true` in a busy repo.

**The owed visual review** found **delete did not exist** on the phone, and
archive would have been a one-way door (the list fetched active rows only).
Shipped both plus the **Archived** view, with the desk's confirm copy. Read:
empty, list, detail, confirm sheet, Archived view, run sheet picker and its
confirm; delivery proved end to end (the pane printed `snippet-ran-ok`). Also:
`.btn-danger` had only a `:hover` rule, and a phone has no hover. Not verified
live: the busy refusal through the UI (needs an agent mid-turn) — the server
test covers it. Docs: architecture, changelog, `open/snippets.md`.
Merge: fast-forward ready.
