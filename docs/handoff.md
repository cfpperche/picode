# Handoff — living project state

> Read this first. At most 100 lines and 8 KB; the pre-commit hook refuses more, and refuses this file committed directly on `main` (ADR-0105).
> Shipped work: `git log`, `docs/decisions/README.md`, `docs/changelog.d/` + `CHANGELOG.md`. Session notes: `docs/handoff/` (newest by filename). Deploy history: `~/.picode/var/deploy-log.jsonl`.

## In flight

`git worktree list` is the truth for branches on disk (`make close-summary` prints it); this section holds only what merging one of them must know.

- `feat/herdr-validation`, `feat/picode-video-pilot` — carry `CHANGELOG.md` edits and the pre-ADR-0105 12 KB `docs/handoff.md`. Their next merge of `main` conflicts in both one last time: keep this file's shape and stay under 8 KB; any further changelog line goes to `docs/changelog.d/`.
- `feat/docs-adversarial` — docs-vs-code sweep fixes + `picode --version`; fragment `docs/changelog.d/docs-adversarial.md`; ff-ready after one more `make close` for the handoff note.
- `feat/fix-default-context-menu` — one-line web fix (default context menu rows match the terminal menu); ff-ready, fragment `docs/changelog.d/fix-default-context-menu.md`.
- `feat/apps-native-surface` — Matrix phase 1: ADR-0109 (manifest `surface`, native surface kind), hidden `demo-native` app, `suspendTermSocket`, ShellTerm pane re-claim; fragment `docs/changelog.d/apps-native-surface.md`. Merging beside `feat/matrix-store` (ADR-0108): both append a row to `docs/decisions/README.md` — keep both rows; no shared source files.

## Next up

1. First release since 0.1.0: `make changelog` on `main`, then `docs/release-process.md` (`[Unreleased]` is 860 lines).
2. Matrix app (plan `docs/plans/matrix-app.md`, phase 0 study `docs/benchmarks/2026-09-09-matrix-live-grid.md`): phase 1 `feat/apps-native-surface` (ADR-0109) ff-ready, phase 2 `feat/matrix-store` (ADR-0108) in flight; phase 3 — the Matrix surface, registered as `matrix` in `web/desktop/src/lib/nativeApps.js` — after both.
3. GitHub CI: the next push exercises the ADR-0105 workflow (Ubuntu-only Go matrix, tmux cache); macOS/Windows run on tags or `workflow_dispatch`.
4. llama delivery 3 live validation; owned-service ARM64 acceptance.
5. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
6. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
7. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
8. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
9. Compaction re-dogfood; historical Inbox rows; remote-mode and browser-preview (owner infra).
10. Windows clean-machine install (ADR-0098): phase 1 = `install-picode` + `install-runtime` stages in `picode-desktop.exe`; phase 2 = `install.ps1` one-liner + winget experiment, no paid signing. Plan: `docs/plans/windows-clean-install.md`.
11. Inspector debts (ADR-0096): `git ls-files` search, per-anchor watch, `+N −M` footer.

## Known debts / open questions

- Process (ADR-0105): worktrees start with a cold Go test cache (results are keyed by directory); `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K) — owner declined config changes 2026-09-09; capture tolerance is a 128 px budget (`scripts/docs-shots.mjs` prints the count per surface; lower it if a real change ever slips through).
- Communication (ADR-0104/0106): physical-device acceptance remains untested; Grok/Hermes automatic setup remains unavailable. Native session discovery is best effort; never share credentials across conversations. Orphan private setup files after owner deletion need maintenance cleanup; revoked files cannot authenticate. OpenCode inline merging requires JSON objects, not JSONC.
- Native packages/providers: real downloads, vendor OAuth, real credential changes and device acceptance remain external; mobile package configuration is desktop-only. Decision tables in `docs/plans/cli-native-packages.md` and `cli-native-providers.md`.
- Native settings: physical iPhone/PWA/IME and a real process restart remain external acceptance; scratch browser tests cover recovery, retained drafts and stopped-agent saves.
- Inbox terminal replies: pi-inbox 0.1.x items (`pi (unmanaged)`) have no address until each pi session updates; daemon death between park and JSONL row is an accepted gap.
- Hermes: live Working→Ready and needs-you confirmed 2026-09-06; `cli-v1-*` screenshots not regenerated; may write `shell-hooks-allowlist.json`. OpenCode live Working/Needs you unproven.
- Handoff (ADR-0088/0094): upstream formats undocumented (bump = refused write); Codex/Grok list a handed-off session only after a restart or by id; Hermes' importer flattens tool calls.
- CLI pane-death signal chain unproven; ADR-0085 instruments it. ADR-0084 pins nothing for terminals stopped before it.
- `Runtime.Stop` start-lease race: a stop during an in-flight managed start returns true without stopping. Windows `Close` kills only the direct child.
- `internal/server` tests swap package-level probes, so `t.Parallel` would race; `scripts/go-test.sh` shards them by process instead.
- `.git` is ~530 MB: UI bundles were committed 339 times and tutorial MP4s re-rendered; a history rewrite is the owner's call.
- Capture integration (ADR-0054/browser preview): no real emitter-to-RPC run, no slow-consumer/cancellation matrix; hub drops on overflow.
- Webhooks are at-least-once within event retention; receivers dedupe by id.
- Task Scheduler retries are not crash recovery; battery/sleep/sign-in acceptance is owner-controlled.
- `TestTerminalBrowse` cleanup can leave tmux shells in deleted temp folders.
- 2026-09-06 incident: a `tmux ls | grep '^picode-'` sweep killed 29 sessions, six in production. Never kill by prefix — only exact names from a fixture's own API.
- Feed: ephemeral events can be missed across reconnects (ADR-0048); paste fallback acceptance across platforms open.
- CLI lifecycle: npm data can lag native Claude releases by hours; grok uninstall guided-only; Windows paths out of scope.
- Pi has one active credential slot; per-agent OAuth is an owner decision.
- Rename watch: a branch adding files under `www/` would resurrect the dir — they belong in `docs-site/`.
- Tutorial video freshness audits are stale after source relocation; recapture is explicit. Branch protection and CODEOWNERS need the owner; desktop requests `/desktop/favicon.svg` and gets 404.
- Inspector: Files filter covers loaded rows only; This-agent chips show only with the agent's tab selected; `gh pr view` answers cached a minute.
- llama: ARM64 hardware, GPU / non-b10809 cancellation unverified; an unknown download with an absent model keeps its reservation; history pruning deferred.
- Mobile v2 (ADR-0095): physical IME/PWA/push/resume and microphone acceptance open; file writes keep the lexical/symlink and non-atomic mtime limits; iOS standalone strip on-device confirmation pending (Preferences → Layout exposes the dials).
- Notices (2026-09-07): needs-you covers the whole fleet, but the finish card only fires for the agent whose socket is open; neither exercised against a real pi dialog.
- Native surfaces (ADR-0109): `host` has no `openTerminal` — the demo POSTs `/api/terminals/{id}/open` itself and keeps the live record locally (feed rows carry no `session`); closing a terminal's tab while a native app shows its pane disposes the xterm (`closeShellTerm`) and the app body stays blank until it remounts — phase 3's ownership rule (plan §4.5) decides both. The desktop's unsupported tile still explains itself only through `title` (pre-existing pattern).
