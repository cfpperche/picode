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

## The non-negotiables

1. **Documentation is a living system.** Code changes and documentation
   changes travel together, in the same commit. Specifically:
   - Behavior/architecture changed → update `docs/architecture.md` **and**
     add/revise an ADR in `docs/decisions/` if the decision is architectural.
   - Anything user-visible changed → add an entry to `[Não lançado]` in
     `CHANGELOG.md` (Keep a Changelog style).
   - **Every session that changes state MUST end by updating
     `docs/handoff.md`** (current state, what's in flight, next steps, debts).
     The handoff is how the next agent (or human) picks up the work.
2. **Never break the build.** Before ending work, run the quality gates
   (below). `make ci` must pass. If you can't finish something, leave the
   tree compiling and tests green, and record the gap in `docs/handoff.md`.
3. **Simplicity and modularity are product features.** Prefer the Go standard
   library. Every new dependency is a decision that deserves a line of
   justification in the PR description. UI follows the benchmarks in
   `docs/benchmarks.md`.
4. **Honesty over polish.** Report what is actually done vs. described. A
   smaller true changelog beats an impressive false one. Unknowns go into
   `docs/handoff.md` as open questions, not into prose as facts.

## Quality gates (before you say "done")

```bash
make fmt-check    # gofmt clean
make vet          # go vet clean
make test         # tests pass
make build        # binary builds
```

Use the skill: `/skill:quality-gate` (interactive checklist).
For any UI work, also run `/skill:uiux-review` against `docs/benchmarks.md`.
At session end, run `/skill:handoff-update`.

## Commands

| Command | What it does |
|---|---|
| `make dev` | Run the dev server on :7331 |
| `make build` | Build `bin/picode` |
| `make test` / `make vet` / `make fmt` | Quality gates |
| `make ci` | Everything CI runs |

## Repo map

```
AGENTS.md          this contract
docs/              living documentation (handoff.md = project state)
docs/decisions/    ADRs — one decision per file, immutable once accepted
.pi/               Pi harness: skills, project settings
cmd/picode/        entrypoint
internal/server/   HTTP server + API
internal/web/      embedded UI assets (go:embed)
web/               (future) frontend sources
.github/           CI
```

## Architectural decisions

Significant choices (frameworks, protocols, persistence, security model) go
through an **ADR**: copy `docs/decisions/template.md`, number it, argue
context → decision → consequences. Never silently contradict an ADR —
supersede it with a new one instead.

## Style

- Go: idiomatic, stdlib-first, table-driven tests, no `init()` magic.
- Commits: imperative, scoped (`server: add /api/version endpoint`).
- Docs: short paragraphs, tables for comparisons, diagrams over prose.
- The audience includes terminal-averse users: UI copy avoids jargon;
  when a technical term is unavoidable (PTY, RPC), a tooltip explains it.
