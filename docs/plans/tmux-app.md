# tmux app — the server inventory (app plan)

- **Date:** 2026-09-14 · **Status:** accepted by the owner the same day
  (chat, "aprovado"), with these defaults taken in the absence of an explicit
  answer and named so they can be reversed cheaply:
  1. v1 is **read + remove one confirmed session**; attaching to a session no
     store row claims is **not** in v1 (it is a new door — §5, open question 1);
  2. **no "adopt"** of a lost session (creating a terminal row for a session
     that survived a restart) — it changes the ownership model (§5, question 2);
  3. **no socket isolation** for the daemon. The harnesses get a private
     server in the `tmux-isolation` branch; the daemon keeps the user's
     (ADR-0025's premise). Unchanged by this plan (§5, question 3);
  4. the name is **`tmux`** (technical, with the one-line subtitle the UI/UX
     bar allows for a term that must appear) (§5, question 4).
- **Companion work:** ADR [0133](../decisions/0133-tmux-app.md);
  [docs/architecture/tmux-app.md](../architecture/tmux-app.md);
  [docs-site/guide/tmux.md](../../docs-site/guide/tmux.md);
  `docs/changelog.d/tmux-app.md`.
- **Reference studied, not copied:** Tachyon's tmux Server Inspector
  (`~/tachyon`, read 2026-09-14) with its ADR-free inspector code, its
  dedicated socket + `-f /dev/null` isolation, its prefix-filter ownership, its
  3-second client poll and its `reapOrphans`. Adopt/adapt/refuse is §2.
- **Benchmark note:** the UI/UX bar in `docs/benchmarks.md` is the ruler.
  Adaptations cited: **Cursor**'s activity-feed density (compact rows, one fact
  per column, status before prose — the row here is identity, badges, the live
  command and the age, in that order) and **Linear**'s speed/density bar
  (13px UI type, a 4px rhythm, no chrome competing with the rows); the
  two-tab shape (Sessions / Server) is the Tachyon inspector's, kept because
  it separates what is running from what the server *is*. A `docs/benchmarks/`
  note was not written: the surface is a document of rows, not a new
  interaction model (contrast the Canvas, which added a plane and needed one).

## 1. The problem, measured

A PiCode terminal or agent IS a tmux session, and a session with no store row
is invisible in the product. Measured on this machine:

| When | Fact |
|---|---|
| 2026-09-13 (`docs/handoff/open/terminal.md`, the isolation branch) | the user's server held 147 PiCode sessions, **136 orphans**; nothing reaps automatically because "PiCode-owned but not in my store" is also how a second instance looks |
| 2026-09-14 (a script over `list-panes -a`) | 152 sessions, 151 `picode-`, **100 unclaimed** (8 <1h, 68 in 1–24h, 24 >1d), 6 still attached |
| 2026-09-14, later | 7 live sessions read as unclaimed — **all seven belonged to another PiCode instance** the script could not reach |

The third row is the design constraint: the measurement tool was wrong, not the
server, and no in-process read can do better. The product answer is an
inventory with the uncertainty written on it, plus a human confirming one row.

## 2. The reference: adopt, adapt, refuse

