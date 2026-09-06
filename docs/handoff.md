# Handoff — living project state

> Read this first. At most 100 lines (the pre-commit hook refuses more).
> Session notes: `docs/handoff/` (newest by filename). Deploy history:
> `~/.picode/var/deploy-log.jsonl` and `git log`. Older prose: `docs/handoff-archive.md`.

## Current state

- **Process (ADR-0086, 2026-09-06):** `picode deploy` refuses while any agent
  or terminal is mid-turn (`GET /api/deploy/readiness`, loopback); `main`
  ships in batches (`make deploy-batch`, `picode-deploy.timer` at
  12:00/18:00/23:00 — `make timers` installs it). A branch closes with
  `make close`; iterate with `make ci-scoped`; `make ci` runs once on `main`
  at the merge. `make worktree NAME=x` / `make worktree-gc`. Capture parity
  is advisory in `make ci`, strict in `close` and the batch.
- **Terminals:** CLI session pin + one-click resume (ADR-0084); flight
  recorder, SIGHUP-immune pane roots and the deploy log (ADR-0085) — the
  18:50 restart left all nine sessions alive. Terminal checklists (ADR-0081)
  with the compact line aligned to the card's text column; absent checklist
  renders silence (ADR-0082). Sessions live under Agent CLIs (ADR-0079).
- **Inspector rail (ADR-0078):** Changes, Files, PR tab, Git actions
  (prepare, run-when-idle, "Ask <agent>" through the agent's own channel).
- **llama.cpp manager:** deliveries 1–2 deployed (ADR-0080/0083: durable
  jobs, progress, cancellation on verified b10809, reconnect); 3–4 planned
  in `docs/plans/llama-manager.md`.
- Also on `main`: File Tree v2 (0074), Git Graph per worktree (0073),
  independent desktop/mobile apps (0072), Windows task reliability (0071),
  Agent CLIs v2 (0069), Integrations (0075), Docker v3, identity favicons
  at 16 px, tab strip overflow phases 2–4. Managed agents remain Pi-only;
  coding CLIs are terminals. `origin/main` is pushed regularly.

## In flight (unmerged branches on disk)

- `feat/llama-service` — explicit execution settings and binary pins.
- `feat/picode-feature-video` — skills record clicks and typing, not slideshows.
- `feat/llama-guidance` — delivery 3 guidance dialog; validate on a scratch instance.

## Next up

1. Watch the first batch deploy (timers installed 2026-09-06 19:20; next
   23:00; `journalctl --user -u picode-deploy`): it talks to a daemon
   without the readiness route and deploys unguarded; later ones refuse.
2. Renumber the duplicate ADR-0082 (browser-capture-sidecar vs
   absent-checklist-renders-silence) and fix the index.
3. llama delivery 3 live validation; delivery 4 needs a service-ownership /
   cache-deletion ADR.
4. Tab strip debts: keyboard close of `.mtab-close`, needs-you dot on an
   arrow verified on a live tab, `scrollbar-width: thin` vs `::-webkit-scrollbar`.
5. Sessions phase 2: codex machine-wide scan cache (~6 s on 907 files);
   narrow injected-block heuristic; Grok has no transcripts.
6. CLI working/approval/settled acceptance matrix per vendor version;
   first-class CLI agents need a protocol/session/package parity ADR.
7. Compose registration ADR (`docs/plans/docker-v2.md`); ADR-0064 cadence
   decision; docs-video recapture policy.
8. Configured compaction re-dogfood; reconcile historical Inbox rows before
   live replies; remote-mode and browser-preview acceptance (owner infra).
9. Inspector follow-ups: merge/rebase/branch switch wait for a picker
   (reset, force push, stash drop stay refused); full filename search
   (`git ls-files`); per-anchor watch lease; per-turn `+N −M` footer.

## Known debts / open questions

- CLI pane-death signal chain unproven; ADR-0085 instruments it — the next
  deploy that loses sessions is the experiment. ADR-0084 pins nothing for
  terminals stopped before it shipped (Sessions → "Open in terminal").
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
- 2026-09-06 incident: a `tmux ls | grep '^picode-' | xargs kill-session`
  sweep killed 29 sessions, six production ones; `orca-tasks` did not come
  back. Never kill by shared prefix — only exact names from an isolated
  fixture's own API.
- Feed: ephemeral events can be missed across reconnects (ADR-0048); paste
  fallback acceptance across platforms open.
- Pi has one active credential slot; per-agent OAuth is an owner decision.
- Tutorial video freshness audits are stale after source relocation;
  recapture/render is explicit. Branch protection and CODEOWNERS need the
  owner on GitHub. Desktop requests `/desktop/favicon.svg` and gets 404.
- Inspector: Files filter covers loaded rows only; This-agent chips show only
  with the agent's tab selected; `gh pr view` answers cached a minute.
- llama: GPU / non-b10809 cancellation unverified; an unknown download with
  an absent model keeps its reservation; history pruning deferred.
