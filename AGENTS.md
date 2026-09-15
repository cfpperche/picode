# AGENTS.md — Operating contract for Pi agents in this repository

> This file is loaded automatically by Pi at session start.
> It is the **source of truth for how agents operate here**. Humans follow
> the same rules via [CONTRIBUTING.md](CONTRIBUTING.md).

## What PiCode is

PiCode is a browser-based Agent Development Environment (ADE) for Pi coding
agents. One Go binary serves a rich web UI that lets users **create, configure
and orchestrate Pi agents** across multiple workspaces — including people who
avoid terminals entirely. The moat: **users control their agents from the
moment of creation**. Read [README.md](README.md) before substantial work;
[docs/architecture.md](docs/architecture.md) is an index; the subsystem
files under `docs/architecture/` are what the table below names.

The direction is a multi-CLI ADE. For the current v1, managed agents remain
Pi; Agent CLIs manages terminal launches for Pi, Claude Code, Codex, Grok,
Hermes Agent and OpenCode, with Muse Code and Antigravity onboarding as
terminal-only entries (ADR-0069). Other CLI protocols, packages and first-class agent support are
future work, not capabilities to infer from a terminal integration.

## What to read for which change (ADR-0086)

Reading is a cost. Start from `docs/handoff.md` (`make handoff` renders it;
in flight, next up, debts) and this file; add only what the change needs:

| Change | Read before the first edit |
|---|---|
| CSS / copy / one component | `.pi/skills/uiux-review/SKILL.md`; the component and its CSS |
| New or redesigned surface | above + `docs/benchmarks.md` (UI/UX section) + the one `docs/benchmarks/` note it adapts |
| Handler / store / RPC | the `docs/architecture/<component>.md` file; the ADRs its comments cite |
| Protocol, persistence, security model, process | the ADRs it touches (index in `docs/decisions/README.md`); `docs/decisions/template.md` |
| Docs site (`docs-site/`) | `docs/guidelines.md` |

Do not read `docs/handoff-archive.md`, `docs/handoff/`, or the ADR corpus
to "get context"; `git log` and `make close-summary` are cheaper and current.

## The non-negotiables

1. **Documentation is a living system.** Code and docs travel in the same
   commit. User-facing command help lives in `docs-site/` (VitePress → GitHub
   Pages), not in the app — see [docs/guidelines.md](docs/guidelines.md).
   - Behavior/architecture changed → update the `docs/architecture/` file
     for that subsystem (the index `docs/architecture.md` only links); add
     an ADR only for a **boundary**: protocol, persistence, security model,
     process (`make adr NAME=x` seeds it; the Boundary line is mandatory).
     A UI refinement or a route move never needs one.
   - Anything user-visible changed → one fragment
     `docs/changelog.d/<branch-slug>.md` (Keep a Changelog sections inside).
     `CHANGELOG.md` is assembled by `make changelog` on `main`; the hook
     refuses direct edits (ADR-0105).
   - **Every session that changes state leaves one note in
     `docs/handoff/<date>-<branch>.md`** (≤ 25 lines, written from
     `make close-summary`). Its optional `## Next up` / `## Debts` sections
     are what the board renders; a durable item goes to
     `docs/handoff/open/<topic>.md` instead. **`docs/handoff.md` is generated
     by `make handoff` and is not in git** (ADR-0123) — never edit it, and the
     pre-commit hook refuses a committed copy, on `main` and in a worktree.
     Shipped work is `git log`, the ADR index and the changelog — never
     handoff prose. Deployment history is `var/deploy-log.jsonl` in the
     data dir (`~/.picode` by default).
2. **Never break the build.** A worktree iterates with `make ci-scoped` and
   ends with `make close` (below); the merge on `main` runs `make ci` once.
   If you can't finish, leave the tree compiling and green and record the
   gap in `docs/handoff/open/<topic>.md`.
3. **Simplicity and modularity are product features.** Prefer the Go standard
   library. Every new dependency is a decision that deserves a line of
   justification in the PR description. UI follows the bars in
   `docs/benchmarks.md`. Substantial features first check
   `docs/benchmarks/` (Cursor, t3code, paseo) and cite an adaptation.
