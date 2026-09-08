# Handoff — living project state

> Read this first. At most 100 lines (the pre-commit hook refuses more).
> Session notes: `docs/handoff/` (newest by filename). Deploy history:
> `~/.picode/var/deploy-log.jsonl` and `git log`. Older prose: `docs/handoff-archive.md`.

## Current state

- **Mobile v2:** focused screens, retained drafts, Sessions/Automations, Files/editor and Git workflows (ADR-0095); acceptance in `docs/plans/mobile-v2.md`.
- **Process (ADR-0086, 2026-09-06):** `picode deploy` refuses while any agent
  or terminal is mid-turn (`GET /api/deploy/readiness`, loopback); `main`
  ships in batches (`make deploy-batch`, `picode-deploy.timer` at
  12:00/18:00/23:00 — `make timers` installs it). A branch closes with
  `make close`; iterate with `make ci-scoped`; `make ci` runs once on `main`
  at the merge. `make worktree NAME=x` / `make worktree-gc`. Capture parity
  is advisory in `make ci`, strict in `close` and the batch.
- **Terminals:** CLI session pin + one-click resume (ADR-0084); flight
  recorder, SIGHUP-immune pane roots and the deploy log (ADR-0085). A dropped
  terminal WebSocket (phone lock, network) reattaches by itself — same xterm,
  scroll position kept (`web/shared/client/termSocket.js`). Hermes
  Agent is a fifth Agent CLI (catalog, sessions, PYTHONPATH activity hooks,
  no `HERMES_HOME` overlay), deployed `0.1.0+405fed1`. OpenCode is a sixth
  (ADR-0088), deployed `0.1.0+d65e9a1`; Activity default-on + start
  banner on `main` as `06da6771` (not yet deployed). Terminal checklists (ADR-0081)
  on the sidebar card only; absent checklist is silence (ADR-0092). Sessions live under Agent CLIs (ADR-0079).
  A pane answers a right-click with PiCode's menu (`lib/termMenu.js`; Shift still
  gives the browser's, the press no longer reaches tmux); the ADR-0089 message bar
  opens from it seeded with the selection, and Find (Ctrl+Shift+F,
  `@xterm/addon-search`) floats over the pane without resizing it.
- **CLI lifecycle (ADR-0087/0093):** Agent CLIs shows update badges (npm
  registry or vendor `--check`) and runs each CLI's own update/reinstall/
  uninstall/install as a durable `cli_jobs` lane with streamed output,
  terminal guards and typed uninstall confirmation; interrupted jobs never
  replay. Unmanageable installs (Homebrew, manual checkouts) get docs
  links, not controls. Detect fix (symlinks/wrappers) makes real installs
  classify; E2E in production: hermes 0.18.2 -> 0.21.0 through the surface.
  ADR-0093 adds install for missing npm-backed CLIs; grok/hermes guided.
- **Inspector rail (ADR-0078):** Changes, Files, PR tab, Git actions
  (prepare, run-when-idle, "Ask <agent>" through the agent's own channel).
- **llama.cpp manager:** deliveries 1–2 deployed (ADR-0080/0083); delivery 4
  (`#/llama/service`, ADR-0090) as `0.1.0+c694fb2`. On main: old-installation
  cleanup and empty setup recovery. Delivery 3 guidance remains separate.
- Also on `main`: File Tree v2 (0074), Git Graph per worktree (0073),
  independent desktop/mobile apps (0072), Windows task reliability (0071),
  Agent CLIs v2 (0069), Integrations (0075), Docker v3, identity favicons
  at 16 px, tab strip overflow phases 2–4. Extra keys first-open `0.1.0+dfa9f7b`.
  ADR-0089 attach bar `0.1.0+aca6628`, amended: the staging folder no longer touches the project's own `.gitignore` (a nested one instead) and ages out after 7 days. GitHub repo picker in the clone form (`0.1.0+a2e699c`, accepted live).
  Workspace card: two actions instead of five, Files/Git graph reading through the workspace even with no agents (ADR-0027/0030), header stayed one line (owner call, same day).
  Managed agents remain Pi-only; guests stay terminals-only (ADR-0091) until a protocol converges.
  **Desktop user menu v2:** grouped rows with subtitles + in-menu search (mobile More
  pattern, shared matcher `@picode/shared/domain/listSearch.js`); theme/layout radios stay.

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
8. Git graph write actions (ADR-0096): all four phases shipped, reviewed adversarially, and the ask door reaches pi terminals (ADR-0089 amendment, proven live) — thirty-odd actions in risk tiers through ADR-0078's three doors, server-side composer (`internal/gitcmd`), typed confirmation for tier C by the run door, worktree+agent in one gesture, undo where an honest inverse exists. Inspector debts left: `git ls-files` search, per-anchor watch, `+N −M` footer.

## Known debts / open questions

- Hermes: live TUI Working→Ready and needs-you confirmed 2026-09-06; `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Working/Needs you unproven (Activity was off on first deploy).
- Handoff (ADR-0088/0094): visual pass done 2026-09-07, every state rendered including the live-source warning; upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes' importer flattens tool calls.
- CLI pane-death signal chain unproven; ADR-0085 instruments it — the next deploy
  that loses sessions is the experiment. ADR-0084 pins nothing for terminals stopped before it (Sessions → "Open in terminal").
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
- Tutorial video freshness audits are stale after source relocation; recapture is explicit. Branch protection and CODEOWNERS need the owner; desktop requests `/desktop/favicon.svg` and gets 404.
- Inspector: Files filter covers loaded rows only; This-agent chips show only
  with the agent's tab selected; `gh pr view` answers cached a minute.
- llama: ARM64 hardware, GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance remain open; file writes retain the existing lexical/symlink and non-atomic mtime limits. iOS standalone: strip workaround needs a real-device re-check after deploy; the `?strip-probe=1` test decides whether buttons can descend into the strip.
- Notices (2026-09-07): needs-you covers the whole fleet (`agent.state`), but the *finish* card still only fires for the agent whose socket is open — a background agent's completion arrives via Inbox + phone. Neither card has been exercised against a real pi dialog; both were staged at the HTTP boundary.
