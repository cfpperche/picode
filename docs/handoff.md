# Handoff — living project state

> Read this first. At most 100 lines and 8 KB; the pre-commit hook refuses more, and refuses this file committed directly on `main` (ADR-0105).
> Shipped work: `git log`, `docs/decisions/README.md`, `docs/changelog.d/` + `CHANGELOG.md`. Session notes: `docs/handoff/` (newest by filename). Deploy history: `~/.picode/var/deploy-log.jsonl`.

## In flight

`git worktree list` is the truth for branches on disk (`make close-summary` prints it); this section holds only what merging one of them must know.

- `feat/herdr-validation`, `feat/picode-video-pilot` — carry `CHANGELOG.md` edits and the pre-ADR-0105 12 KB `docs/handoff.md`; their next merge of `main` conflicts in both: keep this file's shape, under 8 KB, changelog lines to `docs/changelog.d/`.
- `feat/term-key-capture` — fullscreen locks the keyboard (Chromium): `Ctrl+T`/`Ctrl+W` now reach the guest CLIs; ff-ready.
- `feat/matrix-surface` (plan phase 3, both sessions): surface, chunk loading, picker, save/409, keyboard, maximize, guide page and the §7 browser QA are committed and measured (`docs/handoff/2026-09-09-matrix-surface.md`).

## Next up

1. First release since 0.1.0: `make changelog` on `main`, then `docs/release-process.md` (`[Unreleased]` is 860 lines).
2. Matrix phase 4 (`docs/plans/matrix-app.md` §5): desktop `useAgentSocket`, read-only conversation body for managed agents, Needs-you chip; then the v1.1 list. Matrix v2 canvas: C0 measured, GO; the owner's §8 call gates C1 (`docs/plans/matrix-canvas.md`).
3. GitHub CI: the next push exercises the ADR-0105 workflow (Ubuntu-only Go matrix, tmux cache); macOS/Windows run on tags or `workflow_dispatch`.
4. llama delivery 3 live validation; owned-service ARM64 acceptance.
5. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
6. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
7. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
8. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
9. Compaction re-dogfood; historical Inbox rows; remote-mode and browser preview (owner infra).
10. Windows clean-machine install (ADR-0098): phase 1 = `install-picode` + `install-runtime` stages in `picode-desktop.exe`; phase 2 = `install.ps1` one-liner + winget, no paid signing. Plan: `docs/plans/windows-clean-install.md`.
11. Inspector debts (ADR-0096): `git ls-files` search, per-anchor watch, `+N −M` footer.

## Known debts / open questions

- Process (ADR-0105): worktrees start with a cold Go test cache (keyed by directory); `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K) — owner declined config changes 2026-09-09; capture tolerance is 128 px (`scripts/docs-shots.mjs` prints it per surface).
- Communication (ADR-0104/0106/0107/0111/0112): physical mobile, non-Linux recovery and custom Codex resume global args/`--` unverified. PTY race remains; uncertain attempts, including Grok paste refusals, never auto-retry. Never share conversation credentials. Clean orphan private setup files after owner deletion.
- Communication onboarding (ADR-0110/0111/0112): needs a first native event; wrapped OpenCode sidebars/footers block input. Pi needs reload for its new heartbeat; Hermes needs stop/resume. Rows 5, 9, 12 in `docs/plans/communication-onboarding.md` remain open: moved-owner consent, missing-adapter repair, stubborn-child timeout.
- Native packages/providers: real downloads, vendor OAuth, credential changes and device acceptance remain external; mobile package configuration is desktop-only (`docs/plans/cli-native-packages.md`, `cli-native-providers.md`).
- Native settings: physical iPhone/PWA/IME and a real process restart remain external; scratch tests cover recovery, retained drafts and stopped-agent saves.
- Inbox terminal replies: pi-inbox 0.1.x items (`pi (unmanaged)`) have no address until each pi session updates; daemon death between park and JSONL row is accepted.
- Hermes: live Working→Ready and needs-you confirmed 2026-09-06; `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Needs-you unverified.
- Handoff (ADR-0088/0094): upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes flattens tool calls.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; cleaned by exact PID. Signal-chain repair is separate; ADR-0084 pins nothing for terminals stopped before it.
- Windows `Close` kills only its direct child.
- `internal/server` tests swap package-level probes, so `t.Parallel` would race; `scripts/go-test.sh` shards by process.
- `.git` is ~530 MB (UI bundles committed 339 times, MP4s re-rendered); a history rewrite is the owner's call.
- Capture integration (ADR-0054/browser preview): no real emitter-to-RPC run, no slow-consumer/cancellation matrix; the hub drops on overflow.
- Webhooks are at-least-once within event retention; receivers dedupe by id.
- Task Scheduler retries are not crash recovery; battery/sleep/sign-in acceptance is the owner's.
- Never kill tmux by prefix: a `grep '^picode-'` sweep killed 29 sessions, six in production (2026-09-06). Exact names from a fixture's API only.
- Orphan tmux shells (2026-09-09): the `t.Context()` cleanups and `qa-scratch stop` are fixed; a scratch whose daemon dies before `stop` still strands its shells; `picode-sh-shell-6d4d44` is kept on purpose.
- Feed: ephemeral events can be missed across reconnects (ADR-0048); cross-platform paste fallback open.
- Terminal menus (2026-09-09): web/mobile terminal rows still offer only Remove — the desktop one-menu merge (termRowMenu.js) is not ported; sidebar Remove keeps `DELETE /api/terminals/<id>`, Agent CLIs `/launch/remove` (same outcome, two paths).
- CLI lifecycle: npm data can lag native Claude releases by hours; grok uninstall is guided-only; Windows paths out of scope.
- Pi has one active credential slot; per-agent OAuth is the owner's.
- Rename watch: a branch adding files under `www/` resurrects it; they belong in `docs-site/`.
- Tutorial video freshness audits are stale after source relocation; recapture is explicit. Branch protection and CODEOWNERS need the owner; desktop asks for `/desktop/favicon.svg` and gets 404.
- Inspector: the Files filter covers loaded rows only; This-agent chips need the agent's tab selected; `gh pr view` answers cache a minute.
- llama: ARM64 hardware and GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance open; file writes keep the lexical/symlink and non-atomic mtime limits; iOS standalone strip needs on-device confirmation (Preferences → Layout).
- Matrix (ADR-0108/0109): no docs-shots capture — needs a `desktop-matrix` profile and a fixture. Attaches are bounded by scroll *speed*, not distance (300 ms dwell). App-wide, from its captures: the toast covers a surface's Close; dialogs have no scrim and little dark elevation; a grow-resize leaves an idle cursor.
- Matrix v2 (C0): xterm maps a pointer as `cell × zoom` under a CSS transform — a canvas panel takes a mouse only at zoom 1.0; `onlyRenderVisibleElements` stays off (socket churn); a 20-panel drag ran 31 fps; agent panels and maximize untested.
- Fullscreen mode (2026-09-10): real browser fullscreen owns Escape, so one press leaves the mode; the double-Escape is in-app-only. The right strip is pointer-transparent, so its reveal misses an embedded frame (PDF preview). Chromium only.
- Notices (2026-09-07): needs-you covers the fleet, but the finish card only fires for the agent whose socket is open; neither exercised against a real pi dialog.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Matrix POSTs `/api/terminals/{id}/open` itself (feed rows carry no `session`); a tab closing under a panel remounts the body with a fresh xterm. An unsupported tile explains itself only via `title`.
