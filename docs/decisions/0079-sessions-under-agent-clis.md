# ADR-0079: Sessions live under Agent CLIs

- **Status**: accepted (owner approved the move and the multi-CLI roadmap, 2026-09-05)
- **Date**: 2026-09-05
- **Extends**: ADR-0069 (Agent CLI terminals)

## Context

Sessions today are a Pi-only top-level surface: `#/sessions` (every Pi
session on the machine) and `#/sessions/<workspaceId>` (one folder), backed
by `internal/session` reading pi JSONL under `~/.pi/agent/sessions/`. Agent
CLIs (`#/clis`, ADR-0069) is the multi-CLI surface — pi, Claude Code, Codex
and Grok — but only owns terminals and launch settings. The direction is a
multi-CLI ADE in which session search and presentation must cover the other
CLIs too; a Pi-branded top-level Sessions route is the wrong home for that.

Two options were weighed: keep Sessions top-level and add per-CLI sections
later, or move Sessions under Agent CLIs now and grow per-CLI sources on the
new address. Moving later would churn the same routes twice.

## Decision

Sessions become a capability of a CLI, not a top-level surface. The desktop
routes are `#/clis/sessions` (machine-wide, grouped by folder) and
`#/clis/sessions/<workspaceId>` (one folder), rendered as the third tab of
the Agent CLIs surface. Old `#/sessions*` deep links render the Agent CLIs
shell and redirect to the new addresses; `go("sessions")`, the sidebar
folder icon and dashboard top-session links emit the new hashes. No API
changes in this step: the housekeeping endpoints (`/api/sessions/all`,
`/api/workspaces/{id}/sessions/manage`, `/api/pi-sessions*`,
`/api/session-cleanup`) keep their paths and behavior.

Listing, searching and presenting sessions of CLIs other than pi is future
work on this same address, expected to introduce per-CLI session sources
(`GET /api/clis/{id}/sessions`) and per-CLI open actions (resume in a CLI
terminal). Managed agents and chat replay remain Pi-only, per ADR-0069.

## Consequences

One surface now answers "what did my CLIs do", and adding a CLI's session
source needs no new IA. Bookmarks and docs links to `#/sessions*` keep
working through the redirect. The cost: the Agent CLIs page is no longer
only about launching terminals, and the sessions view mounts inside it (it
brings its own page frame). Breaking the API later is a second, separate
migration flagged here in advance.

## Alternatives considered

- **Keep `#/sessions` top-level forever.** Lost: the multi-CLI sessions
  story would either fork into a second surface or slowly repurpose a
  Pi-branded route.
- **Sessions only inside each CLI's detail page.** Lost: the machine-wide
  cross-folder search that exists today would sit two clicks deep behind a
  CLI choice.

## References and adaptation

- [Cursor command palette / feature surfaces](../../docs/benchmark-cursor.md):
  one surface per capability family, deep links preserved by redirect.
- ADR-0069 supplied the tabs-and-catalog shell this slots into.

## Amendment — 2026-09-11: Sessions live in the selected CLI's pane

Owner chose the alternative this ADR refused: Sessions leave the Agent CLIs
strip and sit next to Launch and Terminals inside the selected CLI's page.
The capability decision stands. Chrome and addresses change.

Canonical hashes are `#/clis/<cli>/sessions` (every folder, grouped by cwd)
and `#/clis/<cli>/sessions/<workspaceId>` (one folder). `#/clis/<cli>` is
Launch; `#/clis/<cli>/terminals` is that CLI's terminal list. The catalog is
the CLI picker, so machine-wide search is one click, not two. Old
`#/clis/sessions*`, `?cli=` and `#/sessions*` rewrite onto the new path.
Dashboard top-session rows name the CLI in the hash; the workspace card
Sessions menu stays Pi.

Settings, Packages and Messages remain strip tabs (ADR-0101/0102). Providers
joined the CLI pane on 2026-09-11 (ADR-0103 amendment).

## Amendment 2026-09-25 — old addresses retired (owner)

The owner retired the compatibility addresses this ADR kept. `#/sessions*`, `#/clis/sessions*` and `?cli=` no longer rewrite; the only Sessions address is `#/clis/<cli>/sessions[/<workspaceId>]`. `#/clis/terminals` no longer redirects either. An old bookmark now lands where any unknown address does.
