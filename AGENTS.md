# AGENTS.md — Operating contract for coding agents in this repository

> Every agent CLI that reads `AGENTS.md` loads this file at session start.
> It is the **source of truth for how agents operate here**. Humans follow
> the same rules via [CONTRIBUTING.md](CONTRIBUTING.md). The rules live here;
> their detail and history live in `docs/agents/` — open those files when a
> rule applies to what you are doing.

## What PiCode is

PiCode is a browser-based Agent Development Environment (ADE) for coding-agent
CLIs. One Go binary serves a rich web UI that lets users **create, configure
and orchestrate agents** across multiple workspaces — including people who
avoid terminals entirely. The moat: **users control their agents from the
moment of creation**. Read [README.md](README.md) before substantial work;
[docs/architecture.md](docs/architecture.md) is an index; the subsystem
files under `docs/architecture/` are what the table below names.

An **agent** is a workspace or free instance of any launchable CLI — Pi,
Claude Code, Codex, Grok, Hermes Agent, OpenCode, Muse Code, Antigravity or
Omp (ADR-0160, ADR-0179). Only Pi has a **managed** mode (`pi --mode rpc`,
ADR-0091); every other CLI runs its own TUI in a PiCode terminal. **Pi is one
CLI among nine, never a dependency**: PiCode runs with none of them present,
and tmux is its only runtime requirement. Copy or code that presents Pi as
required, or PiCode as "for Pi", is a defect. Managed mode for the other CLIs
is future work, not a capability to infer from a terminal integration.

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
Where things live: [docs/agents/repo-map.md](docs/agents/repo-map.md).

## The non-negotiables

1. **Documentation is a living system.** Code and docs travel in the same
   commit. User-facing command help lives in `docs-site/`, not in the app
   ([docs/guidelines.md](docs/guidelines.md)).
   - Behavior/architecture changed → update the `docs/architecture/` file
     for that subsystem; add an ADR only for a **boundary**: protocol,
     persistence, security model, process (`make adr NAME=x`; the Boundary
     line is mandatory). A UI refinement or a route move never needs one.
   - Anything user-visible changed → one fragment
     `docs/changelog.d/<branch-slug>.md`. Never edit `CHANGELOG.md`
     directly (ADR-0105).
   - **Every session that changes state leaves one note in
     `docs/handoff/<date>-<branch>.md`** (≤ 25 lines, from `make
     close-summary`); durable items go to `docs/handoff/open/<topic>.md`,
     where a debt is also paid (`- [ ]` → `- [x]`, never delete another
     session's line). **`docs/handoff.md` is generated and not in git** —
     never edit it. A fact that exists only after the merge is one commit on
     `main` amending the note or topic it corrects (ADR-0149). Board rules
     and the rest: [docs/agents/handoff.md](docs/agents/handoff.md).
2. **Never break the build.** A worktree iterates with `make ci-scoped` and
   ends with `make close`; the merge on `main` runs `make ci` once. If you
   can't finish, leave the tree compiling and green and record the gap in
   `docs/handoff/open/<topic>.md`.
3. **Simplicity and modularity are product features.** Prefer the Go standard
   library. Every new dependency deserves a line of justification in the PR
   description. UI follows the bars in `docs/benchmarks.md`. Substantial
   features first check `docs/benchmarks/` and cite an adaptation.
4. **Honesty over polish.** Report what is actually done vs. described.
   Unknowns go into `docs/handoff/open/<topic>.md` as open questions, not into
   prose as facts. **Seeing a visual defect and shipping it as done is a
   violation.** Fix it or say FAIL. `eval` / DOM JSON is not a visual verdict.
5. **Isolated git worktree.** Two agents must not share a working tree.
   Never commit feature work on `main` in the primary checkout. Start with
   `make worktree NAME=<name>` (hardlinked `node_modules`; never symlink
   them); after the merge, `make worktree-gc`. Leave `main` clean — a dirty
   shared tree that blocks another agent is FAIL. Git hooks enforce this
   (`make hooks`; proof: `make hooks-check`); a deliberate one-off switch is
   `PICODE_ALLOW_SWITCH=1 git switch <branch>`.
   **Never `git add -A` or `git add .`** — stage explicit paths.
   **Never run `git clean -fdx` (or `-fdX`) in the primary checkout** — it
   deletes every agent's worktree under `.worktrees/`.
   What the hooks refuse, and why: [docs/agents/git-guards.md](docs/agents/git-guards.md).
6. **Decisions are provisional.** Every ADR, "Refuse" table and architectural
   constraint records a choice that was right when it was made — not a law.
   Never answer a request with "that is impossible" or "that is structural"
   because a document says so. Name the decision, explain the trade it made,
   say what changing it would cost, and **wait for the owner's approval**.
   Routine implementation choices stay with the agent; anything that alters a
   documented decision, or declares something permanently refused, is the
   owner's call. A constraint nobody has re-measured is a candidate for
   re-measuring, not a fact.
7. **Deploy is the owner's call (ADR-0105).** A branch is done when `main`
   can fast-forward to it and `make ci` is green there. A branch session
   never deploys on its own; the owner runs `make deploy` from the root.
   `--force` is the owner's deliberate one-off, never an agent's shortcut.
   Verify UI work on a scratch instance (`scripts/qa-scratch.sh`), not on
   production.
8. **One branch at a time; the owner ends the session (ADR-0105).** A
   session works one branch through its fast-forward before the next; it may
   go on to another task when the owner asks, and the owner says when to
   switch terminals. The closing docs are written from `make close-summary`
   in a subagent, never at the peak context of the working session.
   Screenshots for visual-review are read in a subagent, so images never sit
   in the main context.

## Closing a session (the rite, in one command)

```bash
make ci-scoped     # while iterating: the gates this diff can break
make close         # at the end: scoped gates (or the green run they can
                   # reuse), the living-docs pass, regenerated OpenAPI, the
                   # rendered board, fast-forward check, closing summary
```

Write `docs/changelog.d/<branch-slug>.md` and
`docs/handoff/<date>-<branch>.md` from the summary (non-negotiable 8). Then,
from the root: `make land BRANCH=<branch>`. If `main` moved, merge `main`
into the branch and run `make close` again (ADR-0124). Skills:
`/skill:quality-gate`, `/skill:handoff-update`.

When a change has **interacting conditions that change the outcome**
(delete, restore, auth, cascade, run mode, permissions), write a **decision
table** before claiming done: each row is conditions → action, and tests
cover every row or the untested row is a debt in
`docs/handoff/open/<topic>.md`; skip it for polish, copy and single-path
fixes. **Motion and optimistic UI** are the default
for state that takes time; a static flash then "all done" is FAIL.

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
| `make dev` / `make ui` | Go server reading the UI from disk (`make web` first) / Vite HMR on :5173 |
| `make web` / `make build` | Build React UI → `internal/web/public` / UI + `bin/picode` |
| `make ci-scoped` / `make close` | Gates for this diff / end-of-session rite |
| `make land BRANCH=x` | From the root: fast-forward `main` to the branch, then `make ci` |
| `make ci` | Everything CI runs — the gate for the merge on `main` |
| `make deploy` | Owner only: rebuild, refresh stale captures, restart; refuses while agents work |
| `make adr NAME=x` / `make changelog` / `make handoff` | Seed an ADR / fold changelog fragments (on `main`) / render the board |
| `make worktree-status` / `make worktree-gc` | What is in flight / remove merged, clean, idle worktrees |
| `make desktop-test` / `make desktop-restart` | Host-test the shell's Rust half / the only supported restart of the Windows shell |
| `make cert-timer` | Install the weekly certificate check (systemd --user) |

## Rules for the agent itself

Measurements and incidents behind each rule: [docs/agents/processes-and-tmux.md](docs/agents/processes-and-tmux.md).

- **Owner-grade restarts are serialized, one at a time, verified.** `make
  deploy`, `make desktop-restart` and `picode deploy` share
  `/tmp/picode-mutate.lock`: never run two at once, never work around the
  wait, verify each before the next, never end a turn with one still running,
  and never `--force` while a background job of yours is in flight.
- **Never `pkill -f <pattern>`** (or `killall`): the pattern matches your own
  shell. Use the script's own verb (`./scripts/qa-scratch.sh stop <name>`,
  `fuser -k <port>/tcp`) or an exact PID you started.
