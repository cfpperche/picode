# Snippets v2 — the creation experience

Owner call, 2026-09-13: "péssima UX" na criação. This plan fixes the
**authoring** side (v1 shipped the doors; the editor stayed raw).
Design source for v1: [snippets.md](snippets.md); ADR-0130 unchanged —
v2 is editor/UX, no protocol or persistence boundary except two small
additions named below.

## Friction inventory (audited live, 2026-09-13)

Walked the creation flow on a scratch instance; findings with evidence:

| # | Friction | Evidence |
|---|---|---|
| F1 | **Validation only at Save.** With `{{1bad}}` / unclosed braces in the body, no error shows while typing, chips silently vanish, and **Save stays enabled** — the problem surfaces only after the click. | DOM: `.form-error` empty, `Save` enabled with invalid body |
| F2 | **Placeholder chips are read-only output.** You cannot set a default, mark optional, or add an enum from the UI — enums are API-only. | editor renders `.snip-chip` text only |
| F3 | **No capture path.** Prompts are born in the composer, but saving one forces a context switch to `#/snippets` and retyping. | no selection→snippet affordance |
| F4 | **The draft dies with the tab.** Drafts live in `sessionStorage`; a crash or closed tab loses the work. | inherited Pins pattern |
| F5 | **No try-it.** You cannot see the expanded result with real-ish values from the editor; the preview leaves required slots empty. | preview shows "…:  in " |
| F6 | **No duplicate, no starters.** Cloning a near-miss means retyping; the empty page has no examples (Automations has a template grid — parity gap). | empty state = one line + New |
| F7 | **No import / placeholder detection.** Pasting a prompt from another tool keeps its `[BRACKETS]`/`UPPER_CASE` conventions; nothing suggests conversion. | — |
| F8 | **Slug collisions surface at Save** (409 round trip). | save-time error only |

## Benchmark study

Creation patterns from the tools that do this best (receipts are official
docs; claims kept to documented behavior):

