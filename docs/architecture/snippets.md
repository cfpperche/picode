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

`/expand` and `/run` are later commits. Kind `shell` may be stored; the
editor hides it until the snippet-run door lands.

## Desktop studio

Host page `#/snippets` through `PageFrame` (1240px). Palette **Snippets**
and the user menu Tools group open it. Empty state: one line + New
snippet. Drafts sit in `sessionStorage` (`picode-snip-draft:<id>`). The
editor parses placeholders in the browser (`web/shared/domain/snipDraft.js`)
and shows a local preview (no git). Command kind has no control yet.
