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
| GET | `/api/snips/picker` live kind `prompt` (static path, not an id) |
| POST | `/api/snips` |
| GET/PATCH/DELETE | `/api/snips/{id}` |
| POST | `/api/snips/{id}/starred`, `/archived` |

`/expand` and `/run`: kind `shell` may be stored and runs through the
snippet-run door; the editor's Kind control offers Prompt and Command.

## Desktop studio

Host page `#/snippets` through `PageFrame` (1240px). Palette **Snippets**
and the user menu Tools group open it. Empty state: one line + New
snippet. Drafts sit in `sessionStorage` (`picode-snip-draft:<id>`). The
editor parses placeholders in the browser (`web/shared/domain/snipDraft.js`).
Command kind has no control yet.

### Authoring (v2)

The body stays the single source of truth; everything else is a
projection that writes back into it (`setDefaultInBody`).

| Surface | Behaviour |
|---|---|
| Live validation | parse on every keystroke; inline error naming the first problem; Save disabled **with visible reason**; table and Try-it dim to the last good parse |
| Placeholder table | one row per placeholder — Default / Optional / Enum (comma list, ≤32 × 80); reserved names lose the controls and say "from the target" |
| Try it | sample value per placeholder (enum → `<select>`), body expanded in the browser; required-but-empty names are listed as "will prompt when it runs" |
| Enums | held in editor state, sent in `placeholders[]`; they ride the draft alongside the body |

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
with `target.type=agent` expands with live `cwd` / workspace / agent name
and `SendTurn`s. `target.type=terminal` delivers through the **ADR-0089
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
panes) and **Run command…** (bare shells), each filtered to its kind.
Every attempt announces ephemeral `snip.ran` (ok + reason, never
values).
