# Handoff — living project state

> Read this first. At most 100 lines and 8 KB; the pre-commit hook refuses more, and refuses this file committed directly on `main` (ADR-0105).
> Shipped work: `git log`, `docs/decisions/README.md`, `docs/changelog.d/` + `CHANGELOG.md`. Session notes: `docs/handoff/` (newest by filename). Deploy history: `~/.picode/var/deploy-log.jsonl`.

## In flight

`git worktree list` is the truth for in-flight branches; entries say what a merge must know.

- `feat/canvas-chrome` — the Canvas surface has no `.ft-head`: the plane fills the stage and the chrome floats as `.cv-cluster` (canvas.css) — switcher, Add panel, a `⋯` menu, and the zoom cluster. **Session two continues on this branch** (background pattern, light palette): `.cv-cluster` is the only ground it has to change. `ci-scoped` green, unmerged, not deployed.
- `feat/herdr-validation`, `feat/picode-video-pilot` — carry pre-ADR-0105 `CHANGELOG.md` + 12 KB handoff; next `main` merge conflicts in both: keep this file's shape, changelog lines to `docs/changelog.d/`.
## Next up

1. Runbook step 6 (0.2.0, tag `v0.2.0`): watch the owner's window; a regression becomes a patch tag, never a rewritten one.
2. Dashboard throughput (tokens/s): pick the definition (generation vs turn; reasoning in/out) and the per-CLI coverage, then the UI gates. Codex's `duration_ms`/`time_to_first_token_ms` are still unread by its meter; Grok's timings shipped 2026-09-11.
2. Canvas v1.1 (`docs/plans/matrix-app.md` §5 phase 5): quick reply line in a chat panel, frozen-frame placeholders, drag from the sidebar, inspector follows focus, tile badge. v1 phases 0–4 are done; v2 is done through C4, and C5 (group nodes, one layout library) needs its own plan.
3. llama delivery 3 live validation; owned-service ARM64 acceptance.
4. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
5. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
6. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
7. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
8. Compaction re-dogfood; historical Inbox rows; remote-mode (owner infra).
9. Windows clean-machine install (ADR-0098): phase 1 = `install-picode` + `install-runtime` stages in `picode-desktop.exe`; phase 2 = `install.ps1` one-liner + winget, no paid signing. Plan: `docs/plans/windows-clean-install.md`.

## Known debts / open questions

- Process (ADR-0105): worktrees start with a cold Go test cache; `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px (`scripts/docs-shots.mjs` prints it per surface).
- Communication (ADR-0104/0106/0107/0111): physical-mobile and non-Linux pane/process recovery, and custom Codex resume global args/`--`, unverified. PTY rechecks cannot remove the check-to-write race; uncertain attempts never auto-retry. Orphan private setup files need cleanup after owner deletion.
- Codex resume (2026-09-11): a live sub-agent hook payload was never captured — the filter's field names are inferred from the 0.154 binary roster; a pin that is already a sub-agent persists until the terminal's next native session (one-off repair: `codex resume 01a08246-98e1-7bb0-bb5d-e2c472367623`).
- Communication onboarding (ADR-0110/0111): unobserved conversations need a first native event; six-CLI rerun and long wrapped OpenCode footers pending. Partial rows 5, 9, 12 in `docs/plans/communication-onboarding.md`: moved-owner consent, missing-adapter repair, stubborn-child timeout.
- Native packages/providers/settings: real downloads, vendor OAuth, credential changes, device acceptance, physical iPhone/PWA/IME and a real process restart remain external; mobile package configuration is desktop-only (`docs/plans/cli-native-packages.md`, `cli-native-providers.md`).
- Inbox terminal replies: pi-inbox 0.1.x items (`pi (unmanaged)`) have no address until each pi session updates; daemon death between park and JSONL row is accepted.
- Hermes: `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Needs-you unverified.
- Handoff (ADR-0088/0094): upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes flattens tool calls.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; cleaned by exact PID. Signal-chain repair is separate; ADR-0084 pins nothing for terminals stopped before it.
- Windows `Close` kills only its direct child. `internal/server` tests swap package-level probes, so `t.Parallel` would race; `scripts/go-test.sh` shards by process.
- `.git` is ~530 MB (UI bundles committed 339 times, MP4s re-rendered); a history rewrite is the owner's call.
- Capture integration (ADR-0054): no real emitter-to-RPC run, no slow-consumer/cancellation matrix; the hub drops on overflow.
- Webhooks are at-least-once within event retention; receivers dedupe by id.
- Task Scheduler retries are not crash recovery; battery/sleep/sign-in acceptance is the owner's.
- tmux: never kill by prefix (a `grep '^picode-'` sweep killed 29 sessions, six in production, 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells (`picode-sh-shell-6d4d44` is kept on purpose).
- Feed: ephemeral events can be missed across reconnects (ADR-0048); cross-platform paste fallback open.
- Terminal menus (2026-09-09): web/mobile terminal rows still offer only Remove — the desktop one-menu merge (termRowMenu.js) is not ported; sidebar Remove keeps `DELETE /api/terminals/<id>`, Agent CLIs `/launch/remove` (same outcome, two paths).
- CLI lifecycle: npm data can lag native Claude releases by hours; grok uninstall is guided-only; Windows paths out of scope.
- Pi has one active credential slot; per-agent OAuth is the owner's.
- Tutorial video freshness audits are stale after source relocation. Branch protection and CODEOWNERS need the owner; desktop asks for `/desktop/favicon.svg` and gets 404.
- Inspector: the Files filter covers loaded rows only; This-agent chips need the agent's tab selected; `gh pr view` answers cache a minute. Debts (ADR-0096): `git ls-files` search, per-anchor watch.
- llama: ARM64 hardware and GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance open; file writes keep the lexical/symlink and non-atomic mtime limits; iOS standalone strip needs on-device confirmation.
- Canvas (ADR-0108/0118): no docs-shots capture — needs a `desktop-canvas` profile and a fixture, so the guide has no screenshots. App-wide, from its captures: the toast covers a surface's Close, dialogs have no scrim and little dark elevation, a grow-resize leaves an idle cursor.
- Canvas v2 (C2–C4, 2026-09-10): `+`/`-`/`0` need a focused panel; a still stamps its age only when the feed moved; only the picker adds a `file`/`diff` panel (ADR-0109). Edges (ADR-0116): no keyboard path to *draw* a link, so the Messages audit list is the keyboard way to read and revoke, at one canvas read per canvas (capped at 50).
- Fullscreen (2026-09-10): real browser fullscreen owns Escape, so one press leaves the mode; double-Escape is in-app only. The right strip is pointer-transparent, so its reveal misses an embedded frame (PDF preview). Chromium only.
- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Canvas POSTs `/api/terminals/{id}/open` itself; a tab closing under a panel remounts the body with a fresh xterm.
- Grok (2026-09-11): tokens/cost are `partial` by design (`usage.json` is new in 1.0.x); `clisession.GrokSource` still lists prompt history only, so Agent CLIs rows show no model/cost that `summary.json`/`usage.json` hold; the new dashboard values were not screenshot-verified.
