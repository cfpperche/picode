# ADR-0130: PiCode Snippets — GUI library, expand-before-send, snippet-run door

- **Status**: accepted (owner, 2026-09-13 — Q1–Q6 confirmed in the design session)
- **Date**: 2026-09-13
- **Boundary**: persistence (SQLite `snips` overlay), protocol (`/api/snips` CRUD + `/expand` + `/run`), security model (expand is pure string replace; kind `shell` PasteText+Enter is a new user-initiated door, not Attach)
- **Does not change**: ADR-0089 (kind `prompt` into a plain shell is 409 `kind` in v1; Attach is the file+caption door — interactive agents gained a sibling drop/prompt in the 2026-09-14 amendment of 0089), ADR-0069 (guest CLIs are not managed agents), ADR-0109 (host surface, not an app), ADR-0003 (no vendor SDKs)
- **Plan**: [docs/plans/snippets.md](../plans/snippets.md)

## Context

Users repeat the same prompts and the same shell commands across managed Pi agents, Agent CLI terminals (ADR-0069) and ordinary shells. The only reusable prompt path today is **Pi's files** (`~/.pi/agent/prompts/*.md`), which the composer **inserts as `/name `** and which **pi expands**. Guest CLIs never see those files. `internal/snippet` already means "run a conversation fence" (`POST /api/agents/{id}/snippet`) and must not be reused.

Benchmarks (Cursor Commands/Skills, Claude Code slash commands, Copilot prompt files, Warp Workflows, Pi templates) converge on named on-demand templates. File-only stores fail PiCode's terminal-averse audience. Claude/OpenCode `` !`bash` `` interpolation is live command execution at expand time — an ADE that expands on behalf of every CLI would run that as the daemon user.

ADR-0089's prompt door pastes+Enter into Agent CLI TUIs only ("plain shells wait"). `tmux.Manager.PasteText` always presses Enter. Inspector `type`/`run` already types into plain shells **without** Enter, gated by `isShell`.

Owner calls, 2026-09-13: name Snippets; SQLite-only in v1; `{{name}}` placeholders; no v1 export to `~/.pi/agent/prompts/`; PageFrame 1240px; shell snippets may PasteText+Enter into a live `isShell` pane; prompt snippets must not.

## Decision

PiCode owns a **Snippets** library: one body, two kinds (`prompt` | `shell`), named placeholders, GUI CRUD on host routes `#/snippets` and `#/snippets/:id`. Go package **`internal/snips`**. HTTP **`/api/snips`**. Events **`snip.*`**. Table **`snips`**. The fence runner `internal/snippet` is untouched.

Expand happens **inside PiCode** before send (pure one-pass substitution). Guest CLIs receive already-expanded text. No `` !`cmd` ``, no `@file` includes, no Handlebars, no eval. Placeholder syntax is `{{name}}` / `{{name=default}}` with `{{{{` / `}}}}` escapes so a body may teach literal braces. Values are not re-scanned. Reserved names (`cwd`, `workspace`, `branch`, `date`, `agent`, `cli`) are filled from the invoke target, never from client `values`.

Delivery reuses existing doors where they already fit, and adds one:

1. Managed agent, composer **Send snippet** — existing `POST /api/agents/{id}/prompt` with the spliced draft (images kept).
2. Managed agent, palette **Send snippet** — `POST /api/snips/{id}/run` → `SendTurn` (snippet text only, no images).
3. Agent CLI TUI, kind `prompt`, `termHoldsCLI && !isShell` — existing ADR-0089 `POST /api/terminals/{id}/prompt`.
4. Kind `shell`, live `isShell` pane — **snippet-run door**: `ClearLine` + `PasteText` + Enter. Official UI always confirms (names the terminal, shows the expanded command, states that PiCode does not shell-quote). `confirm: true` is a UI invariant, not authz; a paired API client may set the flag. Re-check `isShell` at handler time. Inspector `refuseInspectorCLI` is unchanged.
5. Interactive agent, kind `prompt` — receiver-or-paste into `tmux.SessionName(id)`, not `SendTurn`. Same `/run` URL. Proof is tmux/receiver accept, not model delivery. Provenance is source `"snippet"`, never Inspector.

Kind `prompt` into an `isShell` pane (including a CLI whose TUI has exited — `termHoldsCLI` stays true from the launch row) is **409 `kind`** in v1. Kind `shell` into any agent target stays **409 `kind`**. Do not write `.claude/commands/`, `.opencode/commands/` or `.cursor/commands/`. Optional export to Pi templates is v1.1, opt-in, lossy.

## Consequences

Easier: one library, three runtimes; terminal-averse users get GUI CRUD; guest CLIs do not need their own command files; Attach stays the CLI-only file+caption door it is.

Harder: a new SQLite table and feed entity; both desktop and mobile composers must grow `run === "snip"`; kind `shell` is a second submit primitive next to Inspector type (which still does not press Enter).

Who breaks if we are wrong: a prompt body pasted+Enter into bash (mitigated by 409 `kind` in v1); an unquoted `{{file}}` of `foo; rm -rf /` after the user confirms a shell run (mitigated by preview+confirm, not auto-quote); a Global chord stolen from guest TUIs (v1 ships none).

## Alternatives considered

| Alternative | Why not |
|---|---|
| File-only store (Cursor / Claude / Pi) | Fails the GUI-first audience; PiCode would not own the library |
| Wrap Pi templates only | Guest CLIs never see `~/.pi/agent/prompts` |
| Write into every CLI's command dir | Mutates another product; incomplete coverage (Grok/Hermes); `` !` `` footgun |
| `${input:name}` or `$1` syntax | Collides with real shell `${VAR}` / `$1` |
| Snippets as an app (`#/app/snippets`) | ADR-0109 closed doors; this is host chrome like Automations |
| Extend ADR-0089 Attach to plain shells | Attach is file+caption for CLI TUIs; prompt-into-bash is the defect Q2b closed |
| Reuse `internal/snippet` | That package runs a fence in cwd |
