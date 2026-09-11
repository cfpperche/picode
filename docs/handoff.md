# Handoff — living project state

> Read this first. ≤ 100 lines / 8 KB; shipped work is `git log`, the ADR index and the changelog, session notes live in `docs/handoff/`.

## In flight

`git worktree list` is the truth for in-flight branches; entries say what a merge must know.

- `feat/communication-recovery` — native state recovery and empty Grok suggestions; unmerged (a683eed7).
- `feat/floating-edges` — Canvas links anchor on the border facing the other panel, spread when one carries several, and the drag preview matches.
- `feat/herdr-validation`, `feat/picode-video-pilot` — pre-ADR-0105 `CHANGELOG.md` + 12 KB handoff; on merge keep this shape, changelog to `docs/changelog.d/`.

## Next up

1. Runbook step 6 (0.2.0, tag `v0.2.0`): watch the owner's window; a regression becomes a patch tag, never a rewritten one.
2. Dashboard throughput (tokens/s): definition (generation vs turn; reasoning in/out), per-CLI coverage, UI gates. Codex's `duration_ms`/`time_to_first_token_ms` still unread; Grok's timings shipped.
3. Package config descriptors (ADR-0119 accepted; C0–C5 shipped, GUI-validated; docs-site Packages page done) is feature-complete. Backlog (owner, 2026-09-11): upstream `picode.config` proposal. If ever requested: workspace-scope per-agent overlays. Plan: `docs/plans/package-config-manifest.md`.
2. Canvas v1.1 (`docs/plans/matrix-app.md` §5 phase 5): chat quick reply, frozen-frame placeholders, drag from the sidebar, inspector follows focus, tile badge. v1 0–4 and v2 through C4 done; C5 (group nodes, one layout library) needs its own plan.
3. llama delivery 3 live validation; owned-service ARM64 acceptance.
4. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
5. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
6. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
7. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
8. Compaction re-dogfood; historical Inbox rows; remote-mode (owner infra).
9. Windows clean-machine install (ADR-0098): phase 1 stages in `picode-desktop.exe`, phase 2 `install.ps1` one-liner + winget, no paid signing. Plan: `docs/plans/windows-clean-install.md`.
10. WSL control (P0/P1 shipped): P2 is the Storage app plus the Windows-facts route, which needs its own ADR; P3's actions — compact, `--set-sparse`, `--move` — cost the distro's sessions. Plan: `docs/plans/wsl-control.md`.

## Known debts / open questions

- Process (ADR-0105): worktrees start with a cold Go test cache; `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K); capture tolerance is 128 px.
- Communication (ADR-0104/0106/0107/0111): physical-mobile and non-Linux pane/process recovery, and custom Codex resume global args/`--`, unverified. PTY rechecks cannot remove the check-to-write race; uncertain attempts never auto-retry. Orphan private setup files need cleanup after owner deletion.
- Codex resume (2026-09-11): a live sub-agent hook payload was never captured — filter field names come from the 0.154 binary roster; a pin that is already a sub-agent persists until that terminal's next native session.
- Communication onboarding (ADR-0110/0111): unobserved conversations need a first native event; six-CLI rerun and long wrapped OpenCode footers pending. Partial rows 5, 9, 12 in `docs/plans/communication-onboarding.md`: moved-owner consent, missing-adapter repair, stubborn-child timeout.
- Native packages/providers/settings: real downloads, vendor OAuth, credential changes, device acceptance, physical iPhone/PWA/IME and a real process restart remain external; mobile package configuration is desktop-only (`docs/plans/cli-native-packages.md`).
- Inbox terminal replies: pi-inbox 0.1.x items (`pi (unmanaged)`) have no address until each pi session updates; daemon death between park and JSONL row is accepted.
- Hermes: `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Needs-you unverified.
- Handoff (ADR-0088/0094): upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes flattens tool calls.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed. Signal-chain repair is separate; ADR-0084 pins nothing for terminals stopped before it.
- Windows `Close` kills only its direct child. `internal/server` tests swap package-level probes, so `t.Parallel` would race; `scripts/go-test.sh` shards by process.
- `.git` is ~530 MB (UI bundles, MP4s); a history rewrite is the owner's call.
- Capture integration (ADR-0054): no real emitter-to-RPC run, no slow-consumer/cancellation matrix.
- Webhooks are at-least-once within event retention; receivers dedupe by id.
- tmux: never kill by prefix (a `grep '^picode-'` sweep killed 29 sessions, six in production, 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells (one is kept on purpose).
- WSL disk: the tray warns in words only (no alert icon asset); docker's storage is the Docker app's, not measured here.
- Feed: ephemeral events can be missed across reconnects (ADR-0048); cross-platform paste fallback open.
- Terminal menus (2026-09-09): web/mobile terminal rows still offer only Remove — the desktop one-menu merge (termRowMenu.js) is not ported.
- CLI lifecycle: npm data can lag native Claude releases by hours; grok uninstall is guided-only; Windows paths out of scope.
- Pi has one active credential slot; per-agent OAuth is the owner's.
- Tutorial video freshness audits are stale after source relocation. Branch protection and CODEOWNERS need the owner; desktop asks for `/desktop/favicon.svg` and gets 404.
- Inspector: the Files filter covers loaded rows only; This-agent chips need the agent's tab selected; `gh pr view` answers cache a minute. Debts (ADR-0096): `git ls-files` search, per-anchor watch.
- llama: ARM64 hardware and GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance open; file writes keep the lexical/symlink and non-atomic mtime limits; iOS standalone strip needs on-device confirmation.
- Canvas (ADR-0108/0118) and Agent CLIs: no docs-shots capture — Canvas needs a `desktop-canvas` profile and fixture; Agent CLIs is not in SURFACE_PROFILES. App-wide: the toast covers a surface's Close, a grow-resize leaves an idle cursor, Inspector branch chips truncate to `· fe… ·…`, a panel's Run reads as a disabled chip.
- Canvas v2 (C2–C4, 2026-09-10): `+`/`-`/`0` need a focused panel; a still ages only when the feed moved; only the picker adds a `file`/`diff` panel. Edges (ADR-0116): no keyboard path to *draw* a link — the Messages audit list reads and revokes (one read per canvas, capped at 50). Anchors (2026-09-11): the preview cannot show the spread (a crowded drop shifts the line one gap); below zoom ≈0.43 the 12 px grab band hides the header, so the panel cannot be dragged (pre-existing).
- Fullscreen (2026-09-11): the browser owns Escape (one press leaves); the right strip's pointer-transparent reveal misses an embedded frame (PDF); icon-only controls show no label without hover (no touch equivalent). Chromium only.
- Terminal scrollbars (2026-09-11): the web terminal draws none (a tmux client has no scrollback, `term-scrollbar.test.mjs`); the reader's bar is tmux's copy-mode or the TUI's own. A draggable browser bar means the tmux client off the alternate screen — a decision.
- Notices (2026-09-07): the finish card only fires for the agent whose socket is open.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the Canvas POSTs `/api/terminals/{id}/open` itself; a tab closing under a panel remounts the body with a fresh xterm.
- Grok (2026-09-11): tokens/cost `partial` by design (`usage.json` is new); Agent CLIs rows show model/title but not its cost; dashboard values not screenshot-verified.
