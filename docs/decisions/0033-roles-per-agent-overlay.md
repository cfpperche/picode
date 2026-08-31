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
