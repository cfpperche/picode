# Handoff — living project state

> Heartbeat of PiCode. A session that changes state leaves this file matching
> HEAD. Historical detail lives in `docs/handoff-archive.md`.

## Current state (read this first)

**Repository:** the desktop **Inspector rail** (ADR-0078, proposed until the
owner accepts the shipped result) is complete on `feat/inspector`: Changes and
Files beside the center, per-file counts on `gitstatus`, the file tab's Diff
view, the `Ctrl+.` toggle, the seeded docs fixture and `app-inspector` public
capture. Merge and deployment are recorded under Recent activity. Bounded
captures (ADR-0076), history reconciliation and mobile composer wrapping were
merged and deployed as `974780ba` before it.
The capture ADR was renumbered because Integrations took 0075. The opt-in native
emitter and real end-to-end acceptance remain pending: deployment does not
enable browser capture emission. Integrations and the current provider favicons
and tab/sidebar size fixes remain included.
HEAD also includes File Tree v2 (0074), worktree-aware Git Graph (0073), independent
web apps (0072), Windows task reliability (0071), Agent CLIs v2 and Docker v3.
Managed agents remain Pi-only; coding CLIs are terminals. No push was made.
Preserve the unrelated root `.pi/compact.json`.

**Last application deployment:** `0.1.0+974780b`, health `ok`, boot
`1429ba9cb49904d5`. `make deploy` restarted systemd; all six terminal records
and 28 baseline tmux pane identities survived. Installed/build binary hashes
and served desktop/mobile assets match (`index-BRF94S2P.js` / `index-Beff3F6q.js`).
Live screenshots were read, including dashboard scrolling; audits passed and
no browser errors were reported. Receipts: `var/capture-merge/`; private SQLite
and previous-binary backups remain in its `recovery/` folder.

**Quality:** combined main `make ci` passed (668 frontend tests plus Go,
package/build/docs/Vale gates). Ten combined-host screenshots were read;
viewer audits passed. Preexisting mobile Send clipping was fixed with wrapping
and checked at 320/390px, including the running state. This is host QA, not a
real emitter verdict. Fixture and QA browser are stopped; feature worktree and
branch are removed. Original receipts remain in `var/tool-preview/`.
Previous Integrations/File Tree/mobile gate details are archived.

### Product and platform

- The desktop Inspector rail (ADR-0078) follows the selected tab's owner
  (agent, terminal, workspace; apps keep the last anchor) and shows **Changes**
  — a folder-grouped working tree with `+N −M` per file and folder, an
  `Uncommitted` total, branch and worktree, and an `All | This agent` scope
  beside an agent — and **Files**, the lazy project tree with a filter over
  loaded rows. It opens files and diffs as center tabs (the file tab gained a
  Diff view), pins the owner's root and turns a background 409 into "This
  terminal moved to … Follow". Width/open/tab are per-viewer localStorage;
  open by default at ≥1440px; shrinks before it hides and never leaves the
  conversation under 640px. `gitstatus` now carries `add`/`del`/`binary`,
  `totals`, `branch` and `worktree`.
- One Go binary serves independent `/desktop/` and `/mobile/` apps. Mobile
  owns copied UI and lazy screens; shared contracts/tokens have explicit
  exports. HTTPS defaults to `:8445`.
- Integrations owns Connectors and Webhooks on both apps. Generic signed
  delivery lives in core; service-specific tools stay in external MCP packages
  or services. Native configuration import is reviewed, not implicit execution.
  Installed connector packages are distinct from configured/live services.
- Webhooks persist cursors, retry deadlines and revision-guarded acknowledgements.
  Only durable events are eligible; retention gaps are recorded. Secrets appear
  only on creation/rotation. No redirects/environment proxy; metadata addresses
  are refused. HTTP/LAN destinations require user trust, not a sandbox.
- Git Graph has one dirty row per worktree, read-only branch/HEAD-addressed
  sibling reads and manual refresh. Unused `git/head` token endpoints remain.
