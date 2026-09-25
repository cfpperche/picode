# Snippets (ADR-0130)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

User-owned reusable templates (`prompt` | `shell`) with named placeholders.
PiCode expands the body **before** send. Delivery reuses
`POST /api/agents/{id}/prompt`, the ADR-0089 CLI prompt door, and a
**snippet-run door** (kind `shell`, live `isShell` pane: `ClearLine` +
`PasteText` + Enter, confirm in the official UI). Kind `prompt` into a
plain shell is 409 `kind` in v1. Host routes `#/snippets` — **not** an
app (ADR-0109). Plan: [docs/plans/snippets.md](../plans/snippets.md).

**Name collision.** `internal/snippet` runs a conversation fence
(`POST /api/agents/{id}/snippet`, play button **Run** on `SourceBlock.jsx`).
This subsystem is **`internal/snips`**, HTTP `/api/snips`, table `snips`,
events `snip.*`. Never reuse the fence package.

## Parser (`internal/snips`)

One-pass, stdlib only. Not a template engine.

| Token | Meaning |
|---|---|
| `{{name}}` | required placeholder (`[A-Za-z_][A-Za-z0-9_]*`, 1–40) |
| `{{name=default}}` | optional; default trimmed; may contain a single `}` |
| `{{{{` / `}}}}` | literal `{{` / `}}` (Expand only) |

- **`Parse`** collects unique names (first-wins default). It does **not**
  unescape. Escaped regions are skipped when collecting names. The stored
  body keeps `{{{{`. Invalid / unclosed / too many names → error.
- **`Expand`** is the only unescape. Values are not re-scanned. Reserved
  names (`cwd`, `workspace`, `branch`, `date`, `agent`, `cli`) come from
  `ctx`, never from `values`. Empty `values[name]` is missing unless a
  default exists (including empty `{{n=}}`). `{{date}}` without `ctx`
  uses UTC `YYYY-MM-DD` in-process. `` !` `` is ordinary text.
- **`Slug`** hyphenates like `store.newID` (`"Review PR"` → `review-pr`),
  not `slashres.sanitizeName` (which strips to `reviewpr`).

Limits refuse at save (HTTP, later): 32 placeholders, 500-rune defaults,
32 enum values × 80 chars (enums live on the stored schema, not in the
body).

## Store and HTTP (CRUD)

Table `snips` (migration 047). Unique `slug` covers archived rows; reuse
requires DELETE. Mutations append `snip.created` / `snip.updated` /
`snip.deleted` with a **summary** (no body). Starring and archiving do
not bump `updated_at`.

| Method | Path |
|---|---|
| GET | `/api/snips` live list; `?archived=1`; `?q=` search |
| GET | `/api/snips/templates` built-in starters — code, not rows (`internal/snips/templates.go`, the `automate.Templates` pattern); static paths before `/{id}` |
| GET | `/api/snips/slug/{slug}` the editor's address check — 204 free, 409 taken; `?except=<id>` is the snippet being edited, whose own slug is not a clash |
| GET | `/api/snips/picker` live kind `prompt` (static path, not an id) |
| POST | `/api/snips` |
| GET/PATCH/DELETE | `/api/snips/{id}` |
| POST | `/api/snips/{id}/starred`, `/archived` |

`/expand` and `/run`: kind `shell` may be stored and runs through the
snippet-run door; the editor's Kind control offers Prompt and Command.

## Desktop studio

Host page `#/snippets` through `PageFrame` (1240px). Palette **Snippets**
and the user menu Tools group open it. Empty state: one line + New
snippet. Drafts sit in **`localStorage`** (`picode-snip-draft:<id>`, the
same keys and `draftToRestore` base rule as before — a closed tab no
longer takes the text; last-writer-wins across tabs, like the Automations
draft). The editor parses placeholders in the browser
(`web/shared/domain/snipDraft.js`). The Kind control (Prompt / Command)
is a `<select>` on both surfaces.

### Authoring (v2)

The body stays the single source of truth; everything else is a
projection that writes back into it (`setDefaultInBody`).

| Surface | Behaviour |
|---|---|
| Live validation | parse on every keystroke; inline error naming the first problem; Save disabled **with visible reason**; table and Try-it dim to the last good parse |
| Placeholder table | one row per placeholder — Default / Optional / Enum (comma list, ≤32 × 80); reserved names lose the controls and say "from the target" |
| Try it | sample value per placeholder (enum → `<select>`), body expanded in the browser; required-but-empty names are listed as "will prompt when it runs" |
| Starters | `GET /api/snips/templates` — six starters; the grid is open on an empty page and a remembered `<details>` once rows exist (Automations pattern). Clicking one hands the body, tags, kind and a derived slug to the editor as a draft with origin `starter` |
| Duplicate | a row action and a detail button: reads the snippet (a row carries no body), then hands `title + " copy"`, slug `<slug>-copy` (locked) and the body over as origin `duplicate` |
| Save off | what explains it is written where the problem is — the body's own line and the slug's own line — never a badge beside the button (a text span in a `data-align-row` row is the alignment defect `__picodeOverlayAudit` catches) |
| Enums | held in editor state, sent in `placeholders[]`; they ride the draft alongside the body |

### Phone editor (v2)

