# Composer files → MCP — implementation roadmap

- **Date:** 2026-08-26
- **Status:** plan. Auth / llama.cpp / Radius stay out until this track is done.
- **Why:** TUI can `@` files, paste images, and run `!cmd`. PiCode cannot.
  MCP is a status path. We close those gaps in this order.

Canonical pi: [Usage](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/usage.md)
(editor `@`, `!` / `!!`, images) ·
[RPC](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md)
(`prompt.images`, `bash`) ·
[architecture MCP](../architecture.md#mcp-model-context-protocol-support)
(`pi-mcp-adapter`, no native MCP).

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **A — Composer files** | now |
| 2 | **B — MCP manager** | A done **and** adapter write-format decided (table below) |
| 3 | Auth / local models | asked, after A+B |

Do not start Radius, OpenRouter PKCE, or llama.cpp installer in this track.

## Refuse (all tracks)

| Temptation | Why not |
|---|---|
| Dump file bodies into the prompt | tools `read`; blows context |
| Browser-local shell for `!` | not the agent cwd; forks pi |
| Native MCP in PiCode | pi has no native MCP; we orchestrate the adapter |
| MCP in the create-agent wizard | ADR-0009 |
| External editor (`Ctrl+G`) | the composer **is** the editor; TUI keeps `$VISUAL` |
| Images in `tasks.payload` | base64 in SQLite; send live on RPC |
| Invent adapter JSON | wait for the installed package's schema |

---

## Track A — Composer files

TUI editor features we owe. Scope is the **composer**, not Pin Studio
(pins already paste/drop).

### A1 — `@` file picker — **shipped**

**A1b — toolbar attach (out of plan, shipped):** clip button in the
composer toolbar opens a workspace browser (dirs + files, filter,
parent nav). Image → chip via `GET /api/agents/{id}/file` (base64,
inside-cwd only); anything else inserts `@path`. API:
`GET /api/agents/{id}/browse?dir=`.

Type `@` in the composer → fuzzy list of files under **this agent's cwd**.
Pick → insert `@rel/path` (space after). Same token the TUI leaves for
the model; the agent `read`s it.

| # | cwd | query | hit | action |
|---|---|---|---|---|
| 1 | missing / unreadable | * | * | no menu |
| 2 | ok | empty / `@` only | * | recent / top of tree, cap 20 |
| 3 | ok | `@foo` | files match | menu; Tab/Enter insert |
| 4 | ok | `@foo` | 0 | “No files” |
| 5 | ok | path escapes cwd (`../`) | * | refuse |
| 6 | ok | pick | image ext | insert `@path` (A2 may also attach) |

**API:** `GET /api/agents/{id}/files?q=` — walk cwd, skip `.git` /
`node_modules` / obvious junk, cap 200 scan / 20 hits. Do not reuse
`GET /api/fs` (dirs only, any path — folder picker).

**UI:** cmdk popover above the textarea (same family as `/`). Not a
full-page picker.

### A2 — Images on send — **shipped**

Paste or drop an image on the composer. Chip under the textarea.
Send → RPC `prompt` / `steer` / `follow_up` with `images: [{type, data, mimeType}]`
([rpc.md](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md)).

| # | agent | files | text | action |
|---|---|---|---|---|
| 1 | stopped | ≥1 | * | start, then RPC with images |
| 2 | running | ≥1 | empty ok | RPC now; **not** `EnqueueTask` |
| 3 | * | 0 | empty | no-op (today) |
| 4 | * | not image | * | ignore (A1 path token covers files) |
| 5 | * | >4 or >4 MB each | * | refuse + toast |

Reuse pin drop mime sniff. Optimistic: chip appears before upload.
Motion: chip + send stay on the composer (no job overlay).

`deliver()` today sends `{message: payload}` only
(`internal/rpc/runtime.go`). Images skip the task table.

### A3 — `!` shell — **shipped**

Presented as an inline conversation block (spinner → exit code, Stop
→ RPC `abort_bash`) instead of a modal overlay: the output **is**
conversation context, an overlay would hide it. `POST
/api/agents/{id}/bash` + `/bash/abort`; fake rpc double covers it in
tests.

Line is `!cmd` or starts with `! ` → RPC `bash` in the agent cwd.
Show output in the conversation (stream `bash_execution_update`).
Next **Send** includes it (pi already folds `BashExecutionMessage`
into the next prompt).

| # | input | agent | action |
|---|---|---|---|
| 1 | `!ls` | running | RPC `bash`; render output |
| 2 | `!ls` | stopped | start, then bash |
| 3 | `!!ls` | * | **not in A3** — RPC always attaches to the next prompt |
| 4 | `!` empty | * | no-op |
| 5 | user types `!` then more, hits Send as prompt | * | if the whole draft is `!…`, it is bash; mixed prose is a prompt |

**API:** `POST /api/agents/{id}/bash` `{command}` — not a new task kind
(CHECK on `tasks.kind` stays). Overlay while it runs (same job card as
packages). Abort → RPC `abort_bash`.

`!!` stays TUI until pi documents a silent bash. Do not exec outside pi.

### A4 — later, same track (small)

Only after A1–A3:

- `/export` HTML download (TUI default; GUI is JSONL today). `/share` already
  ships HTML+JSONL.
- `@` on an image path also attaches (A2).

Out of A: `/compact [prompt]`, `/tree` in-place, external editor.

---

## Track B — MCP visual manager

**Blocked** until the write format is a fact, not a guess.

Pi has no native MCP. The community package is `npm:pi-mcp-adapter`.
ADR-0009: `#/mcps` is not the create wizard. Today `GET /api/mcp` only
reports the first existing candidate path.

### Gate (must answer before B1 code)

Read the **installed** adapter (README + schema). Then fill:

| Question | Allowed answers | Forbidden |
|---|---|---|
| Which file do we write? | the path the adapter documents | a new `~/.picode/mcp.json` |
| Layers? | same as packages if the adapter has them; else **machine only** | invent per-agent MCP in SQLite |
| Shape? | adapter's JSON (stdio / url / env) | our own keys |
| Import? | only if the adapter already imports Cursor/Claude/Codex | re-encode foreign configs ourselves |

If the package is not installed: page says **Install `pi-mcp-adapter`**
(link to `#/packages`). Do not write configs for an absent adapter.

### B1 — list

Replace the path-only page. Show servers from the file we chose.
Empty: “No MCP servers” + Install adapter / Add.

### B2 — add / toggle / remove

Add stdio or URL. Toggle enabled. Remove. Write **that** file,
read-modify-write, unknown keys stay. Reload the agent so the
extension picks it up (`/reload` already restarts).

Motion: same job overlay on write+reload.

### B3 — import (only if gate says yes)

One button: import host configs **via the adapter**, not a PiCode parser.

### Decision table (runtime)

| # | adapter installed | config file | user action | result |
|---|---|---|---|---|
| 1 | no | * | open `#/mcps` | install CTA, no editor |
| 2 | yes | missing | open | empty list + Add |
| 3 | yes | present | toggle | rewrite file; `/reload` agent |
| 4 | yes | present | add invalid | 400, file untouched |
| 5 | * | * | create agent | unchanged (ADR-0009) |

---

## Track C — parked (discuss after A+B)

- Radius gateway login
- OpenRouter PKCE (GUI is API key)
- llama.cpp install / start / delete `.gguf`
- `/tree` `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645))
- `/compact [prompt]`

---

## Where it lives

| Thing | Path |
|---|---|
| This plan | `docs/design/composer-mcp-roadmap.md` |
| Composer | `web/src/components/Composer.jsx` |
| Send / RPC | `web/src/desktop/App.jsx`, `internal/rpc/runtime.go` |
| FS today | `internal/server/folders.go` (`/api/fs` dirs only) |
| MCP today | `internal/server/system.go` `handleMCP`, `web/src/components/Mcps.jsx` |
| Slash matrix | `docs/design/slash-parity.md` (unchanged by A/B) |
