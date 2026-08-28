# Session surface — next roadmap

- **Date:** 2026-08-27
- **Status:** plan. Track C (waiting, queue, draft) shipped.
- **Why:** the address bar, session chip, `@` picker, and session tree still
  hide the agent you are in. Recorded in
  [conversation-control-roadmap](conversation-control-roadmap.md)
  (next-roadmap) and [study G4–G7](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md).

t3code thread routes · Cursor @-mentions and checkpoints.

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **D1 — `#/agent/<id>`** | **shipped** |
| 2 | **D2 — cost on the session chip** | **shipped** |
| 3 | **D3 — `@skill` / `@agent` mentions** | **shipped** |
| 4 | **D4 — rewind / checkpoints** | **shipped** (new session from a prompt; in-place leaf still pi#8645) |
| 5 | **D5 — package updates** | **shipped** |

Do not start llama installer, mobile parity, worktrees, broker, ACP, or IDE chrome (LSP) while Track E is open. D1–D5 shipped.

## Refuse

| Temptation | Why not |
|---|---|
| Agents talking to each other via `@agent` | mention = context in **this** prompt. Broker is later. |
| Cloud-synced URLs / accounts | this machine; hash is enough |
| Cost as a new page | it belongs on the session chip the user already clicks |
| `/tree` in-place jump | needs pi `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)) |
| Auto-update packages | user clicks **Update**. A badge is enough to notice. |
| System crontab | poll on Packages open + a slow interval. Not a machine cron. |

## D1 — `#/agent/<id>`

Today `#/` is the workspace. The open agent lives in `localStorage` tabs.
Reload usually restores it; a shared or bookmarked link does not.

**Ship:** workspace hash is `#/agent/<id>` while an agent is open.
The URL wins over tabs on load. `#/` still means “last agent” and is
replaced (not pushed) to `#/agent/<id>`. Other hashes (`#/mcps`, …) unchanged.

**Not:** query `?agent=`. Path like pins (`#/pins/<id>`).

| # | hash | agent exists | action |
|---|---|---|---|
| 1 | `#/agent/x` | yes | select x, open tab |
| 2 | `#/agent/gone` | no | one line “That agent is gone.” + pick another (sidebar) or Add workspace |
| 3 | `#/` | last tab | keep last; replace hash to `#/agent/<id>` |
| 4 | user picks y | yes | write `#/agent/y` (history push — Back works) |
| 5 | `#/mcps` (or other pane) | keep selected | do not rewrite |
| 6 | pane → Chat | yes | `#/agent/<id>` |
| 7 | close last tab | none | `#/` |
| 8 | F5 on `#/agent/x` | yes | x, even if tabs said y |

D1 **shipped** (2026-08-27). visual-review: PASS (`agent-url.png`, `agent-url-gone.png`).
Hash apply is hash-driven only — do not depend on `selectedId` or the tab strip fights the URL.

## D2 — cost on the session chip

`/session` and the footer already know tokens and dollars. The session
chip in the composer does not.

**Ship:** when cost > 0, the chip shows `$0.12` next to the name.
Zero cost: no extra word (not `$0.00`).

D2 **shipped** (2026-08-27). visual-review: PASS (`session-cost.png`).

## D3 — `@skill` / `@agent`

`@` today lists files. Same token, extra rows: skills and other agents.
Inserts text into **this** prompt (like `@file`). Not a message to that agent.

D3 **shipped** (2026-08-27). `@` lists Agent / Skill / File. Inserts `@agent:Name` or `@skill:name`. The open agent is omitted.

## D5 — package updates

The TUI already banners **Package Updates Available** (`pi update --extensions`).
`#/packages` still shows **Installed** with no way to update (2026-08-27:
`pi-mcp-adapter` was behind; the GUI did not say so).

**Ship:** periodic check (open Packages + background interval). User menu
badge when any installed package has an update. That row shows **Update**
(not a second Installed). Click runs the update. No badge when none.

**Not:** silent auto-update. npm/architecture copy in chrome. macOS/Windows
crontab.

D5 **shipped** (2026-08-27). npm packages only (git / path / pinned skipped).
Check: `GET /api/packages/updates`. Click: `POST /api/packages/update`.

## D4 — rewind

Session JSONL is already a tree (`id` / `parentId`). `/tree` shows it.

D4 **shipped** (2026-08-27): current card says **Now** (not clickable). Other
cards confirm then start a **new session** from that prompt. This one stays.
In-place leaf jump still needs pi `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)).

## Where it lives

| Thing | Path |
|---|---|
| This plan | `docs/design/session-surface-roadmap.md` |
| Hash routes | `web/src/lib/routes.js` |
| Workspace shell | `web/src/desktop/App.jsx` |
| Session chip | `web/src/components/SessionBar.jsx` |
| `@` picker | `web/src/lib/atMention.js`, `Composer.jsx` |
| Package updates (D5) | `web/src/components/Packages.jsx`, `UserMenu.jsx` |