| Product | Creation pattern | PiCode v2 takes |
|---|---|---|
| **JetBrains Live Templates** ([template variables](https://www.jetbrains.com/help/idea/template-variables.html)) — the canonical placeholder editor | Body stays in the editor; an **Edit Variables** table lists each variable with name, default value, skip-if-defined — rows are a projection of the body | **F2**: the placeholder table (below). Body keeps single-source-of-truth; rows write back into it |
| **Raycast Quicklinks** ([docs](https://www.raycast.com/blog/quicklinks), `{argument}` tokens auto-listed as fields) | Autodetects tokens in the link and offers them as form fields | **F7**: paste-detection suggests `[BRACKETS]` / `UPPER_CASE` → `{{lower_case}}` conversion |
| **Warp Workflows** ([YAML workflows](https://docs.warp.dev/terminal/entry/yaml-workflows/), AI can draft title/args) | Form + arg table; AI authoring | **F6**: starter gallery now; AI drafting later through the focused agent (not v2 scope) |
| **Postman environments** ([learning.postman.com](https://learning.postman.com/docs/sending-requests/variables/)) | Variable table with values; immediate usage preview | **F5**: Try-it pane — fill values → expanded preview via the pure `/expand` |
| **TextExpander / Alfred snippets** (fill-in fields, create from selection) | Capture what you already wrote | **F3**: capture from composer selection; duplicate action |
| **Our Automations (ADR-0045)** | `GET /api/automations/templates` + Suggested grid + read-once draft slot | **F6/F3**: same three patterns, snippets edition |

Refused: rich-text body editors (WYSIWYG), folders/spaces, a second
template language — the body is one plain text with one grammar
(ADR-0130 K5).

## Design

### 1. Live validation (F1)

`parseSnip` already runs on every keystroke in the browser (pure,
`snipDraft.js`). The editor stops hiding it:

- inline error line under the body the moment the parse fails, naming
  the first problem ("Close every {{placeholder}}.", "Invalid
  placeholder name at …");
- Save disabled **with the reason visible** (house rule: a disabled
  control explains itself in text, not a tooltip);
- chips dim (not vanish) while the body is invalid, keeping the last
  good parse visible.

### 2. Placeholder table (F2)

Under the body, one row per parsed placeholder — a projection that
writes back into the body text:

| Column | Control | Writes |
|---|---|---|
| Name | read-only text (reserved names get a "from the target" badge) | — |
| Default | input; empty by default | `{{name=default}}` in **every** occurrence-safe way: only the first occurrence gains `=default` (first-wins parse) |
| Optional | toggle (on ⇔ `=…` present or empty default) | `{{name}}` ⇔ `{{name=}}` |
| Enum | comma input (≤32 × 80 chars) | `placeholders[].enum` on save (server-validated; runs render a `<select>`) |

Editing rules: the table never reorders or renames (rename = edit the
body); saving merges enums/defaults the server already knows
(`mergeSnipPlaceholders` semantics, unchanged). This makes enums a
first-class UI citizen for the first time.

### 3. Try-it (F5)

A collapsed **Test** section in the editor: the run sheet's value form
(reused component) + expanded preview via `POST /expand` (pure, no
door). Defaults pre-filled. This is the dress rehearsal without a
target.

### 4. Capture from the composer (F3)

Select text in the agent composer → **Save as snippet** (toolbar row +
selection context action): opens the editor sheet pre-filled with
`body = selection`, `title = first words`. Images are not carried.
Mobile: same via selection action. The studio also gains **Import**:
paste a prompt → detection suggests `[BRACKETS]` / `UPPER_CASE`
conversions (client-only — the parser already lives in
`snipDraft.js`; no new endpoint).

### 5. Starters + duplicate (F6)

`GET /api/snips/templates` — code list like `automate.Templates()`:
six starters (Review a PR, Explain an error, Write a commit message,
Summarize uncommitted changes, Run tests and fix, Standup update).
Empty state grows the Suggested grid (Automations pattern); list rows
and detail gain **Duplicate** (opens the editor pre-filled, `-copy`
slug).

### 6. Crash-safe drafts (F4)

Drafts move `sessionStorage` → `localStorage` (same keys, same
`draftToRestore` base semantics). Cost accepted: last-writer-wins
across tabs, like the Automations draft.

### 7. Slug availability (F8)

`GET /api/snips/slug/{slug}` → 204 free / 409 taken. Debounced check
as you type; inline "already used" before Save.

### 8. Kind copy (small)

Kind becomes a segmented control (Automations pattern) with one honest
line per door: "Sent to an agent or a CLI terminal" / "Pasted into a
shell — always confirmed".

## Decision table (authoring)

| Condition | Action |
|---|---|
| Body parse fails while typing | inline error; Save disabled with reason; chips dim |
| Default edited in the table | body rewritten (`{{name}}` → `{{name=value}}`); preview updates |
| Required toggled off, no default | body rewritten to `{{name=}}` |
| Enum edited | held in editor state; sent in `placeholders[]` on Save |
| Reserved name row | default/optional/enum controls hidden; badge explains |
| Paste with `[X]` / `UPPER` | suggestion chips; apply rewrites the body |
| Slug taken (debounced check) | inline 409 text; Save disabled |
| Draft (localStorage) vs server moved | existing `draftToRestore` base rule |
| Capture from selection | body = selection verbatim; title = first words; no images |
| Starter picked | editor pre-filled; slug from starter slug + `-copy` |

## Non-goals (v2)

AI drafting (Warp-style) — later, through the focused agent;
marketplace/sharing; folders; rich-text body; simultaneous multi-tab
draft merging.

## PRs (one branch `feat/snippets-v2`, stacked)

| # | Title | Notes |
|---|---|---|
| 1 | `snips: live validation in the editor` | F1 + tests on `snipDraft` helpers |
| 2 | `snips: placeholder table` | F2, default/optional/enum write-back + tests |
| 3 | `snips: try-it pane` | F5, reuses run-sheet form + `/expand` |
| 4 | `snips: capture from selection and import` | F3/F7, both apps |
| 5 | `snips: starters and duplicate` | F6, templates endpoint + grid |
| 6 | `snips: slug check and durable drafts` | F8/F4 |
| 7 | `snips: mobile editor v2` | table + capture on the phone |
| 8 | `docs: snippets v2 guide update` | docs-site + changelog |

Each PR keeps `make ci-scoped` green; visual review per PR (pixel gate
owed once a harness reads images — recorded debt).