The list is the studio's list: **Active** / **Archived** (the desktop's
`?archived=1` view), the same search over title, tags and body, and the row
that says `archived` in its subtitle. The detail screen owns the two
actions the desk has — **Archive** / **Unarchive** and **Delete** (a
confirm naming the snippet; deletion navigates back) — so archiving is
reversible from the phone. Without that view the phone could archive a
snippet with no way to see it again, which is why the view and the action
shipped together.

`SnippetEdit` (`#/snippets/new`, `#/snippets/{id}/edit`) carries the same
rules as the studio — parse on every keystroke, the address check against
`/api/snips/slug/{slug}` (with `?except=` when editing), enums in
`placeholders[]`, drafts under the same keys — with two phone decisions:

| Difference | Why |
|---|---|
| The table is **a card per placeholder**, not a `<table>` | a four-column table at 390px scrolls sideways or squeezes the name; the name, its default and its choices are read in one look |
| A required placeholder shows a **disabled** default field reading "Required — the agent asks" | the default means nothing for a required name (it is not written as `{{name=…}}`); the sentence is inside the field, so a dimmed control is never a mystery |

A reserved name (`cwd`, `workspace`, `branch`, `date`, `agent`, `cli`)
shows no controls and says "Filled from the target — nothing to set here."
The table dims to the last good parse while the body is invalid, exactly
as the studio does.

### Capture and import (v2)

Text the reader already wrote becomes a snippet without a trip to the
studio. Both doors open the **same editor**:

| Door | Trigger | Payload |
|---|---|---|
| Composer toolbar | a selection exists in the composer (`onCaptureSnippet`) | selection verbatim |
| Context menu | "Save selection as snippet" on any selection — a field's own selection is read from the field, since it is not in the document selection | selection verbatim |
| Mobile composer | **Save as snippet** in Message options | the whole draft |
| Studio **Import** | paste a prompt; `detectConversions` suggests `[BRACKETS]` / `UPPER_CASE` → `{{lower_snake}}` (never inside an existing `{{…}}`, never double-counting a token) and each suggestion can be switched off | converted body |

Desktop capture opens `SnipCaptureSheet` — the editor in a
`ResponsiveDialog` with `prefill` and `keepDraft: false`, so the studio's
"new snippet" draft is neither read nor clobbered. Import (desktop and
mobile) hands the body over through that draft and marks it:
`writeDraft(store, id, draft, base, origin)` records `capture` / `import`,
and the mobile editor shows its "Unsaved changes restored." banner only
for a draft with **no** origin — a handoff is not a crash.

## Expand and run

`POST /api/snips/{id}/expand` is pure substitution (no git). Missing
required names are listed; the body is still returned. `POST /api/snips/{id}/run`
with `target.type=agent` expands with live `cwd` / workspace / agent name.
Managed agents `SendTurn`. Interactive agents take ADR-0130 door 5:
`deliverToInteractiveAgent` (receiver-or-paste into the agent's terminal session, source
`"snippet"`, proof is tmux/receiver accept). Stopped is 409 `stopped`.
`target.type=terminal` delivers through the **ADR-0089
door** (shared `pasteToTerminal`): `termHoldsCLI` must hold, and the
pane's foreground is re-checked at handler time — a shell (the TUI
exited, no live wrapper lease) answers 409 `kind`, because a prompt body
pasted into bash is the defect the owner refused.

Kind `shell` is the **snippet-run door**: terminal target only,
`confirm: true` required (a UI invariant on official clients, not authz
— K14), a live CLI lease refuses (`reason: "cli"`), the pane foreground
must be a shell (`reason: "foreground"` otherwise), ClearLine then a
bracketed paste whose Enter submits. `preview: true` expands with live
gates and context but delivers nothing — the confirm step shows exactly
what would run. Composer `/snip:slug` opens a fill sheet (**Insert
snippet** splices the draft; **Send snippet** calls `fireSend` with the
spliced text so images stay). Palette **Send snippet** posts `/run`. The
desktop terminal right-click menu gains **Send to terminal…** (CLI
panes and interactive agent TUIs) and **Run command…** (bare shells), each filtered to its kind.
A pane invoke keeps `onlyKind: "prompt"` so Command snippets stay off
that picker; palette Send snippet stays unfiltered and uses `via: "tui"`
when the selected agent is interactive (the sheet then says “Sent to the
terminal.”).
Every attempt announces ephemeral `snip.ran` (ok + reason, never
values). That notice is published once the route has identified a snippet,
so a request whose id does not exist (or whose body is malformed) is not a
run and stays out of the feed, while a snippet that ran and failed — a
target that is gone, say — is published with `ok: false`
(`TestSnipRanFeedBoundary` pins both rows).

The shell door also consults `repoBusy` before typing, the rule **Run
command…** in a terminal already follows: if a PiCode agent or another
terminal is working in the repository the target pane sits in, the run is
refused with 409 `reason: "busy"` naming who. The snippet text is
arbitrary, so the state of the repository is what decides, not a
per-command guess. `preview: true` is never blocked — it types nothing, and
the confirm step has to be reachable.

Snips requests are read through `snipDecodeLimit` (256 KB): the store
refuses a body over 100 KB, but without a cap the server buffers an
arbitrarily large request before validation can say so.
