# Handoff — living project state

> Read this first. At most 100 lines (the pre-commit hook refuses more).
> Session notes: `docs/handoff/` (newest by filename). Deploy history: `~/.picode/var/deploy-log.jsonl` and `git log`. Older prose: `docs/handoff-archive.md`.

## Current state

- **Native CLI settings (ADR-0101):** Settings now lives at `#/clis/settings/pi`; old links redirect, agent URLs preserve scope, desktop/mobile editors keep native Pi APIs and the mobile quick sheet. Recovery preserves drafts, blocks stale writes and reports failed restarts; malformed defaults no longer block mobile agent controls. Settings, Sessions and the new-terminal form share a `--ctl-h` CLI combobox with each runtime's favicon.
- **Mobile v2:** focused screens, retained drafts, Sessions/Automations, Files/editor and Git workflows (ADR-0095); acceptance in `docs/plans/mobile-v2.md`. Work empty (Agents/Terminals/Workspaces) is one centered line + primary create; search misses stay top-aligned. Workspace favicons share the 22px row-mark size.
- **Process (ADR-0086, 2026-09-06):** `picode deploy` refuses mid-turn
  (`GET /api/deploy/readiness`); `main` ships in batches (`make deploy-batch`,
  timer 12:00/18:00/23:00 — `make timers`). Iterate `make ci-scoped`; close with `make close`; `make ci` once on `main` at the merge. `make worktree NAME=x` / `make worktree-gc`. Capture parity advisory in `make ci`, strict in `close` and the batch.
- **Docs site:** `docs-site/` (renamed from `www/` 2026-09-08, owner call): same VitePress build and Pages URL; `www/` reserved for the product website. Make var `DOCS_STAMP`; ADR-0086 emended ("née www/").
- **Terminals:** CLI session pin + one-click resume (ADR-0084); flight
  recorder, SIGHUP-immune pane roots and the deploy log (ADR-0085). A dropped
  terminal WebSocket (phone lock, network) reattaches by itself — same xterm,
  scroll position kept (`web/shared/client/termSocket.js`). Hermes
  Agent is a fifth Agent CLI (catalog, sessions, PYTHONPATH activity hooks,
  no `HERMES_HOME` overlay), deployed `0.1.0+405fed1`. OpenCode is a sixth
  (ADR-0088), deployed `0.1.0+d65e9a1`; Activity default-on + start
  banner on `main` as `06da6771` (not yet deployed). Terminal checklists (ADR-0081)
  on the sidebar card only; absent checklist is silence (ADR-0092). Sessions live under Agent CLIs (ADR-0079).
  Right-click gives PiCode's pane menu (`lib/termMenu.js`); the ADR-0089
  message bar opens from it seeded with the selection; Find (Ctrl+Shift+F,
  `@xterm/addon-search`) floats without resizing the pane.
- **Inbox:** replies to `ask_human` from pi in an Agent CLI terminal reach that terminal's receiver and exact session (ADR-0037 amendment 2026-09-09); channelless blocking questions refuse visibly. Needs pi-inbox 0.2.0 per pi session.
- **CLI lifecycle (ADR-0087/0093):** Agent CLIs shows update badges (npm
  registry or vendor `--check`) and runs each CLI's own update/reinstall/
  uninstall/install as a durable `cli_jobs` lane with streamed output,
  terminal guards and typed uninstall confirmation; interrupted jobs never
  replay. Unmanageable installs (Homebrew, manual checkouts) get docs links. Detect fix classifies real installs; E2E proven live (hermes 0.18.2→0.21.0); ADR-0093 adds install for missing npm-backed CLIs, grok/hermes guided.
