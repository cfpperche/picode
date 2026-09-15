# Study: activity signals for Muse Code and Antigravity CLI (Orca, Happy, local receipts)

- **Date:** 2026-09-15
- **Sources:** stablyai/orca repo (agent-kind/detection, `terminal wait --for tui-idle`,
  multiplex-term agy registration, PR #2389/#4852), slopus/happy PR #1430 (agy
  backend), Antigravity CLI docs (`/cli/statusline`, `/cli/title`, examples),
  Guillaume Laforge's title/statusline guide, tiny-flowlab/agy-statusline-custom,
  `muse schema` MSP bundle (1.3.0, offline), `muse session-message list`,
  binary strings of both CLIs, and live PTY measurements on this machine
  (tmux `pane_title` polling + OSC capture during real turns).
- **Scope:** what reports *working/idle* for the `muse` and `agy` binaries
  PiCode launches — input to Fatia 5 of `docs/plans/launch-muse-agy.md`.
  Cost telemetry stays in `2026-09-07-cross-cli-agent-telemetry.md`.

## How the benchmarks do it

| Tool | agy activity | muse activity |
|---|---|---|
| Orca | allowlisted `AgentKind.antigravity`: argv match (`agy`) **plus window-title
patterns (`Antigravity`, `✦`)**; `terminal wait --for tui-idle` resolves on
the working→idle **OSC title transition** (tier-1). PR #2389 wires catalog,
detection, telemetry and notifications. | **No matcher.** Unlisted agents are
invisible to the Agents tab by design (`agent-kind.ts` allowlist). |
| multiplex-term | Same shape: `AgentSignature` on `agy` argv + dynamic titles. | — |
| Happy (`agy` backend) | **None from the CLI.** It spawns `agy --print` per turn and derives
phase (typing/working/tool) from its own child + SSE hints (3 s refresh,
10 s TTL) + run boundaries. Documented limitation, not a channel. | No muse
backend (Claude/Codex only). |
| Vibe Kanban (10 agents) | Not listed. | Not listed. |
| Community agy tooling | statusline/title **command scripts** (`statusline.sh`, `title.sh`,
`agy-statusline-custom`): the CLI pipes agent-state JSON to a script on
every state change. | — |

## Local receipts (this machine, 1.2.3 / 1.3.0)

- **Neither TUI sets the terminal title, working or idle.** `pane_title`
  frozen (muse: cwd basename; agy: hostname), **zero OSC sequences** in
  either buffer across boot, turns and return-to-idle. Orca's tier-1 has
  nothing to read here — the 2026-09-14 title refusal (six CLIs) extends
  to both. Orca's "dynamic Antigravity titles" claim does not hold for
  agy 1.2.3 defaults (IDE or customized `title.sh`, not the stock CLI).
- **agy HAS a first-party push surface: `title` + `statusLine` blocks in
  `settings.json`** (`{type: command, command: <script>}`). Measured live:
  the script fires on every state change with `agent_state` in
  `idle | thinking | working | tool_use | initializing` (observed
  `authenticating → idle → working → idle` across boot + one turn), plus
  conversation_id, cwd, model, tokens, quota. This is Orca's tier-2 and
  Claude's `--settings`-hook shape — a PiCode script can POST to the hook
  endpoint. Notes: `/title` takes only `on|off` in 1.2.3 (custom command
  goes in settings.json + restart); the payload's `transcript_path`
  points at the IDE dir and is stale — resolve the brain dir from
  `conversation_id` as `AgySource` already does.
- **muse has no push surface in-binary** (zero `hook`/`statusline`/
  `title`/`notify` strings; `plugins` subcommand unavailable). Two
  level-4 leads, both unread today: `muse serve` (MSP host over stdio;
  offline schema lists `session/*`, `turn/*`, `view/subscribe` and push
  notifications `turn/started`, `turn/completed`, `session/statusChanged`,
  `userInput/requested`) and `session-message list` (currently answers
  `external_agent_ingress_closed` — needs the serve host).
- **Settings hygiene:** agy `settings.json` holds only
  `model`+`trustedWorkspaces` by default; no hooks key exists.

## What PiCode adapts (Fatia 5)

1. **agy: a PiCode `title` command script** that POSTs `agent_state` to the
   existing hook endpoint — same trust shape as the Claude `--settings`
   injection (server-written script + settings block, owner-visible,
   removable). `statusLine` stays the user's (never hijack their bar).
2. **muse: spike `muse serve` + `view/subscribe`** for
   turn/session notifications; fall back to honest `Open` if the host
   cannot observe a foreign TUI's sessions.
3. **No title watching for either** (measured absent), no screen
   scraping as primary (refused 2026-09-03), no Happy-style derived
   phases (we host interactive TUIs, not `--print` children).
