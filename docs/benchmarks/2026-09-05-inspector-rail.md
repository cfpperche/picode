# Study: the right-hand inspector rail (Paseo, Orca, t3code)

- **Date:** 2026-09-05
- **Sources:** three owner screenshots taken 2026-09-05 — Paseo
  ([getpaseo/paseo](https://github.com/getpaseo/paseo), desktop app,
  workspace "Paseo homepage"), Orca (desktop app, workspace "acme-web"), and
  a photographed screen of a third agent workbench the owner filed under
  t3code. All three are **observations of shipped UI**, not file-path
  receipts; nothing was re-cloned for this study. The in-house counterpart
  is `packages/pi-diff` (worktree `feat/pi-diff`): a `/diff` side panel for
  the Pi TUI that lists changes against `HEAD` with `+n −m` and follows the
  agent's last edit — the same vocabulary the rail uses.
- **Scope:** what a second sidebar beside the chat/terminal does in each,
  what PiCode adopts, where it improves on them, and what it refuses.

## Why now

Every current ADE puts Files / Changes / PR to the right of the
conversation. The Cursor bar already named the shape
([benchmark-cursor.md](../benchmark-cursor.md): "left nav + central work
area + right inspector — all resizable and collapsible; panels remember
state"), ADR-0074 adapted it *inside* the folder tab, and the diff-editor
roadmap parked "File tree as a sidebar tab" under *Later*. The owner asked
for parity with the benchmarks and for better decisions than theirs.

## What each one decided

| | Paseo | Orca | t3code (photo) |
|---|---|---|---|
| Tabs | text: `Files · Changes · PR #3981` | icons: Files, Search, Git, Tasks | text: `Changes · Files · PR` |
| Default tab | Changes | Files | Changes |
| Changes list | folder-grouped tree, `+N −M` per node and per file, status glyph, `Uncommitted ▾` filter with the total `+1.9k −684`, branch selector, collapsed `Commits` section | a separate Git tab | `0 changed · Stage all · Unstage all`, branch `main (unpublished)`, "No changes" |
| Click on a file | opens in the center | opens a center tab | opens in the center |
| Git writes | `Commit ▾` in the top bar | none visible | commit message box, `Commit` / `Commit & Push` |
| Link to the agent's turn | the turn's footer repeats `+1.9k −684` | — | — |
| Refresh | refresh + more menu | refresh + more menu | refresh |

Observations (inference marked *inf.*): all three anchor the rail to the
*workspace or session*, not to a repository picked from the URL (*inf.*);
none distinguishes what one agent changed from what another did; Paseo's
branch selector switches branches — a write — from the rail; the third
reference's stage/unstage/commit are one-click writes.

## What PiCode adopts

- A persistent right rail beside the center with **Changes** (default) and
  **Files**; content opens as center tabs (Paseo, Orca).
- Folder-grouped Changes with per-node `+N −M` sums, status glyphs, an
  `Uncommitted · +N −M` total and a count badge on the tab (Paseo).
- Refresh plus a "more" menu (Paseo, Orca); text tabs with a badge rather
  than an icon rail, since two tabs need no legend (Paseo, t3code).
- `PR #n` as a tab label — designed for a later phase, read through the
  host's `gh`, never a token in the UI (Paseo, t3code; ADR-0034).

## Where PiCode improves on them

| Benchmark decision | PiCode |
|---|---|
| The rail reads whatever folder the app knows | Reads through the **owner** (agent, terminal, workspace) with the folder as an equality precondition — the two identities of ADR-0022/0030/0074. A 409 becomes a blocked line with **Follow**, never a silent retarget |
| One diff for the whole worktree | A **This agent** scope: the working tree intersected with the paths the session's `edit`/`write` tools named. Several PiCode agents can share one folder; attribution is the question the benchmarks cannot answer |
| The rail is always present | Shrinks before it hides, hides when even its minimum width would push the conversation under 640 px, and remembers the viewer's choice through a squeeze |
| An empty rail on a folder without git | Files still works; Changes says "Not a git repository" with one action |
| Client-side polling | The fleet `git.updated` feed plus focus/visibility refetch; no interval (ADR-0048) |
| Branch selector, stage, commit in the rail | Refused for v1 — they are writes (ADR-0022/0032/0038); branch and worktree are shown as facts |

## What PiCode refuses (v1)

Branch switching, stage/unstage, commit/push, content search, a Tasks tab,
a Commits list, virtualization, fsnotify, mobile. PR and Commit are
designed in the Inspector ADR as later phases, pending the owner's
decision after the first dogfood.

## Ritual

The Inspector ADR cites this study. Later rail panels (PR, Preview,
Search, Tasks) extend the same `aside` — one rail, never two.
