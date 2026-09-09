# Handoff — living project state

> Read this first. At most 100 lines and 8 KB; the pre-commit hook refuses more, and refuses this file committed directly on `main` (ADR-0105).
> Shipped work: `git log`, `docs/decisions/README.md`, `docs/changelog.d/` + `CHANGELOG.md`. Session notes: `docs/handoff/` (newest by filename). Deploy history: `~/.picode/var/deploy-log.jsonl`.

## In flight (unmerged branches on disk)

- `feat/herdr-validation`, `feat/picode-video-pilot` — carry `CHANGELOG.md` edits made before ADR-0105; they fast-forward as they are, but any further changelog line goes to `docs/changelog.d/`.
- `feat/card-selection-chevron`, `feat/usermenu-llama` — no living-doc changes yet.

## Next up

1. First release since 0.1.0: `make changelog` on `main`, then `docs/release-process.md` (`[Unreleased]` is 860 lines).
2. GitHub CI: the next push exercises the ADR-0105 workflow (Ubuntu-only Go matrix, tmux cache); macOS/Windows run on tags or `workflow_dispatch`.
3. llama delivery 3 live validation; owned-service ARM64 acceptance.
4. Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.
5. Sessions phase 2: codex scan cache; Hermes titles only, no `profiles/` scan.
6. CLI prompt door iPhone acceptance; first-class CLI agents refused until protocol convergence (ADR-0091).
7. Compose registration ADR; ADR-0064 cadence; docs-video recapture policy.
8. Compaction re-dogfood; historical Inbox rows; remote-mode and browser-preview (owner infra).
9. Windows clean-machine install (ADR-0098): phase 1 = `install-picode` + `install-runtime` stages in `picode-desktop.exe`; phase 2 = `install.ps1` one-liner + winget experiment, no paid signing. Plan: `docs/plans/windows-clean-install.md`.
10. Inspector debts (ADR-0096): `git ls-files` search, per-anchor watch, `+N −M` footer.

## Known debts / open questions

- Process (ADR-0105): worktrees start with a cold Go test cache (results are keyed by directory); `.pi/compact.json` `atPercent 0.5` never fires for large-window models (peaks 379 K) — owner declined config changes 2026-09-09; capture tolerance is 0.05% of pixels (`scripts/docs-shots.mjs`).
- Communication (ADR-0104/0106): full recorded native Claude/Codex resume roundtrips, OpenCode model turns and physical-device acceptance remain untested; vendor model probes used a Pi fixture capability. Grok/Hermes automatic setup remains unavailable; native session discovery is best effort; never share credentials across conversations. Orphan private setup files after owner deletion need maintenance cleanup; revoked files cannot authenticate. OpenCode inline merging requires JSON objects, not JSONC.
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