4. **Honesty over polish.** Report what is actually done vs. described. A
   smaller true changelog beats an impressive false one. Unknowns go into
   `docs/handoff/open/<topic>.md` as open questions, not into prose as facts.
   **Seeing a visual defect and shipping it as done is a violation.**
   Fix it or say FAIL. `eval` / DOM JSON is not a visual verdict.
5. **Isolated git worktree.** Two agents must not share a working tree.
   Never commit feature work on `main` in the primary checkout. Start with
   `make worktree NAME=<name>` (a tree with hardlinked `node_modules`,
   ready to build in a second; never symlink them). After the branch
   merges: `make worktree-gc` (or `git worktree remove` + `git branch -d`).
   Leave `main` clean for the next session. A dirty shared tree that blocks
   another agent is FAIL.
   This is **enforced by git, not by trust**: `.githooks/reference-transaction`
   aborts any `git switch`/`git checkout` that would move the root checkout
   off `main` (switching back to `main` is always allowed), refuses any
   rewind of `main` behind its current tip — including a fast-forward from
   a stale ref onto a divergent tip, which silently erased merged work once
   (rolling back is a deliberate `PICODE_ALLOW_MAIN_REWIND=1` one-off) — and
   `.githooks/pre-commit` refuses feature commits made there, clobbered
   living docs, a committed handoff board, whitespace errors, direct
   `CHANGELOG.md` edits, and handoff or fragment commits on `main`
   (ADR-0105). `make hooks` (implied by
   `make dev` and `make ci`) points git at them; `make hooks-check` proves
   the whole policy on a throwaway repo. A clone that never ran make has no
   guard. Deliberate one-off: `PICODE_ALLOW_SWITCH=1 git switch <branch>`.
   **Never run `git clean -fdx` (or `-fdX`) in the primary checkout:**
   `.worktrees/` is git-ignored, so clean deletes every agent's working
   tree in one stroke.
6. **Decisions are provisional.** Every ADR, "Refuse" table and architectural
   constraint here records a choice that was right when it was made — not a
   law. Never answer a request with "that is impossible" or "that is
   structural" because a document says so. Name the decision, explain the trade
   it made, say what changing it would cost, and **wait for the owner's
   approval**. Routine implementation choices stay with the agent; anything
   that alters a documented decision, or declares something permanently
   refused, is the owner's call. A constraint nobody has re-measured is a
   candidate for re-measuring, not a fact.
7. **Deploy is the owner's call (ADR-0105).** A branch is done when `main`
   can fast-forward to it and `make ci` is green there. A branch session
   never deploys on its own; the owner runs `make deploy` from the root
   whenever they want `main` live (it refreshes stale public captures
   first). `picode deploy` refuses while any agent or terminal is
   mid-turn; `--force` is the owner's deliberate one-off, never an agent's
   shortcut. Verify UI work on a scratch instance (`scripts/qa-scratch.sh`),
   not on production.
8. **One branch, one session (ADR-0105).** The session that did the work
   ends at the fast-forward; the next task starts in a new terminal. The
   closing docs are written from `make close-summary` in a subagent or a
   fresh session, never at the peak context of the working session (the
   rite cost a median 2.7 M tokens per branch when run in place). Screenshots
   for visual-review are read in a subagent, so images never sit in the main
   context.

## Closing a session (the rite, in one command)

```bash
make ci-scoped     # while iterating: the gates this diff can break
make close         # at the end: scoped gates (or the green run they can
                   # reuse), regenerated OpenAPI, the rendered board,
                   # fast-forward check, and the closing summary
```

Write `docs/changelog.d/<branch-slug>.md` and
`docs/handoff/<date>-<branch>.md` from the summary — in a subagent or a
fresh session (non-negotiable 8). Then, from the root:
`git merge --ff-only <branch> && make ci`. If `main` moved, merge `main`
into the branch and run `make close` again (it reuses a green `ci-scoped`
when the merge left the content those gates read untouched, ADR-0124). Use the skills:
`/skill:quality-gate` (review checklist), `/skill:handoff-update` (the
note). When a change has **interacting conditions that change the outcome**
(delete, restore, auth, cascade, run mode, permissions), write a
**decision table** before claiming done: each row is conditions → action.
Tests must cover every row, or the untested row is named as debt in
`docs/handoff/open/<topic>.md`. Skip the table for polish, copy, and single-path fixes.
**Motion and optimistic UI** are the default for state that takes time
(jobs, overlays, lists). A static flash then "all done" is FAIL.

