# 2026-09-07 — feat/handoff-polish: the visual pass the handoff round owed

The approved plan for the cross-CLI handoff (ADR-0088/0094) asked for a
browser pass over the Sessions surface; two rounds shipped without it and
neither session note said so. Done now on a scratch instance.

Found and fixed: a session written from a source with no title of its own
was named after the handoff note, so the listing showed 120 characters of
"Handoff from Claude Code running…" — it now falls back to the source's
own first turn, then to "From <CLI>". The dialog named a session by its
full UUID, which wrapped the opening line; it uses a short id. The summary
line spelled out a CLI's internal record types ("bridge session left out
(4)"); unlabeled kinds are now counted, not named.

Checked and correct: the CLI picker and the row menu list only what
`GET /api/clis` advertises; the Hermes target shows the brief option
disabled with its reason; the brief preview expands with its size; a real
handoff to Grok from the UI opened the terminal and both rows then carried
the lineage badge, linking to it. Mobile renders the same surface.

Not seen: the live-source warning and its confirmation, which need a
terminal that has reported activity and pinned its session. Covered by
`TestHandoffLiveSourceNeedsForce`.

Captures in `var/screenshots/handoff-*.png` (local, not public docs).

Merge: pending `make close` and fast-forward of main.
