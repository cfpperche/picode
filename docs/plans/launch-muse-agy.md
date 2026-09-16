# Launch parity for Muse Code and Antigravity — plan and spike findings

Goal (owner, 2026-09-15): `muse` and `agy` launch like the other CLIs before
any real setup tab is implemented for them. "Equivalent" means the row below
reads the same for every launchable CLI.

Probe versions: Muse Code 1.3.0, agy 1.2.3 (2026-09-15, owner's machine).

## Done means

| Capability | others | muse | agy |
|---|---|---|---|
| Editable launch (defaults + profiles + preview) | yes | read-only | read-only |
| `Launch settings` in the row menu | yes | no | no |
| Customize checkbox on New terminal | yes | no | no |
| Activity `Ready` / `Working` (else honest `Open`) | yes | `Open` | `Open` |
| Sessions + resume | yes | yes | yes |
| `Continue in…` as source (Reader) | yes | no | no |
| `Continue in…` as brief target (Prompter) | yes | no | no |
| `Continue in…` as native target (Writer) | yes | no | no |

An adapter is three integration points, not one flag: (1) the `Catalog()`
row without `SurfaceTerminal`, (2) a branch in `cliIntegrationPlan()` with
generated assets and a hook receiver, (3) `Reader` / `Writer` / `Prompter`
on the `clisession` source (discovered by interface, never by CLI id).

## Spike findings (Fatia 0, this note)

Both CLIs have more surface than assumed. All probes were read-only
(`--help`, `models`, sqlite copies in /tmp, plugin docs); nothing was
written to either CLI's store.

### Muse Code — GO for Prompter and Reader, LIKELY for activity

- **Prompter, exact shape**: `muse [PROMPT]` starts a session with the
  prompt; `resume <uuid>` / `resume --last` confirmed.
- **Reader, official path**: `muse export --session <id> --out <file>`
  writes one self-contained JSON (export_schema_version 1: timestamps,
  messages, tool calls/results, approvals, model ids, lineage) and stays
  non-interactive when piped. No reverse-engineering needed.
- **Activity lead**: native plugin family `hooks` (`{id, event, command,
  timeoutMs?, statusMessage?}`, argv — not shell — commands under the
  plugin root). `PreToolUse` is documented; the full event list
  (session/prompt/stop boundaries) is **unconfirmed** — Fatia 3 must
  enumerate it (`muse schema`, plugin docs, or a probe plugin that logs).
  This mirrors the Claude `--settings` and OpenCode plugin branches.
- **No pollable activity in the index**: `sessions.status` is only
  `valid` / `missing_metadata`. `updated_at_us` moves on writes but is a
  heuristic, not a report — do not ship it as `Working`.
- Writer is UNKNOWN until the Fatia 4 spike (sessions live under
  `sessions/YYYY/MM/DD/<uuid>/` + index row; the CLI must accept a
  hand-made one).

### Antigravity — GO for Prompter, LIKELY for Reader, activity via polling fallback only

- **Prompter, exact shape**: `--prompt-interactive <text>` (initial prompt,
  session continues); `--print` + `--output-format json` for headless;
  `--conversation <id>` / `--continue` for resume; `--model`, `--effort`,
  `--mode` confirmed. `agy models` enumerates model ids (network fetch).
- **Reader, likely**: full trajectories are per-conversation sqlite
  (`conversations/<uuid>.db`: `steps(idx, step_type, status, …,
  step_payload, step_format)` + `gen_metadata`). Codes are numeric
  (`step_type` in {9,14,15,23,98} on the sampled db) — Fatia 2 maps them.
- **Activity: no hooks anywhere** (`settings.json` holds only
  `model` + `trustedWorkspaces`). Pollable signals exist:
  `presence/<uuid>.lock` files (open-conversation locks),
  `conversation_summaries.{status,not_fully_idle,killed,
  last_user_input_time}` (`CASCADE_RUN_STATUS_IDLE` observed). A polling
  fallback is plausible but must be labeled honestly (lock staleness after
  a crash is the known risk — Fatia 3 verifies lock lifecycle
  open → close → kill -9). If it cannot be made truthful, agy keeps
  `Open` plus the Codex-style "completion only" branch.

## Slices (one branch/worktree each)

- **Fatia 1 — Prompter for both.** `PromptArgs` (~5 lines each) +
  table-driven tests. Unlocks both as brief `Continue in…` targets.
- **Fatia 2 — Reader: muse via `export`, agy via conv-db mapping.**
  Unlocks `Continue in…` on muse terminals (source). Agy Reader joins
  only if the step codes map cleanly, else recorded debt.
  Outcome (shipped): agy reads brain transcript.jsonl, not the SQLite
  (web research + local 9/9 id match; protobuf fields unversioned).
  muse `export` gotcha: approval records carry `model` as an object, so
  it decodes late.
- **Fatia 3 — launch flip (3a muse, 3b agy, independent).**
  Drop `SurfaceTerminal`, add the `cliIntegrationPlan` branch per the
  findings above, assets + receiver, preview, profiles, menu item,
  Customize. Decision table first (installed? integration on/off? hook
  supported? resume vs new?), one test per row, including the known
  uncovered tmux-failure branch.
  Outcome 3a (shipped): NO hook surface in Muse Code 1.3.0 — zero
  "hook" strings in the binary, `plugins` subcommand unavailable, no
  user plugin path — so the flip shipped WITHOUT activity: plan summary
  only, toggle hidden behind `hasIntegrationMechanism` (catalog row +
  PUT/prepend guards), seed skips hookless CLIs (copy forced off, never
  persisted). Fatia 5 must revisit the seed flag for existing instances
  when a mechanism appears.- **Fatia 3b outcome (shipped):** same shape as 3a, smaller — registry flip
  + summary-only plan branch; toggle/PUT/seed guards needed zero new code.
  No `surface: terminal` rows remain.
- **Fatia 4 — Writers (native targets, may fail).**
  Outcome (shipped, both GO): muse = index row + session log of plain
  records (UUID-shaped id enforced — the CLI rejects anything else
  before the store lookup; wire field order kept, exporter sniffs the
  line prefix); agy = summaries row + brain transcript + existing
  (possibly empty) store file. Both continue live turns afterwards. Minimum-viable-file
  spike against the real CLI (the Grok ritual), `Write` + round-trip.
  Explicit fallback: brief-only, recorded — never forced.
- **Fatia 5 — activity beyond `Open`.**
  Outcome (shipped for agy): title-command reporter (measured live:
  authenticating/idle/working/idle; stale transcript_path dropped).
  Terminal titles carry no signal in either build (measured absent),
  muse has no push surface (serve deferred to managed agents per owner).
  No hooks.json decision hook: it gates tools. Only on a confirmed surface;
  otherwise `Open` stays and the docs say so.

Open questions for the slices: full muse hook event list; agy lock
staleness; whether `agy --model` ids stay stable enough to offer in
launch args; writer acceptance for both CLIs.