- File Tree keeps files and working diffs inline, protects drafts on replacement,
  close and refresh, and distinguishes Git-ignored entries without hiding them.
- Workspaces support managed Pi agents, free agents and tmux terminals. Private
  sessions follow ADRs 0039/0040/0053; Inbox uses the receiver extension with a
  paste fallback (0060). Store mutations append events transactionally (0048).
- Terminal presence/activity use wrapper leases and exact process fallback,
  never scraped pixels (0062). Agent CLIs v2 owns launch defaults, profiles,
  previews and setup checks; saving configuration never restarts a process.
- Docker v3 and `pi-sysadmin` share inventory, operations, health and reviewed
  maintenance. Compose registration remains proposed.
- Public docs have generated API/llms maps, screenshots and video integrity
  checks. Releases remain tag-driven; ADR-0064's cadence pilot is proposed.
- Windows task policy, diagnosis, scoped repair and normal Quit are locally
  installed. Native lifecycle/race and UAC repair passed; repair is repeatable
  without altering actions/principals/triggers. Device transitions are deferred.

### Compaction policy (`pi-compact`, ADR-0061)

Deployed and dormant without configuration. The root `.pi/compact.json` belongs
to other work and was not evaluated. Real-compaction acceptance must still prove
`fromHook: true`, gemini-3.6-flash pricing and no aborted turns.

## In flight

- OAuth-provider acceptance, model-driven connector usage and first-class
  non-Pi agents are not certified by the public/no-auth DeepWiki protocol check.
  Integrations itself is merged and deployed.
- Physical iOS/Android PWA upgrade and push need device acceptance. The browser
  migration, shared worker matrix and mobile review fixes passed locally.
- Windows next-logon, battery and sleep/resume acceptance is owner-deferred;
  do not interrupt the current Windows session for these tests.
- Compose registration/deployment remains a separate ADR-0065 extension;
  existing-project operations and Docker v3 are merged and deployed.
- Autonomous model-driven Sysadmin dogfood and Docker Desktop/rootless acceptance
  remain open. Real Linux Engine QA passed without a model turn.
- Configured real-compaction and model-driven CLI/Inbox dogfood remain pending.
  Reconcile historical Inbox `[Teste 3]` and `mobile-6bf740` before a live reply.
- ADR-0054 `picode-act` needs real model-emitted dogfood before merge; Browser
  preview needs the native opt-in emitter and real partial/final RPC acceptance
  before panel work; the bounded host increment is implemented.
- Second-account, container, public-OIDC and other remote-mode acceptance require
  owner-controlled infrastructure. Official release dates remain unset.

## Next up

1. Scope provider OAuth/marketplace acceptance only if the owner requests it;
   generic Integrations and external connector setup are already deployed.
2. Validate deployed PWA upgrades and push on iOS/Android. Mobile UI increments
   belong only in `web/mobile`.
3. Run the version-specific CLI working/approval/settled acceptance matrix.
   First-class CLI agents need a protocol/session/package parity ADR.
4. Design Compose registration from `docs/plans/docker-v2.md`: file ownership,
   dependency order, deployment preview and recovery need an ADR.
5. Review ADR-0064 and choose the cadence/pilot window.
6. Re-dogfood configured compaction; inspect historical Inbox rows before replies.
7. Run owner-controlled remote-mode acceptance. Continue Browser preview with
   opt-in native emission and real RPC/cancellation/slow-consumer acceptance,
   not the panel yet; ADR-0054 dogfood remains separate.
8. Decide whether selective docs-video recapture/render should be scheduled.
9. After a few days of Inspector dogfood, decide its later phases (ADR-0078
   designs both, neither approved): a read-only **PR** tab through the host's
   `gh`, and **Commit / Commit & Push** — pre-typed into the owner's terminal,
   or server-side behind an interlock. Accept or amend ADR-0078 then.

## Known debts / open questions