For any UI work:
1. `read` `.pi/skills/uiux-review/SKILL.md` **before** the first JSX/CSS edit.
2. Empty / blocked / error states are first-class: one line + one action.
3. Before done, `read` `.pi/skills/visual-review/SKILL.md`. Screenshots of
   those states must be `read` (they stay in `var/screenshots/`, never
   committed). After overlays, `window.__picodeOverlayAudit()` must be `ok`.
   Answer the 5-question visual-card in the reply.
4. Skip or FAIL on visual-review → do not commit, do not say shipped.

## Commands

| Command | What it does |
|---|---|
| `make worktree NAME=x` | Isolated tree on `feat/x` with hardlinked node_modules |
| `make dev` | Run the Go server — reads the UI from disk; run `make web` once first (ADR-0023) |
| `make ui` | Vite HMR on :5173 (proxies API to the Go server) |
| `make web` / `make build` | Build React UI → `internal/web/public` / UI + `bin/picode` |
| `make ci-scoped` / `make close` | Gates for this diff / end-of-session rite |
| `make ci` | Everything CI runs — the gate for the merge on `main` |
| `make deploy` | Owner only: rebuild, refresh stale captures, restart the service; refuses while agents work |
| `make changelog` | Fold `docs/changelog.d/` fragments into `CHANGELOG.md` (on `main`, before a release) |
| `make adr NAME=x` | Seed the next ADR with its number and index row |
| `make handoff` | Render `docs/handoff.md` (generated view; `make close` refreshes it before the closing summary) |
| `make worktree-status` | What is actually in flight: branch, ahead/behind, dirty files, last commit, last green run |
| `make worktree-gc` | Remove merged, clean, idle worktrees |
| `make cert-timer` | Install the weekly certificate check (systemd --user) |
| `make desktop-restart` | Swap the Windows tray + native-host exes and relaunch via the logon task — the only supported restart; never background a Windows exe from WSL |

## Rules for the agent itself

- **Never `pkill -f <pattern>`** (or `killall`) from a session: the pattern
  matches the command line running it, so the shell that issued it dies first
  (a session killed its own `make` chain this way, and an earlier
  `grep '^picode-'` sweep killed 29 live terminals). Use the script's own verb
  (`./scripts/qa-scratch.sh stop <name>`, `fuser -k <port>/tcp`), or an exact
  PID you started and can still see.
