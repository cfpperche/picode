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
   changes travel together, in the same commit. User-facing command help
   lives in `www/` (GitHub Pages), not in the app bundle — see
   [docs/guidelines.md](docs/guidelines.md). Specifically:
   - Behavior/architecture changed → update `docs/architecture.md` **and**
     add/revise an ADR in `docs/decisions/` if the decision is architectural.
   - Anything user-visible changed → add an entry to `[Unreleased]` in
     `CHANGELOG.md` (Keep a Changelog style).
   - **Every session that changes state MUST end by updating
     `docs/handoff.md`** (current state, what's in flight, next steps, debts).
     The handoff is how the next agent (or human) picks up the work.
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

## Quality gates (before you say "done")

```bash
make fmt-check    # gofmt clean
make vet          # go vet clean
make test         # tests pass
make build        # UI (npm) + binary builds
```

Use the skill: `/skill:quality-gate` (interactive checklist).
For any UI work, also run `/skill:uiux-review` and `/skill:visual-review`.
Visual-review is a **gate**, not a courtesy: screenshot must be `read`,
`window.__picodeOverlayAudit()` must be `ok` after overlays, and the
5-question visual-card must be answered in the reply. Any clip/occlusion
is FAIL — do not commit. See `.pi/skills/visual-review/SKILL.md`.
At session end, run `/skill:handoff-update`.

## Commands

| Command | What it does |
|---|---|
| `make dev` | Run the Go server (serves last UI build) |
| `make ui` | Vite HMR on :5173 (proxies API to the Go server) |
| `make web` | Build React UI → `internal/web/public` |
| `make build` | UI + `bin/picode` |
| `make test` / `make vet` / `make fmt` | Quality gates |
| `make ci` | Everything CI runs |

## Repo map

```
AGENTS.md          this contract
docs/              living documentation (handoff.md = project state; guidelines.md = how to write docs)
www/               public GitHub Pages (slash/command help; iframe in #/docs/{cmd})
docs/decisions/    ADRs — one decision per file, immutable once accepted
docs/screenshots/  committed visual evidence (see its README)
.pi/               Pi harness: skills, project settings
cmd/picode/        entrypoint
internal/server/   HTTP server + API
internal/web/      embedded UI assets (Vite output, go:embed)
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
- **Prefer popular primitives over homemade widgets.** Use Radix (already
  in the app), cmdk, **shadcn/ui patterns**, or native controls. Tailwind
  (ADR-0008) is the utility layer; tokens stay CSS variables. Roll a custom
  control only when no library/pattern covers the case — and say why.
  Native `<select>` / `<input>` still beat a one-off styled fake.
- **Empty states are required.** A list, gallery, or collapsible section
  that can have zero items must show a one-line placeholder (and the add
  action if one exists). Never a blank well. Never a "0" count badge.
- Commits: imperative, scoped (`server: add /api/version endpoint`).
- Docs: short paragraphs, tables for comparisons, diagrams over prose.
- The audience includes terminal-averse users: UI copy avoids jargon;
  when a technical term is unavoidable (PTY, RPC), a tooltip explains it.
