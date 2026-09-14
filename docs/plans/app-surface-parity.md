# App surfaces in page parity — Docker, Inbox, tmux

- **Status:** proposal for owner review (2026-09-14). Nothing implemented yet.
- **Scope:** `web/browser` — the one bundle that renders both `/browser/` and
  `/desktop/` (ADR-0122, one boot path).
- **Out of scope:** the **Canvas** app (owner: "o app canvas é diferente e não
  faz parte do escopo"; native surface, ADR-0118) and `/mobile/`, which ships
  its own `AppSurface` copy (`web/mobile/src/components/AppSurface.jsx`) and is
  untouched by this branch.
- **Amends:** one sentence in `docs/benchmarks.md` (§ "One page width") that
  calls `#/app/*` a canvas. Owner's call — see §4.
- **Blast radius:** presentation only. No protocol, manifest, route, store or
  Go view-tree change; **no new ADR** (ADR-0086: a UI refinement is a
  paragraph in `docs/architecture/`, not an ADR).

## 1. What was asked

The five screenshots compare, on the same screen and the same window size:

| Screenshot | Surface | Route |
|---|---|---|
| `…143044` | Agent CLIs | `#/clis/pi` (route page) |
| `…143053` | Snippets | `#/snippets` (route page) |
| `…143117` | **Docker app** | `#/app/docker` (app tab) |
| `…143122` | **Inbox app** | `#/app/inbox` (app tab) |
| `…143126` | **tmux app** | `#/app/tmux` (app tab) |

Ask: the three apps read as a different product than the routes; bring their
layout, design and styles into parity with the system routes, on `/desktop`
and `/browser`, and improve the UX while doing it.

## 2. Why they differ (evidence, not impression)

The product has **two families** and the apps are in the second one:

| | Route pages (the standard) | Apps today |
|---|---|---|
| Frame | `PageFrame` → `.settings-wrap` (max **1240px**, centred, 28/24/40 gutters) + `.settings-head` (Back + `h2` 16px/650) + `.settings-card` (panel, 1px border, `--radius-panel`, 16/24/20) — `web/browser/src/components/PageFrame.jsx`, `app.css:3565-3577` | `.app-surface` full-bleed canvas: no page head, no card, rows start at the tab strip — `app.css:4463+` |
| Top navigation | underline nav inside the card: `.cli-tabs` / `.cli-pane-tabs` (2px accent underline, 16/12 padding, `agent-clis.css:2-7`) | segmented radio pills in a toolbar: `.app-tabs` (`app.css:4584+`) |
| Page actions | live in a card toolbar next to what they act on: `.auto-toolbar` (`app.css:4792`), `.mcp-empty` for zero items (`app.css:805`) | one toolbar row mixing pills + a 160px filter + `Refresh` + `Close` (`.ft-head`, `app.css:4083`) |
| Lists | inside the card: `.auto-list` rows, 8px gaps, 13px/600 names, per-row actions | `.app-row` on bare `--bg-base`, full window width |
| Split | card-contained grid: `.cli-layout` (200px catalog + detail) | full-width `.app-split` with its own border and 380px pane |
| Title weight | 16px/650 page title | 13px **mono** app title (reads as a pane label) |

The apps are the only **list/reader** surfaces in the canvas family; the rest
of that family (terminal, file tree, git graph, canvas) is a real canvas and
earns its full width.

## 3. What the screenshots actually show (UX findings)

Measured on the 2559px window in the captures:

1. **No page identity.** Docker/tmux/Inbox have no "where am I, how do I
   leave" line — the route pages open with `← Back · Title`.
2. **Related controls sit ~1300px apart.** Inbox: `Active | Done 5 | All 5` on
   the far left, `Refresh · Close` on the far right. tmux: `Filter tmux` (160px)
   and the list it filters are separated by empty space; Docker's group counts
   float at the right edge, ~1200px from their group titles.
3. **Empty states float in a well.** Inbox's "Nothing needs you right now."
   sits dead-centre in ~1100px of nothing. The bar wants one line + one action
   on a bounded surface, not a centred poster (§ benchmarks, "Empty states
   teach").
4. **One band, three roles.** Pills (navigation) + a filter (input) + page
   actions (buttons) share one toolbar row. Heights already agree
   (`--ctl-h`), roles do not.
5. **No boundary.** Rows start immediately under the tab strip and run to the
   window edge, so the app reads as a bare list, not as a PiCode page.
6. **Refresh is redundant.** Both Docker and tmux already refetch from the
   change feed (`docker.changed`, terminal events) plus focus/visibility
   (`AppSurface` — ADR-0048); the route pages carry no Refresh button at all.

## 4. The documented decision this changes (owner's call)

`docs/benchmarks.md`, "One page width": *"Workspace surfaces (`#/`, `#/term/*`,
`#/file/*`, `#/git/*`, `#/tree/*`, `#/app/*`) are canvases, not page frames —
they keep their own layout."*

**Cost of changing it:** the three apps lose the full window width. Docker and
tmux lists, and Docker's log output, get 1240px instead of 2559px on this
screen; a log line wraps or scrolls inside the card. Mitigations: the split
panes stay resizable inside the card, log output keeps horizontal scroll, and
`.app-body > .app-block` already caps prose at 720px.

**What stays a canvas:** `#/`, `#/term/*`, `#/file/*`, `#/git/*`, `#/tree/*`
and the **Canvas app** — the sentence narrows to name the app ids it applies
to (or drops `#/app/*` and names the three apps that became pages).

**Recommendation:** amend it. A container list, a needs-you queue and a tmux
inventory are pages: they are read top-to-bottom and they have a page's
actions. ADR-0133 already rejected a second UI standard for tmux once; the
primitives chrome is that second standard, just host-owned instead.

## 5. Parity spec — element by element

The target is one geometry for app pages, owned by the **host** (chrome stays
host-owned: ADR-0036, ADR-0109's door table = "the app's own body, inside that
tab"). No app gains a new door.

| Element | Target (the route standard) |
|---|---|
| Page frame | `.settings-wrap` 1240px + `.settings-head` + `.settings-card`, exactly `PageFrame`'s geometry |
| Page head | `h2` 16px/650 + the app's own mark (`manifest.icon`); the head action is `Close` (the tab's exit, as on every other tab surface) — see **D1** |
| Tabs | underline nav inside the card, the `.cli-pane-tabs` look; badge becomes the quiet count chip (`.app-sect-count` idiom), never a filled pill |
| Filter | full-width search row inside the card (`.snip-search` model), not a 160px box in a toolbar |
| Page actions | a card toolbar (`.auto-toolbar` model): count/state on the left, the primary action on the right; `Refresh` only where the feed truly cannot cover it |
| List rows | inside the card, the `.auto-list` rhythm (8px gaps, 13px/600 names, quiet meta), keeping the host's row primitives — hover/keyboard row actions, unread dot, swipe on touch |
| Groups | keep the disclosure group (ADR-0066), restyled to the card: `.settings-section` heading rhythm, count as a chip beside the title, not at the window edge |
| Split | card-contained (`.cli-layout` model): list pane + detail inside the card, resizable to 300–640px, stacking ≤880px as today |
| Loading | the page skeleton (`skel-line`), matching the real layout |
| Empty | `.mcp-empty` shape: one line + one action, inside the card, no centred poster |
| Error | one line + `Try again`, inside the card, same copy as the routes |
| Detail view | the page head stays; detail content is a card section, not a full-bleed pane |

## 6. Plan

**P1 — Geometry (the visible fix).** `AppSurface` renders through a page frame
(`.app-page`: wrap + head + card) instead of `.app-surface`'s full-bleed
canvas. Both `paneMode` branches (desktop split / phone list+detail) keep their
current data flow; the phone is untouched because it has its own copy.

**P2 — In-card navigation.** Tabs → underline nav; filter → card search row;
`Refresh`/`Close` → card toolbar. `AppSurface`'s header loses `.ft-head`
(`FileTreeSurface` keeps it; that surface is still a canvas).

**P3 — Content rhythm.** Rows, groups, sections, counts and the detail pane
adopt the card's spacing and typography. Copy stays the app's (`view.Title`,
`Block.Title`, `ListItem`) — the host owns how it is drawn, not what it says.

**P4 — States.** Empty, blocked (unknown API version / app needs a newer
PiCode), error and loading are re-cut to the page standard.

**P5 — Verification and docs.** Screenshots of all three apps in
empty/blocked/error on **both** `/browser/` and `/desktop/`; `make web`,
`make test-js`, `make ci-scoped`; uiux-review + visual-review verdicts;
architecture notes (`docs/architecture/docker-app.md`, `tmux-app.md`, and the
host paragraph in `routes.md`); benchmark sentence amended; changelog fragment;
handoff note.

## 7. Decision table (conditions → outcome)

| Condition | Outcome | New test |
|---|---|---|
| App tab selected | page frame with the app's head and card | visual |
| App tab hidden | stays mounted, `hidden`, no reload, no layout shift | existing |
| Deep link `#/app/<id>/<path>` | detail opens inside the card; the page head stays | manual |
| Unknown manifest version / native surface missing | blocked card: one line + why, no rows | existing copy, new screenshot |
| Empty view (`view.Empty` or zero blocks) | `.mcp-empty` inside the card | new |
| Load error | one line + `Try again` inside the card | new |
| Narrow ≤880px | split stacks, card keeps its gutters, tabs scroll horizontally | new |
| Split resized by the reader | width pref (300–640px) preserved inside the card | existing |
| `/browser/` and `/desktop/` | identical (one boot path) | screenshot pair |
| `/mobile/` | unchanged — own component, not this branch | n/a |
| Canvas / native apps | unchanged | n/a |

## 8. Risks

- **Width loss** on wide screens (the §4 cost) — the one thing to look at in
  the owner's review screenshots.
- **Two CSS families touching** (`.ft-head` is shared with `FileTreeSurface`
  and `NativeDemoSurface`): the app frame must add its own classes, never
  restyle `.ft-*` in place.
- **Worktree overlap:** `feat/desktop-entry-fix` touches `web/desktop` +
  `bootstrap.jsx`; `feat/tmux-orphans` touches `internal/tmux`. This branch
  touches `web/browser/src/components/AppSurface.jsx` + `app.css` — low
  overlap, but `main` may move before the merge.

## 9. Open decisions for the owner

- **D1 — the head action.** `Close` only (tab semantics, recommended) ·
  `← Apps` route-style Back (link to the Apps grid) · both.
- **D2 — tabs.** Underline nav inside the card (route parity, recommended) or
  keep the segmented pills (the owner's earlier call when they sat in the
  toolbar — `app.css:4584`).
- **D3 — the benchmark sentence** in §4: amend as proposed, or leave it and
  record this as an exception.
- **D4 — Refresh.** Drop it (feed-driven, recommended) or keep it in the card
  toolbar.