| Tachyon mechanism | Verdict here | Why |
|---|---|---|
| Dedicated socket `tmux -L tachyon` + `-f /dev/null` | **refuse** | PiCode shares the user's server by decision (ADR-0025 §1) |
| One `list-panes -a` for the whole server | **adopt** | one subprocess, no round-trip per session |
| Prefix filter as ownership | **refuse** | it is the rule behind the 29 killed sessions; and it cannot tell a leftover from another instance |
| `serverHealth` incl. two `ps` scans | **adapt** | keep version/socket/clients/keys format; drop the `ps` work (measured: 78 ms of the reference's ~100 ms tick) |
| 3s client poll | **refuse** | the change feed exists (ADR-0048) |
| Kill after a modal confirm + identity receipt | **adopt, stronger** | `sessionId` + `created` + `panePid` re-read from tmux, plus the marker's owner |
| Bulk "reap orphans" | **refuse** | a one-click sweep is the 2026-09-06 incident; the leak was 136 rows, not 136 clicks |
| Foreign sessions shown, read-only | **adopt** | the shared server is the product's own reality |
| Dead-pane vocabulary | **adapt** | PiCode does not set `remain-on-exit`, so a dead pane usually means a dead session; the badge appears only when the user's config produces one |

## 3. What shipped in this delivery

- `internal/tmux/server.go` — `ServerSessions` (one call, one row per session,
  active pane wins, `pane_start_command` last so a tab cannot shift fields),
  `ServerInfo`, `SessionReceipt` + marker constants. New file on purpose: the
  `tmux-isolation` branch owns the rest of that file, and this must merge
  without touching it.
- `internal/server/tmux.go` — the API family, attribution, the absence list,
  the decision table's handler, the audit event.
- `internal/apps/tmux.go` + `BuiltIns` — a native-surface app with no badge
  (a badge would run a tmux subprocess on every grid render).
- `web/browser/src/components/tmux/TmuxSurface.jsx` + `styles/tmux.css`, the
  icon key `tmux` in `AppIcon.jsx`/`Icons.jsx`, the lazy registration in
  `App.jsx`, and the second row in `web/tools/app-boundary.test.mjs`.
- Tests: `internal/tmux/server_test.go` (parsing, one-call invariant, marker
  variants, unreadable receipts), `internal/server/tmux_test.go` (the whole
  decision table, the absence list, the marker/constant pin), the app's
  manifest test, the boundary row.

### Amendment — 2026-09-14: primitives, not a native surface

The owner's production screenshot showed the native body drawn **on top of the
Docker app** (the component never applied the mount's `hidden`), and named the
UI/UX deviation from the app this was modeled on. The body moved back to the
frozen vocabulary on the Docker mold — phase 1+2 now live as `View`/`Action`
(`internal/apps/tmux.go`), `/api/tmux/*` and the desktop component/CSS/chunk
are deleted, `apps.Host` grows `Tmux TmuxServer` + `LostSessions`, and the
phone gains the app. Decision table, markers and audit event unchanged
(ADR-0133's amendment has the full argument).

## 4. Phases

| Phase | Content | State |
|---|---|---|
| 1 | read: `ServerSessions` + `/api/tmux/server` + the Sessions tab (own, unclaimed, foreign) | **shipped in this branch** |
| 2 | act: receipt-carrying removal, audit event, `tmux.changed`, the Server tab, the absence list | **shipped in this branch** |
| 3 | polish: filters, folds, the restart report rendered as history, badge with a cached read | open |
| 4 | open question 1 (attach to an unclaimed session) — needs its own ADR | not started |

## 5. Open questions (owner)

1. **Attach to an unclaimed session.** The valuable half of a leftover is that
   the work inside it survived; reading it needs a route for a session with no
   terminal row (a read-only attach). That is a new door and a new ADR.
2. **Adopt a lost session** (create a terminal row for a session that
   outlived its store). Real recovery, but it changes what ownership means.
3. **Socket isolation for the daemon** (the Tachyon model, already applied to
   harnesses). Separate decision; it changes ADR-0025's premise.
4. **Naming.** `tmux` is honest and the audience is technical; a product name
   is a one-line change in the manifest.

## 6. Debts this plan records

- No test drives a **real** tmux server: every test scripts the tmux seam, so
  the parsing and the routes are covered but a live `list-panes` against a
  running server is not. Safe-by-construction was chosen over a test that
  strands sessions on the developer's own server (the leak the isolation
  branch exists for).
- The absence list covers terminals only. An agent in managed mode legitimately
  has no session, so listing "missing agents" would be noise dressed as
  forensics; the agent case stays the sidebar's business.
- No badge: the app is opened when something is wrong (§3). A badge needs a
  cached read with a short TTL and should say *how many* leftovers exist.