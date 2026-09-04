# ADR-0060: Compaction policy as an opt-in pi package

- **Status**: accepted
- **Date**: 2026-09-04
- **Extends**: [ADR-0028](0028-model-roles.md), [ADR-0010](0010-pi-packages.md), [ADR-0033](0033-roles-per-agent-overlay.md)

## Context

Pi auto-compacts when `contextTokens > contextWindow - reserveTokens`
(default reserve 16 384). On a 1M-window model that is ~984k tokens; the
summarization request itself has overflowed (`Prompt is too long`). On
adaptive-thinking models at high effort, summarization inherits the session
thinking level, shares the ~13k output cap with thinking tokens, and can
fail in a loop.

Pi already exposes the hooks a package needs: `ctx.compact()`,
`session_before_compact`, `ctx.getContextUsage()`, and
`modelRegistry.complete(..., { reasoning })`. The built-in `/compact
[instructions]` command is the vocabulary users already know.

A coding ADE should not drop Pi's ~20k recent-token tail (the files just
read and edited). The package is policy in front of Pi's compact, not a
replacement algorithm.

## Decision

PiCode ships **`packages/pi-compact/`**, a pi package:

1. **MIT in that directory only**, same carve-out as ADR-0028.
2. **Opt-in install.** Nothing is installed by default (`pi install` /
   `#/packages`, ADR-0010). Config is not copied into SQLite (ADR-0005).
3. **Active defaults without a file.** Unlike `pi-roles` (dormant until
   `.pi/roles.json` exists, so it cannot fight per-agent `--model`), this
   package applies a conservative policy as soon as it is loaded:
   early compact at 100k tokens **or** 50% of the window, whichever
   comes first, not below 32k; summarizer thinking `off`; cheap-model
   auto chain then the session model; Pi's overflow compact stays on.
4. **Workspace file** `<cwd>/.pi/compact.json` overlays those defaults.
   With `PI_COMPACT_AGENT=<id>` (PiCode `Agent.SpawnEnv`, same slug
   rules as ADR-0033), `<cwd>/.pi/compact/<id>.json` overlays the
   workspace file. Overlay keys win; missing keys inherit.
5. **The package registers `/compact`.** Extension commands run first, so
   this replaces Pi's built-in command. `/compact` and `/compact <text>`
   call `ctx.compact()` (same pipeline as Pi, including
   `session_before_compact`). Sole-word subcommands: `edit`, `on`,
   `off`, `model`. A first word plus more text is always instructions
   (`/compact edit the summary` still compact).
6. **Early trigger is `turn_end` + `ctx.compact()`.** Overflow and
   threshold compact inside Pi remain the safety net. The package never
   sets `compaction.enabled: false`.
7. **`session_before_compact` picks a summarizer** from `model`, then
   `fallback[]`, then the auto cheap chain, then the session model, with
   configured thinking (default `off`). If none are usable, the handler
   returns and Pi's default summarizer runs. The cut keeps
   `preparation.firstKeptEntryId` (Pi's recent-token tail).
8. **`/compact on` / `off` are session locks** for the early trigger.
   They do not persist. Config `enabled` is the persistent switch.
   Custom summarization still applies to manual `/compact` and overflow.
9. **Cancel aborts, back is explicit** in the edit wizard (ADR-0028
   amendment). Under `PI_COMPACT_AGENT`, save asks *this agent* vs
   *workspace*.

npm publish is a follow-up, same train as `pi-roles`.

## Consequences

Easier: long sessions compact before the window edge; summarization does
not inherit xhigh thinking; one `/compact` vocabulary; the same extension
runs in the TUI and in PiCode.

Harder: the package shadows Pi's `/compact` (by design; `ctx.compact()`
is the equivalent). Two files to explain (workspace + overlay). Auto
cheap models may be missing if the user has no Google/Anthropic auth —
then the session model is used with thinking off.

If wrong: uninstalling the package restores Pi's command and default
timing. Overlay files can be deleted by hand.

## Alternatives considered

- **A `/compaction` command family, leaving `/compact` to Pi.** Rejected:
  two vocabularies for one action. Owner chose to overlay `/compact`.
- **Dormant until a JSON file exists (pi-roles).** Rejected: installing
  the package would not fix "too late" until a wizard ran.
- **Drop the recent-token tail and keep only the summary.** Rejected for
  a coding ADE: the last ~20k tokens are the live reads/edits.
- **Disable Pi auto-compact and own every trigger.** Rejected: overflow
  recovery must stay an airbag.
- **Per-model profiles in Pi settings (`compaction.profiles`).** That is
  an upstream Pi change ([earendil-works/pi#8133](https://github.com/earendil-works/pi/issues/8133)).
  A package can ship now without waiting.
- **PiCode GUI / composer chip in M1.** Deferred. Commands and dialogs
  already surface as chat cards. A state file may be published later.
