# ADR-0061: Compaction policy as an opt-in pi package

- **Status**: accepted (amended 2026-09-04 after first dogfood — no defaults, dormant until configured; trigger moved to `agent_settled`)
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

> **Amendment 2026-09-04 (owner, after first dogfood): no defaults, ever.**
> Dogfood surfaced two defects in the original decision: (a) with no config
> file the package silently imposed a policy and a summarizer chain whose
> first links now 404 (gemini-2.5-flash sunset for newer Google accounts);
> (b) triggering from `turn_end` aborted active agent runs (`ctx.compact()`
> starts with `abort()`) — "the agent does not continue after compaction".
> Points 3, 6 and 7 below record the amended decision.

1. **MIT in that directory only**, same carve-out as ADR-0028.
2. **Opt-in install.** Nothing is installed by default (`pi install` /
   `#/packages`, ADR-0010). Config is not copied into SQLite (ADR-0005).
3. **Dormant until configured — no defaults, ever.** While no config layer
   exists, the package applies nothing: Pi's stock compaction and
   summarizer run untouched, and the status line reads
   `compact: not configured · /compact edit`. Any layer file (workspace
   `.pi/compact.json` or a per-agent overlay) is the explicit opt-in;
   documented schema defaults then fill keys the file does not set, and
   `/compact edit` shows exactly what will be written before saving.
   While unconfigured, bare `/compact` and `/compact on|off` report
   "not configured" instead of acting.
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
6. **Early trigger is `agent_settled` + `ctx.compact()`, never `turn_end`.**
   Pi emits `turn_end` between turns of an active run, and `ctx.compact()`
   begins with `abort()` — dogfood showed the run dying with
   "This operation was aborted" before the compaction banner. `agent_settled`
   is emitted after Pi marks the run inactive (the same boundary Pi's own
   threshold compaction uses), and the trigger additionally checks
   `ctx.isIdle()`. Overflow and threshold compact inside Pi remain the
   safety net. The package never sets `compaction.enabled: false`.
7. **`session_before_compact` picks a summarizer** from `model`, then
   `fallback[]`, then the auto cheap chain, then the session model, with
   configured thinking (default `off`). Links are tried in order: a link
   that throws, stops with an error, or yields an empty or length-capped
   summary falls through to the next; Pi's default summarizer runs only
   when every link failed. The auto chain default is
   `google/gemini-3.6-flash` → `anthropic/claude-haiku-4-5` (2.5-flash
   now 404s for newer Google accounts). The cut keeps
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

Harder: installing the package alone changes nothing until a config file
exists (deliberate — the status line points at `/compact edit`). The
package shadows Pi's `/compact` (by design; `ctx.compact()` is the
equivalent). Two files to explain (workspace + overlay). Auto cheap
models may be missing if the user has no Google/Anthropic auth — then the
session model is used with thinking off.

If wrong: uninstalling the package restores Pi's command and default
timing. Overlay files can be deleted by hand.

## Alternatives considered

- **A `/compaction` command family, leaving `/compact` to Pi.** Rejected:
  two vocabularies for one action. Owner chose to overlay `/compact`.
- **Dormant until a JSON file exists (pi-roles).** Originally rejected to
  keep zero-step activation; **chosen by amendment** after dogfood — the
  owner's no-defaults directive outweighs zero-step activation, and the
  unconfigured status line points straight at `/compact edit`.
- **Drop the recent-token tail and keep only the summary.** Rejected for
  a coding ADE: the last ~20k tokens are the live reads/edits.
- **Disable Pi auto-compact and own every trigger.** Rejected: overflow
  recovery must stay an airbag.
- **Per-model profiles in Pi settings (`compaction.profiles`).** That is
  an upstream Pi change ([earendil-works/pi#8133](https://github.com/earendil-works/pi/issues/8133)).
  A package can ship now without waiting.
- **PiCode GUI / composer chip in M1.** Deferred. Commands and dialogs
  already surface as chat cards. A state file may be published later.
