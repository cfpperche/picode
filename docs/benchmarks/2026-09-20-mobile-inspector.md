# Study: the agent inspector on a phone

- **Date:** 2026-09-20
- **Sources:** the in-house rail study
  ([2026-09-05-inspector-rail.md](2026-09-05-inspector-rail.md)), the phone
  supervision study
  ([2026-09-01-mobile-agent-supervision.md](2026-09-01-mobile-agent-supervision.md)),
  the mobile v2 study ([2026-09-07-mobile-v2.md](2026-09-07-mobile-v2.md)),
  and one fresh shipped surface: AgentsRoom (App Store, September 2026), a
  remote control for coding CLIs whose release notes advertise an
  "enhanced mobile panel for file changes" and whose listing promises
  "review the diff of what an agent changed" from the phone. Desk study;
  AgentsRoom observed only through its store listing.

## What the references decide

| Source | The phone-side inspector | PiCode adaptation |
|---|---|---|
| Paseo / Orca / t3code (rail study) | Changes / Files / PR in a persistent rail, `+N −M` sums, `PR #n` tab label | Same three surfaces and vocabulary; the persistent rail itself is refused below 768 px (ADR-0078) |
| Codex in ChatGPT, Cursor for iOS | Review diffs, approve, redirect — supervision, not an IDE | The Inspector is a review surface: it hosts no editor; a tap hands the file to the existing mobile Files tool |
| AgentsRoom | A per-agent file-changes panel on the phone | Confirms the unit of review is **one agent's** changes, not the whole folder — PiCode's `All | This agent` scope, now on both surfaces |
| mobile v2 constraints | One view owns the viewport; sheets for actions; header ≤56px | A pushed screen, not an aside; scope chips and worktree rows are flat lists; Git actions stay in the sheet |

## What PiCode ships

The Inspector as a pushed screen (`#/inspector/{a\|t\|w}/<id>`), anchored
to the owner the way the rail is: Changes (folder-grouped sums, session
scope, multi-worktree view from `resolveSessionView`), Files (lazy browse
that hands off to the Files tool), PR (`PR #n` tab label, checks, review).
The agent screen wears a glance line — changed files, `+N −M`, PR state —
rendered only while there is something to review. The change-shape logic
is shared (`@picode/shared/domain/inspector.js`), so the rail and the
phone cannot disagree about scope or worktrees.

## What PiCode refuses

| Temptation | Why not |
|---|---|
| A second rail or split view on the phone | ADR-0095: tools take the whole viewport; the keyboard owns the rest |
| A PR read per keystroke of navigation | One read per screen mount, server cache per folder+branch, feed-driven reload only (ADR-0048) |
| Editing inside the Inspector | Card or tab, not a third hatch (ADR-0078); the Files tool owns the editor |
| Merging from the phone | Merging stays a workstation act (supervision study) |

## Ritual

Cited by the mobile-inspector branch; the surface's acceptance table lives
in `docs/plans/mobile-v2.md`'s decision-table style tests
(`mobileRoutes.test.js`, `fleetReads.test.js`, `folders.test.js`) and the
shared suite (`web/shared/domain/inspector.test.js`).