- **Inspector rail (ADR-0078):** Changes, Files, PR tab, Git actions
  (prepare, run-when-idle, "Ask <agent>" through the agent's own channel).
- **llama.cpp manager:** deliveries 1–2 + 4 deployed (ADR-0080/0083/0090;
  `#/llama/service` as `0.1.0+c694fb2`). On main: old-installation cleanup
  and empty setup recovery; delivery 3 guidance separate.
- **Git graph listing +/-:** every commit row shows its own diff's line total (`+N −M`, first-parent, merges included; reconciles with the commit detail); hides ≤900px.
- Also on `main`: File Tree v2 (0074), Git Graph per worktree (0073),
  independent desktop/mobile apps (0072), Windows task reliability (0071),
  Agent CLIs v2 (0069), Integrations (0075), Docker v3, identity favicons
  at 16 px, tab strip overflow phases 2–4. Extra keys first-open `0.1.0+dfa9f7b`. ADR-0089 attach bar `0.1.0+aca6628`, amended: the staging folder no longer touches the project's own `.gitignore` (a nested one instead) and ages out after 7 days. GitHub repo picker in the clone form (`0.1.0+a2e699c`, accepted live). Workspace card: two actions instead of five (ADR-0027/0030), header stayed one line.
  Managed agents remain Pi-only; guests stay terminals-only (ADR-0091) until a protocol converges. **Desktop user menu v2:** grouped rows with subtitles + in-menu search (`@picode/shared/domain/listSearch.js`); theme/layout radios stay.
- **Agent CLIs layout:** every tab uses its app-owned `AgentClisFrame`: one 1240px desktop maximum, full mobile width, consistent card padding and stable tab alignment. Desktop reserves scrollbar space; narrow Sessions and CLI setup actions wrap.
- **Native CLI packages (ADR-0102/0099):** Agent CLIs → Packages at `#/clis/packages/pi`; legacy links redirect with explicit workspace/agent/scope context. Pi APIs and native files remain authoritative. Failed refreshes retain drafts and block writes; changed file targets require confirmed reload. Desktop pi-roles configuration retains independent workspace/agent drafts, scoped clear and malformed-file recovery. Mobile config links offer the desktop layout. Next adapters: pi-compact and a declarative manifest.

## In flight (unmerged branches on disk)

- `feat/picode-feature-video` — skills record clicks and typing, not slideshows.

## Next up

1. First batch deploy (timer 23:00) is unguarded; later ones refuse mid-turn.
2. llama delivery 3 live validation; owned-service ARM64 acceptance.
3. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
4. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
5. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
6. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
7. Compaction re-dogfood; historical Inbox rows; remote-mode and browser-preview (owner infra).
9. Windows clean-machine install (ADR-0098, accepted 2026-09-08): phase 1 = `install-picode` + `install-runtime` bootstrap stages in `picode-desktop.exe`; phase 2 = `install.ps1` one-liner + winget experiment, no paid signing (Trusted Signing excludes Brazil; SignPath needs OSI). Plan: `docs/plans/windows-clean-install.md`.
8. Git graph write actions (ADR-0096) fully shipped — the ask door reaches
   pi terminals (ADR-0089 amendment, proven live). Inspector debts left:
   `git ls-files` search, per-anchor watch, `+N −M` footer.

## Known debts / open questions

- Native packages: real package-manager downloads and physical-device acceptance remain external; mobile configuration editing is still desktop-only. Fixture/browser coverage includes all rows in `docs/plans/cli-native-packages.md`.
- Native settings: physical iPhone/PWA/IME and real process restart remain external acceptance; both app adapters have failure coverage, and scratch browser tests cover recovery, retained drafts and stopped-agent saves.
- Inbox terminal replies: pi-inbox 0.1.x items (`pi (unmanaged)`) have no
  address — answered by hand until updated per pi session; daemon death
  between park and JSONL row = accepted gap (as the terminal ask).
- Hermes: live TUI Working→Ready and needs-you confirmed 2026-09-06; `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Working/Needs you unproven (Activity was off on first deploy).- Handoff (ADR-0088/0094): visual pass done 2026-09-07, every state rendered including the live-source warning; upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes' importer flattens tool calls.
- CLI pane-death signal chain unproven; ADR-0085 instruments it — the next deploy that loses sessions is the experiment. ADR-0084 pins nothing for terminals stopped before it (Sessions → "Open in terminal").
- `Runtime.Stop` start-lease race: a stop during an in-flight managed start
  returns true without stopping. Windows `Close` kills only the direct child.
- `internal/server` tests run serially (~55 s); they swap package-level
  probes, so `t.Parallel` would race. Scoping avoids the suite for non-Go diffs.
- GitHub CI matrix was red 142/157 runs on two macOS assumptions (fixed
  2026-09-06); watch the next runs before trusting green again.
- `.git` is 567 MB: UI bundles were committed 339 times and tutorial MP4s
  re-rendered; a history rewrite is the owner's call.
- Capture integration (ADR-0054/browser preview): no real emitter-to-RPC run,
  no slow-consumer/cancellation matrix; hub drops on overflow.
- Webhooks are at-least-once within event retention; receivers dedupe by id.
- Task Scheduler retries are not crash recovery (exit-one probe stayed down
  90 s); battery/sleep/sign-in acceptance is owner-controlled.
- `TestTerminalBrowse` cleanup can leave tmux shells in deleted temp folders.
- 2026-09-06 incident: a `tmux ls | grep '^picode-'` sweep killed 29 sessions,
  six of them production. Never kill by prefix — only exact names from a fixture's own API.
- Feed: ephemeral events can be missed across reconnects (ADR-0048); paste
  fallback acceptance across platforms open.
- CLI lifecycle: npm data can lag native Claude releases by hours (the badge
  names the source); grok uninstall guided-only; Windows paths out of scope.
- Pi has one active credential slot; per-agent OAuth is an owner decision.
- Rename watch: a branch adding files under `www/` (pre-rename base) would resurrect the dir — they belong in `docs-site/` (open branches touch none, checked 2026-09-08).
- Tutorial video freshness audits are stale after source relocation; recapture is explicit. Branch protection and CODEOWNERS need the owner; desktop requests `/desktop/favicon.svg` and gets 404.
- Inspector: Files filter covers loaded rows only; This-agent chips show only
  with the agent's tab selected; `gh pr view` answers cached a minute.
- llama: ARM64 hardware, GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance remain open; file writes retain the existing lexical/symlink and non-atomic mtime limits. iOS standalone strip: on-device confirmation pending; Preferences → Layout (Auto/Low/Screen edge) exposes the dials so the owner tunes without a code change.
- Notices (2026-09-07): needs-you covers the whole fleet (`agent.state`), but the *finish* card only fires for the agent whose socket is open. Neither card has been exercised against a real pi dialog; both were staged at the HTTP boundary.