- **Capture integration: FAIL/deferred.** No real emitter-to-RPC run or measured
  slow-consumer/cancellation matrix. The hub drops on overflow and caches no
  missed partials. Desktop reconnect and same-agent session replacement during
  a pending session-changing API need dedicated acceptance; mobile reconnect
  and selection switching passed with fixtures. See `docs/plans/browser-preview.md`.

- Webhook delivery is at-least-once within event retention, not an unlimited
  archive. Receivers must handle duplicate IDs; arbitrary receiver response
  bodies are not retained. Public connector QA proves protocol, not agent use.
- Task Scheduler retries are not general crash recovery: an exit-one probe
  stayed stopped for 90 seconds. Runtime supervision needs a new approved design.
  Battery/sleep/sign-in remain owner-controlled; see the desktop reliability plan.
- Existing `TestTerminalBrowse` cleanup can leave tmux shells in deleted temporary
  folders. Review cleanup lifetime; preserve pre-existing unrelated shells.
- CLI lifecycle coverage is version-specific; executable/setup checks alone do
  not prove working/approval/settled behavior for a vendor.
- Docker leaves volume deletion, backup/restore, remote engines, historical
  charts and automatic repair for future decisions. Secret masking is best effort.
- Legacy terminal sessions get exact PID/command fallback; instrumented sessions
  have stronger presence. `/proc` start tokens give Linux stronger identity checks.
- Ephemeral feed events can be missed across reconnects (0048). Cross-platform
  paste-fallback acceptance remains open; receiver ack precedes Pi processing.
- Pi has one active credential slot; per-agent OAuth and proactive quota switching
  need an owner decision. The Shift+Enter shim and some legacy role-state/slash
  assumptions remain until stable replacements are measured.
- Tutorial integrity passes, but all three strict freshness audits remain stale
  after source relocation. Recapture/render is explicit; hashes were not relabeled.
- Branch protection and CODEOWNERS require owner action on GitHub.
- Inspector debts (ADR-0078): no complete filename search yet — the Files
  filter covers loaded rows only (`/files?q=` is agent-only and stops at 200
  files; `git ls-files` for all three owner kinds is the planned fix); free
  terminals outside a watched folder refresh on focus/visibility and their row's
  live facts, not on a watcher (a per-anchor watch lease is the fallback);
  the This-agent scope chips show only while the agent's own tab is selected;
  the sizer idiom is still copied in Sidebar and FileTreeSurface; the per-turn
  `+N −M` footer beside the conversation is not drawn. ADR-0078 may be
  renumbered at merge — `feat/pi-diff` also holds an unmerged 0076.

## Recent activity

- **2026-09-05 — Inspector rail (ADR-0078).** Study
  `docs/benchmarks/2026-09-05-inspector-rail.md` (Paseo, Orca, t3code), ADR
  and `docs/plans/inspector.md`; `gitgraph.StatusWithStats` with Go tests;
  `lib/inspector.js` + `lib/resizeEdge.js` (17 node tests); `Inspector.jsx`
  mounted after `<main>`; file-tab Diff view; fixture seeds a dirty repository;
  `app-inspector` docs capture. Browser QA on an isolated fixture:
  `scripts/qa-inspector.mjs` passed 11/11 groups (`docs/screenshots/inspector-qa.json`);
  11 screenshots read, overlay/row audits ok. visual-review: PASS. Two defects
  found and fixed in review: the blocked line named the stale cwd (now read live
  from `…/cwd`), and the Diff view's header row was misaligned outside the
  folder tab.
- **2026-09-05 — Capture host merged and deployed.** Reconciled main and
  renumbered the capture ADR to 0076; corrected mobile toolbar clipping found
  in QA. Combined `make ci`, refreshed host states, 320/390px geometry and live
  deployment checks passed. visual-review: PASS. Deployed `974780ba`; emitter
  remains pending. No push or package-setting changes.

Older activity lives in `docs/handoff-archive.md`.
