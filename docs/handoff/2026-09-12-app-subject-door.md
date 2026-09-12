# 2026-09-12 — an app publishes its subject; the Inspector follows

`feat/app-subject-door` → `main` (71eae082). ADR-0109 amended a second time.

## Why it needed a door

Focusing a canvas panel left the rail empty. The obvious fix — the Canvas
setting `inspectorAnchor` — is the leak ADR-0109's 2026-09-11 amendment
forbids, and its door table already reads "the host owns … the Inspector
anchor". So the direction is inverted: `host.subject({kind, id})` states a
fact about the app's own content, `App.jsx` keeps `appSubjects[tabId]`, and
`anchorFor` reads it beside `gitOwners` / `treeOwners`. Nothing in
`components/canvas/` imports or names the rail; the boundary test is
untouched.

## Rules that keep it narrow

- Goes through `ownerExists`: an app cannot point the rail at a folder
  nobody owns.
- A note, file or diff panel publishes **nothing**, not null — the rail
  stays put instead of blanking.
- Cleared on unmount (tab closed), not on `hidden`: the host only reads the
  selected tab's subject, so coming back shows the folder you left.

## The trap

`host` is rebuilt inline on every `App.jsx` render, so `host.subject` is a
new function each time. Read it through a ref: as an effect dependency the
cleanup publishes `null` on renders that changed nothing here.

## Verified

Isolated instance, one terminal and one agent panel: rail follows
shell → Atlas → shell, each with its own folder in the head, Changes listing
this branch's diff. Note for the next QA: the rail does not fetch while
collapsed (`OPEN_BY_DEFAULT_FROM` is 1440), so at 1280 it sits on the
skeleton — open it before reading anything into that.
