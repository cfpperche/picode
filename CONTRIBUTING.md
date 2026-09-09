# Contributing to PiCode

Humans and Pi agents contribute under the same contract. The full operating
rules live in [AGENTS.md](AGENTS.md) — this file covers the mechanics.

## For humans

1. Fork / branch, do your work, keep changes small and reviewable
   (benchmark: one logical change per PR).
2. `make ci` must pass locally (gofmt, vet, tests, build).
3. PR description must include:
   - What changed and why (link an issue when one exists).
   - Justification for any new non-stdlib dependency.
   - Doc updates that traveled with the change (they must — see below).
4. UI changes: run the `/skill:uiux-review` checklist mentally or attach
   a screenshot; the benchmarks in [docs/benchmarks.md](docs/benchmarks.md)
   are enforced in review. Prefer Radix / cmdk / native controls over a
   new homemade widget (see AGENTS.md Style).

## For Pi agents

1. Read `AGENTS.md` (auto-loaded) and `docs/handoff.md` (≤ 100 lines)
   **before** starting; then only what AGENTS.md's reading table names.
2. Use the skills: `read` `/skill:uiux-review` **before** the first UI edit,
   `/skill:visual-review` before calling UI done, `/skill:quality-gate`
   before declaring done, `/skill:handoff-update` for the session note.
3. Never break the build: `make ci-scoped` while iterating, `make close`
   at the end, `make ci` on `main` for the merge. A boundary change
   (protocol, persistence, security, process) → ADR (`make adr NAME=x`);
   a UI refinement never needs one.
4. **Work in an isolated git worktree** (`make worktree NAME=<name>`,
   AGENTS.md non-negotiable 5). Do not edit `main` in the primary checkout
   in parallel with another agent. After merge, `make worktree-gc`.
5. **Do not deploy** (ADR-0105): the owner runs `make deploy` when they
   want it; a branch is done when `main` fast-forwards to it.
6. **One branch, one session** (ADR-0105): end the session at the merge and
   write the closing docs from `make close-summary` in a subagent or a
   fresh session, never at the peak context of the working session.
7. Code under `packages/pi-roles/`, `packages/pi-inbox/`, `packages/pi-checklist/`,
   `packages/pi-compact/`, and `packages/pi-diff/` is MIT; everything else is PolyForm
   Noncommercial. See [LICENSING.md](LICENSING.md) and ADR-0028.

## The documentation contract (applies to everyone)

Code and docs change together, in the same commit:

| You changed... | Then also update... |
|---|---|
| Behavior or architecture | the `docs/architecture/` file for that subsystem (+ ADR if it crosses a boundary) |
| Anything user-visible | `docs/changelog.d/<branch-slug>.md` — `make changelog` assembles `CHANGELOG.md` on `main` |
| A slash command users can type | `docs-site/commands.md` heading `{#id}` + pi correlation per [docs/guidelines.md](docs/guidelines.md) |
| Project state at all | `docs/handoff/<date>-<branch>.md` (≤ 25 lines) + `docs/handoff.md` kept true (≤ 100 lines, ≤ 8 KB: in flight, next up, debts — never shipped work) |
| A benchmark we hold | `docs/benchmarks.md` with rationale |

## License of contributions

PRs are [PolyForm Noncommercial 1.0.0](LICENSE) **and** grant the
copyright holder the right to dual-license (including a commercial
license). See [LICENSING.md](LICENSING.md). Do not send code you cannot
offer on those terms.

## Language

English is the repository's official language — code, docs, commits,
changelog, issues and PRs (policy in [AGENTS.md](AGENTS.md)). Reviewers
should request translation rather than merge non-English content.

## Releases

Maintainers cut releases: version tags follow SemVer; every release
compiles its section from the changelog. No release without a green CI. The
operational checklist is in [docs/release-process.md](docs/release-process.md);
the proposed cadence and release lanes are tracked in
[ADR-0064](docs/decisions/0064-release-cadence.md). Neither sets official dates
until the proposal is accepted.

## Frontend applications

`web/desktop` and `web/mobile` own their UI and build independently (ADR-0072).
From `web/`, use `npm run dev:desktop` or `npm run dev:mobile` for one app;
`make ui` runs both with the root launcher. `npm run build:desktop` and
`npm run build:mobile` preserve sibling output. `make build` assembles the
complete Go release. Shared contracts and theme tokens live in `web/shared`;
import only its explicit package exports. A UI change belongs to its app,
while a shared contract change requires both client suites.
