# ADR-0117: remove the browser surface

- **Status**: accepted (owner verdict after real-world testing — "feature
  completamente reprovada, vamos remover o browser surface do projeto" —
  2026-09-10)
- **Date**: 2026-09-10
- **Boundary**: protocol and security model — the `/ws/browser` proxy and the
  `/api/agents/{id}/browser-input` consent route leave the daemon's public
  surface, and the agent tab loses its third view. Supersedes ADR-0114 and
  ADR-0115 (both retained as history; their benchmark study
  [docs/benchmarks/2026-09-10-browser-surface.md](../benchmarks/2026-09-10-browser-surface.md)
  also stays as evidence).

## Context

The surface shipped through three phases in one day (watch-only view,
consent-gated control, design pick) and went to production. The owner tested
the real flow against a live agent (CPO) and rejected the feature outright.
A real defect was found and fixed along the way (ADR-0114's name derivation
cut hex-char hashes as byte-slices, so discovery never matched a genuine
rendezvous — caught by the owner's test, not by the suite), but the verdict
is about the feature as a whole, not that defect.

## Decision

**Remove the browser surface entirely**: the Browser view of the agent tab,
`GET /ws/browser`, the rendezvous discovery in `internal/rpc`, the
`/api/agents/{id}/browser-input` consent route, the surface UI and its
shared helpers, and the phase-2 `/browser-input` consent mirror additions to
pi-browser-capture. The changelog fragments of the never-released phases are
deleted.

What stays: the capture sidecar and chat pill (ADR-0082 — it predates the
surface), the agent's `agent_browser` tool, ADR-0114/0115 as history, and
the benchmark study.

## Consequences

**Easier:** the agent tab returns to Chat | TUI; no second WS transport, no
consent mirror, no engine-rendezvous coupling in the daemon. The
`piab-<slug>-<id>-<hash>` naming knowledge lives only with the sidecar
again.

**Harder / accepted costs:** pointing-at-code and live view are gone — if
the idea returns, the ADRs and the study document everything learned,
including the hex-slice trap (name hashes are 12/16 hex CHAR slices, not
bytes) and the self-confirming-test failure mode (both sides of the scratch
E2E shared the same wrong derivation).

**Who breaks if we are wrong:** nobody — the feature is off; agents keep
driving browsers exactly as before ADR-0114.

## Alternatives considered

- **Keep it watch-only (drop control/design)**: the verdict was about the
  feature, not a phase; half-keeping keeps the proxy transport and the
  rendezvous coupling — the parts with the maintenance cost.
- **Park behind a flag**: a rejected feature behind a flag is code to
  maintain; removal is the honest state.
