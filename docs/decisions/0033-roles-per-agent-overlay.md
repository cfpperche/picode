# ADR-0033: Model roles per-agent overlay

- **Status**: accepted
- **Date**: 2026-08-31
- **Extends**: [ADR-0028](0028-model-roles.md)

## Context

ADR-0028 stores roles in `<cwd>/.pi/roles.json`. Two PiCode agents in the
same workspace share that file, so they cannot keep different defaults
without fighting. Copying the config into SQLite would reject ADR-0005.
The deferred M2.1 note in 0028 was "override via env at spawn."

A terminal `pi` has no PiCode agent id. The overlay must be opt-in per
process, not implied by the folder.

## Decision

1. **Workspace file stays the default.** Missing overlay = v1 behaviour.
2. **Process env `PI_ROLES_AGENT=<id>`** selects `<cwd>/.pi/roles/<id>.json`.
   PiCode sets this on managed RPC start and on the agent's TUI tmux
   session (`Agent.SpawnEnv`). A raw `pi` in a terminal does not get it.
3. **Overlay merges on top of the workspace file.** Builtin slots and
   custom names in the overlay win; everything else is inherited.
   `/roles edit|add` in that process writes the overlay, not the
   workspace file. `/roles remove` only deletes overlay customs —
   workspace presets stay until edited in a process without the env.
4. **The id is a filename slug** (`[A-Za-z0-9][A-Za-z0-9_-]{0,63}`).
   PiCode uses the agent id. Plain pi may set any slug.
5. **Still not SQLite.** Both files are workspace files the extension
   owns. No PiCode GUI page.

## Consequences

Easier: two agents in one folder can override `default` without forking
the repo file; TUI `pi` in the same folder keeps the shared file.

Harder: two files to explain; overlay filenames are PiCode ids, not
display names; a renamed agent keeps the old overlay until the user
moves the file.

If wrong: dropping the env returns every process to v1. Overlay files
can be deleted by hand.

## Alternatives considered

- **Copy roles into the agent row in SQLite.** Rejects ADR-0005 / 0028.
- **Key overlays by agent display name.** Names collide and rename.
- **Replace (not merge) when the overlay exists.** First `/roles edit`
  would freeze the workspace file into the agent copy and diverge
  forever. Merge keeps shared `vision`/`plan` unless overridden.
- **`PI_ROLES_FILE` absolute path.** Useful later; not needed to ship
  M2.1. The slug path is enough and stays inside the workspace.

## Amendment (2026-08-31): save-to select and /roles clear

Dogfooding showed the fixed write target (§3: env ⇒ overlay, always) made
the workspace file unreachable from any PiCode surface — chat and agent
TUIs both carry `PI_ROLES_AGENT`, so only a plain terminal `pi` could
write `.pi/roles.json`.

Owner-approved changes:

1. Under `PI_ROLES_AGENT`, `/roles edit` and `/roles add` end with a
   **Save to** select — *this agent* (overlay, the old behavior and the
   first option) or *workspace* (`.pi/roles.json`). Without the env the
   question is skipped and the workspace file is written, as before.
   `/roles remove` still deletes overlay customs only.
2. **`/roles clear [agent|workspace]`** deletes a whole roles file after
   a confirm. With no argument under the env it asks which; without the
   env it clears the workspace file. A lock whose role stops resolving
   falls back to `/auto`.

## Amendment #2 (2026-08-31): active-role state file, restored locks, scoped remove, rich labels

Nothing outside the chat said which role was active, and a lock died with
the pi process. Per the ADR-0036 philosophy (extensions never paint PiCode
chrome; first-party consumers mature a contract before it generalizes):

1. **State contract v1.** Under `PI_ROLES_AGENT`, the extension writes
   `~/.pi/agent/roles-state/<agentId>.json` on every mode change and
   whenever the effective role list changes (never on per-input routing):
   `{v:1, mode:"auto"|"lock", role?, model?, thinking?, roles:[{name,
   model?, thinking?}]}` — `roles` is the merged list so the UI needs no
   second source. The file is ephemeral and best-effort; deleting it only
   forgets the lock. PiCode serves it at `GET /api/agents/{id}/role-state`
   (`{"state": null}` when absent, unreadable, or from a future version)
   and renders a composer chip from it — present only when the file is
   (the package install is the feature toggle; no settings flag).
2. **Locks survive restarts.** `session_start` restores `mode` from the
   state file when the role still resolves; the model applies on the next
   input (never at startup — that would fight `--model`, ADR-0009).
3. **`/roles remove` gained the scope edit/add have** — with smart-skip:
   a preset held by one layer is removed from it without a question; held
   by both, a `Remove from` select (this agent / workspace / back) asks.
   §3's "overlay customs only" rule is superseded.
4. **Selects carry definitions.** Role pickers list
   `vision — xai/grok-4.5 · medium` (extension-built labels, so the TUI
   benefits too); the extension parses the name back out of the choice.
