# Study: the actions on a workspace card (VS Code, PatternFly, Carbon, WCAG, GitHub Desktop)

- **Date:** 2026-09-07
- **Sources:** public docs, cited per fact below. Nothing was cloned;
  inferences are marked *inf.* The in-house counterpart is the
  `.ws-group-head` block of `web/desktop/src/components/Sidebar.jsx` with
  `.ws-group-actions` / `.ws-context` in `web/desktop/src/styles/app.css`.
- **Scope:** how many actions a list row may carry, what may hide until
  hover, and where a repository's history is entered from. What PiCode
  adopts; what it refuses.

## Why now

The owner's screenshot (2026-09-07) shows a workspace card with five
icon-only actions and the line *"Empty — add an agent or a terminal."*,
and reports the pain plainly: **there is no way to reach the git graph of a
workspace that has no agents and no terminals.** The cause was in the
browser, not the server:

| Fact | Value |
|---|---|
| Git graph entry points before this change | `ContextLine` only, rendered by `AgentRow` and `TermRow` — never by the card header |
| `GET /api/workspaces/{id}/git` | shipped with ADR-0030 (`internal/server/gitgraph.go:69`), never called by desktop |
| `workspaceView.Git` in `GET /api/workspaces` | branch + dirty count, already on every sidebar poll (`internal/server/workspaces.go:105`) |
| Mobile | reaches the same routes since ADR-0095 (`web/mobile/src/lib/git/model.js`) — desktop was behind the phone |
| Inline actions on the card header | 5 (Files, new agent, new terminal, Sessions, Remove) |
| Their resting opacity | `0` — the whole strip appeared on hover or focus-within |
| Sessions on a workspace with no agents | 409 `workspace has no agents — add one first` (`internal/server/sessions.go:277`) |

## What the field does

1. **Three inline actions, then an overflow.** VS Code's UX guidelines tell
   extension authors not to add more than three actions to an item and warn
   that a crowded view toolbar is "noise and confusion"
   ([code.visualstudio.com/api/ux-guidelines/views]). PatternFly's overflow
   menu guidance is the same number from the other side: don't fully display
   more than three actions in a toolbar, and don't use an overflow menu for
   two or fewer ([patternfly.org]). Carbon caps a menu at ~12 items and
   accepts an ellipsis that appears on row hover ([carbondesignsystem.com]).
2. **Hover may reveal, but not gate.** WCAG 2.1 SC 1.4.13 and SC 2.1.1 put
   the line at reachability: content that appears on hover must also appear
   on focus, and functionality reachable only by pointer fails outright
   ([w3.org/WAI]). The practitioner reading is blunter — do not hide the
   controls a user needs to proceed behind a hover, because touch, keyboard
   and low-vision users never find them ([boia.org], [accessitree.com]).
3. **The branch is the door to the history.** VS Code puts branch, dirty
   state and ahead/behind in the status bar as a permanent, clickable
   target; GitHub Desktop enters History from the same region that names the
   branch ([code.visualstudio.com/docs/editor/versioncontrol],
   [docs.github.com]). Neither hides that behind an icon labelled "git".

## What PiCode adopts

- **Two controls on the header, both visible at rest.** One **New** menu
  (agent · shell terminal · Agent CLI terminal — the card had two creation
  affordances with different shapes) and one overflow holding Files, Git
  graph, Sessions and Remove. Five became two, and the destructive action
  stopped sitting one pixel from the creative one. The triggers hold a fixed
  right-hand gutter so the collapsed face strip and the name end before them.
- **The workspace is an owner of its own files and history.** `Files` and
  `Git graph` on the menu read through the workspace (`#/tree/w/<id>`,
  `#/git/w/<id>`), which is the fix for the reported pain.
  *Shipped and then withdrawn the same day:* the card first carried the
  folder/branch pills of `wsLine(ws)` as a second line, the way its rows do.
  Live on a real fleet the owner read it as noise — every agent row under
  the card repeats the same path and branch — and asked for the header to
  stay one line, the menu carrying the actions. `wsLine` remains: the menu
  asks it whether the folder is a repository at all.
- **The menu tells the truth about an empty workspace.** No Git graph item
  on a folder that is not a repository; no Sessions item while the workspace
  has no agents, because that route answers 409.
- **The empty state acts.** "Empty — *add an agent* or *a terminal*" carries
  the two buttons the Agents and Terminals tabs already offer, which is what
  ADR-0027 meant by "the empty state's actions are the way in".

## What PiCode refuses

- A dedicated git icon button on the header. Two visible controls is the
  budget; the third entry point belongs in the menu.
- Hiding the triggers again to buy back density: the resting `.55` opacity
  is the compromise, and hover, focus and an open menu take it to full.
- A second identity for the graph tab. A workspace-owned graph collapses
  onto the same `g:<repo key>` tab as its agents' (ADR-0022): the owner
  authorises the read, the repository names the tab.

[code.visualstudio.com/api/ux-guidelines/views]: https://code.visualstudio.com/api/ux-guidelines/views
[patternfly.org]: https://www.patternfly.org/components/overflow-menu/design-guidelines/
[carbondesignsystem.com]: https://carbondesignsystem.com/components/menu/usage/
[w3.org/WAI]: https://www.w3.org/WAI/WCAG21/Understanding/content-on-hover-or-focus.html
[boia.org]: https://www.boia.org/blog/hover-actions-and-accessibility-addressing-a-common-wcag-violation
[accessitree.com]: https://www.accessitree.com/wcag-ultimate-guide/ensure-controls-needed-to-progress-are-visible-without-hover-focus/
[code.visualstudio.com/docs/editor/versioncontrol]: https://code.visualstudio.com/docs/editor/versioncontrol
[docs.github.com]: https://docs.github.com/en/desktop/making-changes-in-a-branch/viewing-the-branch-history-in-github-desktop
