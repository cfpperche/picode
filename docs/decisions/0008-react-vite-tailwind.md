# ADR-0008: React + Vite + Tailwind for the web UI

- **Status**: accepted
- **Date**: 2026-08-24
- **Supersedes**: [ADR-0004](0004-defer-frontend-framework.md)

## Context

ADR-0004 deferred a frontend framework while the UI was a single vanilla
file (~350 lines, then ~800). By M2 the surface has agent tabs, a managed
chat panel, a tmux dock, settings, theme, and port rebind — plus a growing
list of `[hidden]` vs `display` bugs that come from hand-rolled DOM.

The owner asked to adopt **React + Vite + Tailwind** and keep the **current
design system 100% faithful** (tokens, density, anatomy). ADR-0001 still
holds: one Go binary, UI embedded via `go:embed`.

Thresholds in ADR-0004 (1k lines / 3 views / reactive state tax) are met.

## Decision

The UI lives in `web/` as a **React 19 + Vite + Tailwind CSS v4** app.
Production assets build into `internal/web/public` and ship inside the
binary (`go:embed`, ADR-0001 unchanged).

Design tokens stay CSS variables (`--bg-base`, `--accent`, …) — the same
values as the vanilla UI. Tailwind `@theme` maps onto those variables so
new utilities cannot drift from the palette. Existing component classes
(`btn`, `composer`, `dock`, …) are preserved in `web/src/styles/app.css`.

`make build` / `make ci` run `npm ci && npm run build` in `web/` before
the Go compile. Node 20+ is now a build-time dependency, not a runtime one.

## Consequences

- **Easier**: component model, HMR (`make ui` + running `picode`), xterm
  as a real npm dependency instead of a vendored file, fewer
  display-vs-hidden classes of bugs.
- **Harder**: contributors need Node to change the UI; `go build` alone
  embeds whatever was last written to `internal/web/public`. The contract
  is `make build`, not raw `go build`.
- **If wrong**: the vanilla tree is gone after this ADR; reverting means
  restoring from git history. The token file is the fidelity contract —
  visual drift is a bug, not a restyle.

## Alternatives considered

- **Keep vanilla (ADR-0004)**: rejected by owner; complexity already
  crossed the documented thresholds.
- **Svelte + Vite**: rejected — owner specified React.
- **React without Tailwind**: rejected — owner specified Tailwind; tokens
  still own the palette so Tailwind cannot invent a second system.
- **SPA history API + Go rewrite**: rejected for now — hash routes
  (`#/settings`) keep the Go file server unchanged.
