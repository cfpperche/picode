# AGENTS.md — Operating contract for Pi agents in this repository

> This file is loaded automatically by Pi at session start.
> It is the **source of truth for how agents operate here**. Humans follow
> the same rules via [CONTRIBUTING.md](CONTRIBUTING.md).

## What PiCode is

PiCode is a browser-based Agent Development Environment (ADE) for Pi coding
agents. One Go binary serves a rich web UI that lets users **create, configure
and orchestrate Pi agents** across multiple workspaces — including people who
avoid terminals entirely. The moat: **users control their agents from the
moment of creation**. Read [README.md](README.md) and
[docs/architecture.md](docs/architecture.md) before substantial work.

The direction is a multi-CLI ADE. For the current v1, managed agents remain
Pi; Agent CLIs manages terminal launches for Pi, Claude Code, Codex and Grok
(ADR-0069). Other CLI protocols, packages and first-class agent support are
future work, not capabilities to infer from a terminal integration.

## The non-negotiables

1. **Documentation is a living system.** Code changes and documentation
   changes travel together, in the same commit. User-facing command help
   lives in `www/` (VitePress → GitHub Pages), not in the app — see
   [docs/guidelines.md](docs/guidelines.md). Specifically:
   - Behavior/architecture changed → update `docs/architecture.md` **and**
     add/revise an ADR in `docs/decisions/` if the decision is architectural.
   - Anything user-visible changed → add an entry to `[Unreleased]` in
     `CHANGELOG.md` (Keep a Changelog style).
   - **Every session that changes state MUST end by updating
     `docs/handoff.md`** so it matches HEAD (current state, in flight, next
     steps, debts). Listing shipped work under *Next up* is FAIL. Archive
     old *Recent activity* to `docs/handoff-archive.md` past ~150 lines.
2. **Never break the build.** Before ending work, run the quality gates
   (below). `make ci` must pass. If you can't finish something, leave the
   tree compiling and tests green, and record the gap in `docs/handoff.md`.
3. **Simplicity and modularity are product features.** Prefer the Go standard
   library. Every new dependency is a decision that deserves a line of
   justification in the PR description. UI follows the bars in
   `docs/benchmarks.md`. Substantial features first check
   `docs/benchmarks/` (Cursor, t3code, paseo) and cite an adaptation.
4. **Honesty over polish.** Report what is actually done vs. described. A
   smaller true changelog beats an impressive false one. Unknowns go into
   `docs/handoff.md` as open questions, not into prose as facts.
   **Seeing a visual defect and shipping it as done is a violation.**
   Fix it or say FAIL. `eval` / DOM JSON is not a visual verdict.
5. **Isolated git worktree.** Two agents must not share a working tree.
   Never commit feature work on `main` in the primary checkout. Start from
   current `main`:
   `git worktree add .worktrees/<name> -b feat/<name>`
   After the branch merges: `git worktree remove .worktrees/<name>` and
   delete the branch. Leave `main` clean for the next session. A dirty
   shared tree that blocks another agent is FAIL.
   This is **enforced by git, not by trust**: `.githooks/reference-transaction`
   aborts any `git switch`/`git checkout` that would move the root checkout
   off `main` (switching back to `main` is always allowed), and
   `.githooks/pre-commit` refuses feature commits made there. The guards are
   tool-agnostic — they hold for every agent runtime, editor and script.
   `make hooks` (implied by `make dev` and `make ci`) points git at them and
   **fails** if `core.hooksPath` was redirected elsewhere; `make hooks-check`
   (in `make ci` and in GitHub CI) runs `scripts/hooks-selftest.sh`, which
   proves the whole policy on a throwaway repo — refusals *and* the flows
   that must keep working. A clone that never ran make has no guard.
   Deliberate one-off: `PICODE_ALLOW_SWITCH=1 git switch <branch>`.
   **Never run `git clean -fdx` (or `-fdX`) in the primary checkout:**
   `.worktrees/` is git-ignored, so clean deletes every agent's working
   tree in one stroke. Untracked leftovers are removed by name, or not at all.

6. **Decisions are provisional.** Every ADR, "Refuse" table and architectural
   constraint here records a choice that was right when it was made — not a
   law. Never answer a request with "that is impossible" or "that is
   structural" because a document says so. Name the decision, explain the trade
   it made, say what changing it would cost, and **wait for the owner's
   approval**. Routine implementation choices stay with the agent; anything
   that alters a documented decision, or declares something permanently
   refused, is the owner's call. A constraint nobody has re-measured is a
   candidate for re-measuring, not a fact.

