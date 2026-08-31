# ADR-0025: The whole tmux catalog is a settings surface

- **Status**: accepted
- **Date**: 2026-08-30
- **Supersedes in part**: ADR-0024's rule that the flag list "grows from real
  parity gaps" and its curated-only registry.

## Context

ADR-0024 shipped with one exposed flag (`mouse`) and a rule that the list
grows only when a parity gap is hit. The owner overruled the rule the same
day: the GUI must expose the entire tmux configuration space, not an agent's
curated subset. Decisions here are provisional (AGENTS.md #6); this one
changed, and this ADR records the new shape.

Two facts measured on the owner's machine (tmux 3.6) shaped the design:

1. **The catalog is large and typed only by convention** — 159 options across
   three scopes (31 server, 61 session, 67 window). tmux does not expose
   value types; it validates at `set-option` time with a usable message
   ("value is invalid: abc", "unknown value: middle").
2. **tmux normalises more than a whitelist would guess** — `yes` is a valid
   boolean, discovered when a test asserted the opposite. A hand-written
   validator here would be wrong exactly where tmux is lenient.

## Decision

1. **The catalog is read live from the running tmux** (`show-options -sg/-g/
   -wg`), never compiled in. Options gained or retired between tmux versions
   appear and disappear on their own, and the values shown are the user's
   real ones — including whatever their tmux.conf set, since PiCode shares
   the user's socket by choice.
2. **tmux is the validator.** A curated flag keeps its enum check; everything
   else is applied to a scratch session (or, for server options, for real)
   and tmux's own refusal is surfaced verbatim, trimmed of command-line
   noise. A whitelist that drifts from the installed tmux would refuse valid
   values and accept stale ones.
3. **The old code-level forces became curated defaults** — `status off`,
   `allow-passthrough on`, `extended-keys on`, `extended-keys-format xterm`
   moved from hardcoded `set-option` calls in the bridge into the same
   defaults layer as `mouse`. A user override now wins over every one of
   them, with the consequence written beside the control instead of enforced
   silently.
4. **Scope is the page structure.** Session and window options are
   per-terminal (a PiCode terminal is one session with one window). Server
   options appear only on the global page under "This machine's tmux",
   labelled as reaching every session on the machine — PiCode's and the
   user's alike. The API refuses a server-scoped key on a per-terminal PATCH.
5. **Dangerous options are labelled, never hidden** — `destroy-unattached`,
   `exit-unattached`, `default-terminal` and the rest carry the consequence
   in plain words. Hiding them would contradict the point of exposing the
   catalog.
6. **Arrays are visible but not editable in V1** (`command-alias[]`,
   `terminal-features[]`, …): shown with a note, edited via tmux.conf. Raw
   text editing for styles and format strings; visual editors can come later.

## Consequences

- **Easier**: any tmux behaviour is a setting now. The next parity gap needs
  no release.
- **Easier**: PiCode stops owning a validator it cannot keep correct.
- **Harder**: the page must make 140+ rows navigable — search is the primary
  control, and the featured tier (curated flags with help text) sits above
  the raw catalog.
- **Cost accepted**: a scratch tmux session is created and killed to validate
  non-curated values when no owned session fits. ~50ms per save.
- **Risk accepted**: the user can configure PiCode's surface into a broken
  state (status bar on, passthrough off). The warnings say so; the defaults
  remain one click away (`Inherit`).

## Weighed and not taken

| Option | Why it lost, today |
|---|---|
| Curated-only registry (ADR-0024's original rule) | the owner asked for the whole space, and the registry was already wrong once (`yes` as a boolean) |
| Compiled-in catalog with types | drifts from the installed tmux; the live server is the only honest source |
| Hiding dangerous options | contradicts the decision; labelling keeps agency with the user |
| Editable arrays in V1 | indexed writes (`name[2]`) need list UI; deferred, shown read-only so nothing is hidden — **paid off 2026-08-30, see the amendment below** |

## Amendment — 2026-08-30: arrays are editable, as a block

The read-only debt above is paid. An array option is edited as **text, one
entry per line**: line *n* is `name[n]`, blank lines are ignored, and Apply
replaces the whole list. The block is what the catalog already carried —
`show-options` reports the entries in index order, so joining them with
newlines loses nothing and needs no list widget.

Two things were measured on tmux 3.6 before the code was written, and both
shape it:

**Shrinking a list needs per-index unsets.** Writing `name[0]` and `name[1]`
over a three-entry layer leaves `name[2]` exactly where it was — it survives
every rewrite. So `SetArrayOption` writes the new entries and then unsets each
index the layer had beyond the new length. Unsetting the whole option first
would be worse, not better: it resurfaces the layer underneath (for a server
option, tmux's own defaults) between the two writes.

**An empty list cannot be pinned.** tmux keeps no empty array layer — removing
the last index removes the override itself. An empty block is therefore
refused with a message pointing at Reset, rather than stored as a pin that
silently behaves as inherit.

One bug surfaced in the browser QA that had nothing to do with arrays and
everything to do with clearing: the **global** panel dropped a cleared
non-curated key from the store but never unset it on the live sessions, so it
stayed applied forever. The per-terminal path had always done this
(`applyPatchLive`); the global one now does too (`unsetClearedEverywhere`),
with a test that fails without it.
