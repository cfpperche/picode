# Omp (oh-my-pi) onboarding — plan and probe findings

Goal (owner, 2026-09-17): onboard the Omp Agent CLI (https://omp.sh/,
github.com/can1357/oh-my-pi) the way muse and agy were onboarded — catalog
card first, adapter surfaces in measured slices. Omp is a batteries-included
Pi fork (v18.2.4 measured), which makes it the row closest to pi and the one
with the sharpest collision risks (shared env vars, similar JSONL, `-e`
extensions).

## Done means

| Capability | full rows | omp (today) | omp target |
|---|---|---|---|
| Detection + Check setup (`--version`) | yes | yes | yes |
| New terminal (Launchable) | yes | yes | yes |
| Identity (mark, label, favicon, "Open") | yes | yes | yes |
| Guided install (docs link, no script run) | some | yes | yes |
| Sessions list + resume | yes | no | Fatia 2 |
| Editable launch + PATH wrapper + reserved env | yes | no | Fatia 3 |
| Activity beyond `Open` | muse no, others yes | no | Fatia 4 (probe-gated) |
| Managed update check / jobs | yes | no | Fatia 5 |
| Handoff Reader / Prompter / Writer | most | no | Fatia 6 |

## Fatia 0 — probe findings (2026-09-17, omp 18.2.4 via npm)

Installed with `npm install -g @oh-my-pi/pi-coding-agent` on the owner's
machine (the curl installer demands Bun ≥ 1.3.14 and the machine had
1.3.10). All probes read-only except the install itself; no omp session was
created (no credentials were pointed at omp).

- **Bun floor is a real Check-setup hazard.** The npm dist is a Bun bundle
  (`#!/usr/bin/env bun`) that declares `bun >= 1.3.14` and dies with a
  minified `SyntaxError` under older Bun. Measured: 1.3.10 → crash;
  1.3.14 → fine. Check setup will honestly report that failure text.
- **`omp --version`** → `omp/18.2.4`, ~0.7 s — bounded check is fine.
- **Home layout** (from `omp config path` + source strings):
  `~/.omp/agent` (sessions under it), honors `PI_CODING_AGENT_DIR`,
  `PI_CONFIG_DIR`, `OMP_PROFILE`/`PI_PROFILE`, `--session-dir`, `--profile`.
- **Flags confirmed from `--help`**: `-p/--print`, `-c/--continue`,
  `-r/--resume <id|path>` (picker when omitted), `--from-claude/--from-codex`,
  `--mode text|json|rpc|rpc-ui`, `-e/--extension`/`--hook` (repeatable),
  `--no-extensions`, `--approval-mode always-ask|write|yolo`, `--api-key`,
  `--allow-home`, `--cwd`, `--max-time`, `--no-session`.
