# Compaction policy (ADR-0061)

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

**An extension, not core.** Opt-in pi package at `packages/pi-compact/`
(MIT; the rest of this tree is PolyForm Noncommercial). The Go server has
no code path that configures, config-reads or invokes it — its only
involvement is exporting `PI_COMPACT_AGENT=<id>` into agent processes
(`internal/store/agents.go`, `Agent.SpawnEnv`). Everything else runs
inside each pi session, like any installed extension; uninstalling the
package (or the config file) restores Pi's stock behavior exactly.
Users install it with `pi install -l` / `#/clis/pi/packages`.

Dormant until configured — a missing config file applies nothing (same
rule as roles): no early trigger, no summarizer override, status line
reads `compact: not configured · /compact-edit`. A config layer file is
the opt-in; documented schema defaults fill keys the file does not set.
When configured: early compact at 100k tokens or 50% of the window
(whichever first, not below 32k) on `agent_settled` (never `turn_end` —
`ctx.compact()` aborts mid-run turns), summarizer thinking per config
(default off), cheap-model chain `config.model` → `fallback` → auto
chain → session model with per-link retry, Pi's overflow compact stays
on. Pi's TUI hard-codes the `/compact` dispatch, so the package
registers the hyphenated family `/compact-edit|model|on|off` (second
amendment); bare `/compact` stays native and the `session_before_compact`
hook still feeds it the cheap chain. Workspace file
`<cwd>/.pi/compact.json`, agent overlay `<cwd>/.pi/compact/<id>.json`
when `PI_COMPACT_AGENT` is set; no machine layer. The recent-token tail
is not dropped.