- **Never `tmux kill-server`** — not even with `TMUX_TMPDIR` set, which is how
  a QA session took the production tmux down and cost the owner every running
  terminal (2026-09-15). Kill the exact session you created
  (`tmux kill-session -t picode-sh-<id>`), and let
  `./scripts/qa-scratch.sh stop <name>` do it for a scratch instance.
  **`$TMUX` outranks `TMUX_TMPDIR`** — measured 2026-09-15: a client started
  inside a session talks to the server named in `$TMUX` no matter what
  `TMUX_TMPDIR` says (a probe with `TMUX_TMPDIR=/tmp/x` printed production's
  `/tmp/tmux-1000/default`). `-L` does override `$TMUX`, but then
  `TMUX_TMPDIR` counts only if that directory exists — missing, tmux falls
  back to the shared `tmux-<uid>` dir silently. The safe scratch recipe is
  `mkdir -p $dir && TMUX_TMPDIR=$dir tmux -L <unique-name> …`; a session
  launched with neither lands in the owner's tmux (and in every other
  agent's `pkill` radius). Clean up by exact session name, or check the
  scratch's own Terminals list before blaming the server. PiCode terminals
  refuse `kill-server` through the tmux guard — but it is a guardrail, not a
  boundary: an absolute `/usr/bin/tmux kill-server` still bypasses it.
  A Go test that launches a real terminal cleans it up with a context of its
  own: `t.Context()` is canceled *before* cleanup functions run, so every
  tmux call made with it fails silently and the fixture leaks one session
  per run into the shared server (four `picode-sh-feed-fixture-*` sessions
  were found in the owner's `tmux ls` this way).
- **Know which tree you are in.** `make dev`, `make ci-scoped` and `make close`
  print `<worktree> on <branch>` before doing anything; the same line answers
  "did I edit the root checkout by mistake?" (`make worktree-status` lists every
  tree with its dirty count). The root checkout is shared — a stray edit there
  is the failure mode that blocks every other session (AGENTS.md §5).

## Repo map

```
AGENTS.md          this contract
docs/              living documentation (handoff.md = generated view, `make handoff`;
                   handoff/ = session notes; handoff/open/ = durable next/debts per topic;
                   architecture/ = one file per subsystem; changelog.d/ = changelog fragments)
docs-site/         public docs (VitePress Markdown → GitHub Pages)
docs/decisions/    ADRs — one decision per file, immutable once accepted
docs/screenshots/  frozen visual history (ADR-0086); new evidence stays in var/screenshots/
.pi/               Pi harness: skills, project settings, roles
cmd/picode/        entrypoint
ext/               Chrome MV3 extension, sideload (ADR-0043)
internal/browserhost/  native-messaging host + Chrome install
internal/server/   HTTP server + API
internal/web/      UI loader: from disk by default, embedded with `-tags embedui` (ADR-0023).
                   public/ is Vite output and is NOT committed
web/               Independent desktop/mobile apps + shared contracts/tokens (ADR-0072)
scripts/           gates, close, worktree, changelog assembly, ADR seed, QA scratch instance
.github/           CI
```

## Architectural decisions

Significant choices (frameworks, protocols, persistence, security model,
process) go through an **ADR**: `make adr NAME=<short-title>` allocates the
number across every worktree, seeds the file from `docs/decisions/template.md`
and appends the index row. Fill the **Boundary** line first: a change that
crosses no protocol, persistence, security-model or process boundary is not
an ADR — it is a paragraph in `docs/architecture/` or a note in `docs/plans/`
(seventeen ADRs landed in the three days after ADR-0086; four were UI or
route refinements). Argue context → decision → consequences. Never silently
contradict an ADR — supersede it with a new one.

## Style

- **Language policy: English.** The repository's official language is
  English — code, comments, docs, commits, changelog entries, issues and
  PR descriptions. No exceptions for canonical content.
- Go: idiomatic, stdlib-first, table-driven tests, no `init()` magic.
- UI: React in `web/`; design tokens live in `web/shared/tokens/theme.css`
  (do not invent a second palette). After any UI change run `make web`
  and a JS/JSX syntax check (`npm run build` must succeed).
- **Apps must not leak into PiCode's own interface**: an app reaches the
  host only through the closed list of doors ADR-0109 declares (tile and
  badge, its tab, its own body, the manifest icon key, the `host` object,
  the `#/app/<id>` route), so anything else — a Preferences group, a
  section inside another surface, a row in a host list — needs a new door
  in that ADR, never a commit.
- **Forms: Zod, never native browser validation.** Schemas live in
  `web/shared/contracts/schemas.js`. Forms set `noValidate`. Same messages in every
  browser.
- **Prefer popular primitives over homemade widgets.** Use Radix (already
  in the app), cmdk, **shadcn/ui patterns**, or native controls. Tailwind
  (ADR-0008) is the utility layer; tokens stay CSS variables. Roll a custom
  control only when no library/pattern covers the case — and say why.
  Native `<select>` / `<input>` still beat a one-off styled fake.
- **One control height.** Adjacent input/select/button use `--ctl-h` (36px).
  A row of mixed heights is FAIL (shadcn `h-9` / HIG).
- **Empty states are required.** A list, gallery, or collapsible section
  that can have zero items must show a one-line placeholder (and the add
  action if one exists). Never a blank well. Never a "0" count badge.
- **State changes go through the store and announce themselves (ADR-0048).**
  A mutation is a store method that appends its event in the same
  transaction; writing to SQLite around the store, or adding a mutator
  without a row in `TestEveryMutationAppendsAnEvent`, is a bug. UI lists
  subscribe to the feed (`web/shared/client/feed.js`) and patch or refetch; a
  new `setInterval` against `/api/*` needs a reason the feed cannot cover
  (metrics, presence, tmux).
- Commits: imperative, scoped (`server: add /api/version endpoint`).
- Docs: short paragraphs, tables for comparisons, diagrams over prose.
- The audience includes terminal-averse users: UI copy avoids jargon;
  when a technical term is unavoidable (PTY, RPC), a tooltip explains it.