- **Never `tmux kill-server`**, not even with `TMUX_TMPDIR` set: `$TMUX`
  outranks it. Kill the exact session you created, and let `qa-scratch.sh
  stop` clean a scratch instance. Tools that shell out to tmux pass
  `-S <socket>`. A Go test that launches a real terminal cleans up with a
  context of its own, never `t.Context()`.
- **Know which tree you are in.** `make dev`, `make ci-scoped` and `make close`
  print `<worktree> on <branch>`; `make worktree-status` lists every tree with
  its dirty count. A stray edit in the shared root checkout blocks everyone.

## Architectural decisions

Significant choices go through an **ADR** (`make adr NAME=<short-title>`).
Fill the **Boundary** line first: a change that crosses no protocol,
persistence, security-model or process boundary is not an ADR — it is a
paragraph in `docs/architecture/` or a note in `docs/plans/`. Argue context →
decision → consequences. Never silently contradict an ADR — supersede it.

## Style

- **Language policy: English.** Code, comments, docs, commits, changelog
  entries, issues and PR descriptions. No exceptions for canonical content.
- Go: idiomatic, stdlib-first, table-driven tests, no `init()` magic.
- UI: React in `web/`; design tokens live in `web/shared/tokens/theme.css`
  (no second palette). After any UI change, `make web` must succeed.
- **Apps must not leak into PiCode's own interface**: an app reaches the host
  only through the closed list of doors ADR-0109 declares; anything else needs
  a new door in that ADR, never a commit.
- **Forms: Zod, never native browser validation.** Schemas live in
  `web/shared/contracts/schemas.js`; forms set `noValidate`.
- **Prefer popular primitives over homemade widgets**: Radix, cmdk, shadcn/ui
  patterns, or native controls; Tailwind is the utility layer (ADR-0008). A
  custom control only when nothing covers the case — and say why.
- **One control height.** Adjacent input/select/button use `--ctl-h` (36px);
  a row of mixed heights is FAIL.
- **Empty states are required**: a one-line placeholder (and the add action
  if one exists). Never a blank well. Never a "0" count badge.
- **State changes go through the store and announce themselves (ADR-0048).**
  A mutation is a store method that appends its event in the same
  transaction, with a row in `TestEveryMutationAppendsAnEvent`. UI lists
  subscribe to the feed (`web/shared/client/feed.js`); a new `setInterval`
  against `/api/*` needs a reason the feed cannot cover.
- Commits: imperative, scoped (`server: add /api/version endpoint`).
- Docs: short paragraphs, tables for comparisons, diagrams over prose.
- The audience includes terminal-averse users: UI copy avoids jargon; an
  unavoidable technical term gets a tooltip.