## Quality gates (before you say "done")

```bash
make fmt-check    # gofmt clean
make vet          # go vet clean
make test         # Go tests pass
make test-js      # frontend tests pass
make build        # UI (npm) + binary builds
```

Use the skill: `/skill:quality-gate` (interactive checklist).
When a change has **interacting conditions that change the outcome**
(delete, restore, auth, cascade, run mode, permissions), write a
**decision table** before claiming done: each row is conditions → action.
Tests must cover every row, or the untested row is named as FAIL/debt in
`docs/handoff.md`. Two happy-path clicks are not coverage. Skip the table
for polish, copy, and single-path fixes.
**Motion and optimistic UI** are the default for state that takes time
(jobs, overlays, lists). Enter / step / exit must move. A static flash
then “all done” is FAIL. Prefer showing the next state immediately and
reconciling when the server answers.

For any UI work:
1. `read` `.pi/skills/uiux-review/SKILL.md` **before** the first JSX/CSS edit.
2. Empty / blocked / error states are first-class: one line + one action.
   A lecture, npm spec, or blank well is FAIL.
3. Before done, `read` `.pi/skills/visual-review/SKILL.md`. Screenshot of
   those states must be `read`. After overlays, `window.__picodeOverlayAudit()`
   must be `ok`. Answer the 5-question visual-card in the reply.
4. Skip or FAIL on visual-review → do not commit, do not say shipped.
   `eval` / DOM JSON is not a visual verdict.
At session end, run `/skill:handoff-update`.

## Commands

| Command | What it does |
|---|---|
| `make dev` | Run the Go server — reads the UI from disk; run `make web` once first (ADR-0023) |
| `make ui` | Vite HMR on :5173 (proxies API to the Go server) |
| `make web` | Build React UI → `internal/web/public` |
| `make build` | UI + `bin/picode` |
| `make install` | Copy to `~/.local/bin` and enable systemd --user |
| `make deploy` | Rebuild + restart the installed service |
| `make desktop-restart` | Swap the Windows tray + native-host exes and relaunch the tray via the logon task — the only supported restart; never background a Windows exe from WSL |
| `make test` / `make test-js` / `make vet` / `make fmt` | Quality gates (Go tests, frontend tests) |
| `make ci` | Everything CI runs |

## Repo map

```
AGENTS.md          this contract
docs/              living documentation (handoff.md = project state; guidelines.md = how to write docs)
www/               public docs (VitePress Markdown → GitHub Pages)
docs/decisions/    ADRs — one decision per file, immutable once accepted
docs/screenshots/  committed visual evidence (see its README)
.pi/               Pi harness: skills, project settings
cmd/picode/        entrypoint
ext/               Chrome MV3 extension, sideload (ADR-0043)
internal/browserhost/  native-messaging host + Chrome install
internal/server/   HTTP server + API
internal/web/      UI loader: from disk by default, embedded with `-tags embedui` (ADR-0023).
                   public/ is Vite output and is NOT committed
web/               React + Vite + Tailwind sources (ADR-0008)
.github/           CI
```

## Architectural decisions

Significant choices (frameworks, protocols, persistence, security model) go
through an **ADR**: copy `docs/decisions/template.md`, number it, argue
context → decision → consequences. Never silently contradict an ADR —
supersede it with a new one instead.

## Style

- **Language policy: English.** The repository's official language is
  English — code, comments, docs, commits, changelog entries, issues and
  PR descriptions. No exceptions for canonical content.
- Go: idiomatic, stdlib-first, table-driven tests, no `init()` magic.
- UI: React in `web/`; design tokens live in `web/src/styles/app.css`
  (do not invent a second palette). After any UI change run `make web`
  and a JS/JSX syntax check (`npm run build` must succeed).
- **Forms: Zod, never native browser validation.** Schemas live in
  `web/src/lib/schemas.js`. Forms set `noValidate`. Same messages in every
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
  subscribe to the feed (`web/src/lib/feed.js`) and patch or refetch; a
  new `setInterval` against `/api/*` needs a reason the feed cannot cover
  (metrics, presence, tmux).
- Commits: imperative, scoped (`server: add /api/version endpoint`).
- Docs: short paragraphs, tables for comparisons, diagrams over prose.
- The audience includes terminal-averse users: UI copy avoids jargon;
  when a technical term is unavoidable (PTY, RPC), a tooltip explains it.
