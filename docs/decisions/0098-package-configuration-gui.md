# ADR-0098: Package configuration in the Packages view

- **Status**: accepted
- **Date**: 2026-09-08
- **Extends**: [ADR-0010](0010-pi-packages.md), [ADR-0028](0028-model-roles.md);
  supersedes one clause of [ADR-0033](0033-roles-per-agent-overlay.md)

## Context

Packages that configure pi behaviour — pi-roles today, pi-compact next —
could only be configured through slash commands or by hand-editing JSON.
ADR-0033 §5 explicitly said "Still not SQLite. Both files are workspace
files the extension owns. No PiCode GUI page." That clause kept the files
authoritative but left terminal-averse users without a way to see, edit or
undo a role assignment, and it conflated two questions: where the truth
lives (files) and whether a GUI may edit it.

The benchmarks converge on the shape:

- **VS Code** scopes extension settings to user/workspace, filters
  `@modified`, and resets one setting through the gear — reset is
  per-scope, never a global "clear everything".
- **Raycast** declares extension preferences in the manifest and gates
  commands on `required` values — configuration is a property of the
  package, surfaced where the package is managed.
- **Obsidian** gives every plugin an optional settings tab next to
  enable/disable — installed list and configuration are adjacent.
- **Home Assistant** separates *reconfigure* (setup data) from *options*
  (preferences) and makes "reload after change" an explicit, named step —
  saving and applying are not silently the same event.

pi-roles' own semantics define what "reset" must mean here: the agent
overlay wins per slot over the workspace file; removing an override
reveals the inherited value; a layer's file can be cleared wholesale
(`/roles clear`, confirmed); and the extension ignores unknown keys, which
every write must preserve.

## Decision

1. **Known adapters get a GUI; the files stay the only source of truth.**
   ADR-0033 §5's "No PiCode GUI page" is superseded for adapters the
   server recognises (`pipkg.ConfigKindOf`; today `pi-roles` → `roles`).
   No configuration is copied into SQLite (ADR-0005/0010 unchanged). The
   server reads and writes the same files the extension reads and writes:
   the workspace file at `<workspace>/.pi/roles.json` and the agent
   overlay at `<AgentCwd>/.pi/roles/<agentId>.json` (the same
   `store.AgentCwd` rule the runtime uses, so a worktree-bound agent's
   overlay is found where the extension looks for it).
2. **API**: `GET/PUT/DELETE /api/packages/config` return both layers plus
   the effective merge. PUT validates with the extension's rules (model
   `provider/id`, thinking enum, reserved names, duplicates), merges the
   typed config back onto the raw document so unknown keys survive, and
   writes atomically (tmp + rename). DELETE removes one layer's file and
   is idempotent. Saves and clears publish an ephemeral `packages.config`
   feed event (ADR-0048).
3. **Reset is scoped, never one button.** "Use workspace value" deletes an
   agent override and reveals the inherited slot; "Clear" removes a slot
   from the workspace layer; "Clear file…" deletes one layer's file after
   a confirm (the `/roles clear` semantics). The agent-scope view renders
   inherited slots read-only with an "Override for this agent" seed, so
   inheritance is visible and breakable in one click.
4. **A file the parser rejects is never silently overwritten.** GET
   reports the parse error; PUT answers 409 and the UI requires an
   explicit "Replace file…" (force), mirroring how the extension treats
   invalid config as an error state, not an empty one.
5. **No invented scopes.** pi-roles has no machine-level config file, so
   the GUI offers exactly workspace and agent. Absence of an adapter is
   honest: packages without one show no Configure affordance rather than a
   generic JSON editor (v1) and never a claim that they "have no settings".
6. **Honest application.** The page says "Roles apply on the agent's next
   message" (the extension reloads the file per input); it never claims a
   running agent picked the change up mid-turn. Active-mode switching
   stays with the composer chip and slash commands — this surface edits
   the *configuration*, not the session's live lock.

## Consequences

- Easier: terminal-averse users can view/edit/reset roles per workspace
  and per agent; the effective merge answers "which model will this agent
  actually route to"; pi-compact (delivery 2) reuses the adapter shape
  instead of a bespoke page.
- Harder: `RolesConfigFromMap` in Go and `rolesConfigSchema` in Zod must
  track the extension's logic.ts (three implementations of one grammar).
  Schema changes need the coordinated bump ADR-0028 already warned about.
  The adapter registry is code, not data — adding a package's editor is a
  server + client change until a declarative manifest proves itself.
- If wrong: the GUI diverges from the extension (drift shows up as the
  file "losing" keys — the unknown-key preservation tests guard the most
  likely break), or a package needs richer config than files give; both
  are recoverable by superseding this ADR.

## Alternatives considered

- **Keep "No PiCode GUI page" (ADR-0033 §5) literally.** Rejected: it
  protected the files from a SQLite copy, not users from a GUI; the
  GUI-only-for-files boundary achieves the same protection.
- **Generic JSON editor per package.** Rejected for v1: no validation, no
  layer semantics, no reset meaning — a text box is how the file gets
  mangled, not managed.
- **Copy config into SQLite per agent** (agent overrides as store rows).
  Rejected again (ADR-0005/0028/0033): two sources of truth, and a plain
  `pi` in a terminal would not see agent overrides.
- **Declarative config manifest in package.json** (Raycast-style
  `preferences`). The right end state for third-party packages, but
  designing it from one adapter would be speculation; extract it after
  pi-compact proves the second case (delivery 2).

## Sources

- VS Code settings scopes, `@modified`, per-setting reset:
  <https://code.visualstudio.com/docs/configure/settings>
- Raycast manifest preferences (`required`, types):
  <https://developers.raycast.com/information/manifest>
- Obsidian plugin settings tabs:
  <https://docs.obsidian.md/Plugins/User+interface/Settings>
- Home Assistant reconfigure/options flows and reload semantics:
  <https://developers.home-assistant.io/docs/core/integration/config_flow/>,
  <https://developers.home-assistant.io/docs/core/integration/options_flow/>
- Zed settings editor + project overrides (GUI beside JSON, not instead):
  <https://github.com/zed-industries/zed/blob/main/docs/.doc-examples/configuration.md>