- **Subcommands** (from the bundle's help registry): acp, agents,
  auth-broker, auth-gateway, bench, browser-relay, cleanse, collab, commit,
  completions, compress, config, dry-balance, gallery, gc, git, grep,
  grievances, if-bench, images, install, join, models, plugin, ps, read,
  render, say, search, setup, share, shell, ssh, stats, tiny-models, token,
  ttsr, update, usage, worktree.
- **`omp update --check`** → "Current version: 18.2.4 / ✔ Already up to
  date", exit 0 — the shape Fatia 5's vendor check can parse. Honors
  `GITHUB_TOKEN`/`GH_TOKEN`.
- **npm layout** → `DetectMethod` classifies it npm (path contains
  `/node_modules/`); the npm package is `@oh-my-pi/pi-coding-agent` — never
  pi's `@earendil-works/pi-coding-agent`.

### Risks carried into the slices

1. **`PI_CODING_AGENT_DIR`**: omp reads pi's agent-dir override. Fatia 3
   must add it to the launcher-owned env keys (beside `GROK_HOME` /
   `HERMES_HOME`) so a launch can never point omp at `~/.pi/agent`.
2. **Session JSONL similarity**: omp's sessions live in the same format
   family as pi's. The Fatia 2 reader must be its own source over
   `~/.omp/agent/sessions` — never reuse `PISource`/`session.ListAll`.
3. **`-e` extension events are unprobed**: whether pi's
   `session_start`/`agent_start`/`ui_prompt_start` fire in omp and whether
   the hook env survives to attribute a report to a terminal needs a real
   authed run (Fatia 4 gate). Muse's lesson applies: no truthful report →
   stay `Open`, never fake `Ready`.
4. **Maintenance passthrough**: the Fatia 3 wrapper needs an `OMP_MAINT`
   list (config, update, models, gc, stats, token, completions, plugin,
   install, setup, acp, share, join, collab, render, export…) so presence
   leases only real sessions; `--mode acp|rpc|rpc-ui` and `-p` are
   non-TUI like pi's.

## Slices

- **Fatia 1 — card (this branch, feat/omp-cli).** Catalog row
  `{"omp", "Omp", "omp", "https://omp.sh/docs", SurfaceTerminal}`, UI
  identity (mark `Om`, vendor favicon), docs. Everything else refused with
  the existing per-capability guards. Shipped 2026-09-17.
- **Fatia 2 — sessions.** Gate: one real omp session (owner runs omp, or
  approves pointing omp at a key). `OmpSource` over `~/.omp/agent/sessions`,
  `ResumeArgs: ["--resume", id]` (confirm on the installed `--help`).
  Outcome (shipped 2026-09-17): owner authorized auth — the probe ran two
  real headless sessions against the owner's Google key via `--api-key`
  (nothing written to omp's auth store), Bun upgraded to 1.4.2 on this
  machine, and the format was read from real files, not docs: schema-v3
  header (`session` id/timestamp/cwd), `title`/`model_change`/
  `thinking_level_change`/`message`/`custom` records, message timestamps in
  epoch millis (outer records ISO), bucket = cwd with "/" → "-".
  `OmpSource` lists machine-wide and per-cwd (filter reads the header's
  cwd, not the bucket name), name = newest non-empty title else first user
  text, model = provider-qualified `model_change` else last message model,
  no-header/no-message files are not sessions. Resume verified headless:
  `omp --resume <id>` continues the conversation (needs `--model` again in
  print mode).
- **Fatia 3 — launch.** Drop `SurfaceTerminal`, `cliIntegrationPlan` branch
  (PATH wrapper + presence lease), `PI_CODING_AGENT_DIR` reserved,
  `OMP_MAINT` passthrough list, decision table.
  Outcome (shipped 2026-09-17): wrapper mirrors muse's shape — maintenance
  subcommands (the omp subcommand registry, measured off `--help`: acp
  through worktree, including its interactive tools git/shell) exec past
  the lease; bare runs, prompts, -c/--resume keep it; protocol modes
  `--mode rpc|json|acp|rpc-ui` mark non-TUI in the shared wrapperLifecycle
  template (pi's row lives in the same case). `PI_CODING_AGENT_DIR` joined
  the launcher-owned env keys (validate refuses it). Seed bumped v3→v4 so
  existing instances give omp the Activity-on default (the muse v2→v3
  precedent). `normalizeTerminalCLI` (server) learned omp — runtime reports
  were being canonicalized to "" without it.
- **Fatia 4 — activity.** Gate: extension-event probe against a real
  session with `PICODE_TERM_ID` in the env. No mechanism measured → omp
  stays honestly `Open` (the muse outcome).
  Outcome (shipped 2026-09-17): the gate measured **GO** on 2026-09-17 —
  a minimal extension loaded under `-e`, `session_start`/`agent_start`/
  `agent_end` all fired, and `PICODE_TERM_ID` survived into the extension
  process. Shipped the pi-shaped terminal-state extension (same template,
  reporter name "omp"), self-guarded on TUI mode so headless runs report
  nothing. Live TUI Ready/Working acceptance with the owner's auth is the
  external remainder (the scratch has no omp credentials by design).
- **Fatia 5 — lifecycle.** Spec: `updateArgs ["update"]`,
  `CheckArgs ["update","--check"]`, vendor output parse; npm fallback via
  the real package name. Uninstall guided (vendor docs).
  Outcome (shipped 2026-09-17): pi-shaped spec with the real npm package
  (`@oh-my-pi/pi-coding-agent`, `npmUpdateOnly` like claude-code): npm
  installs update/reinstall/uninstall through npm; native installs run
  `omp update` / `omp update --force` (measured working); uninstall
  guided. The bun-global layout classifies unknown until observed.
- **Adversarial finding (measured, guarded).** `--trusted-extension` in
  launch args + PiCode's injected `-e` = omp refuses the whole run
  ("--trusted-extension cannot be combined with --extension, -e, or
  --hook"). Preview names the conflict and prepare refuses with the
  reason (`ompTrustedExtensionConflict`, 5-row decision table);
  `--no-extensions` + explicit `-e` measured fine (explicit paths still
  load).
- **Fatia 6 — handoff.** Reader on the JSONL; Prompter via positional
  prompt; Writer only after a minimum-viable-file round-trip against the
  real CLI (the grok ritual), with brief-only fallback recorded.

Open question carried now: does omp accept a hand-made session file (Fatia
6 Writer) — a real session exists now, so the minimum-viable-file spike can
run. The Fatia 3 wrapper's maintenance passthrough still needs the exact
subcommand enumeration (config, update, models, gc, stats, token,
completions, plugin, install, setup, acp, share, join, collab, render,
export…).
