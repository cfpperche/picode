# ADR-0088: Install an agent CLI that is missing

- **Status**: accepted (owner approved closing the lifecycle cycle, 2026-09-07)
- **Date**: 2026-09-07

## Context

ADR-0087 shipped update checks, update, reinstall and uninstall for installed
agent CLIs. A CLI that is not installed shows "Not found" with only a docs
link — the lifecycle cycle has no first step. The owner asked to close the
cycle. npm-backed CLIs (pi, codex, claude-code) have a documented npm install
whose argv is byte-identical to the already-accepted reinstall action
(`npm install -g <pkg>@latest`). Grok and Hermes Agent install through vendor
`curl … | bash` scripts, which PiCode will not execute on the user's behalf.

## Decision

Add the `install` action to the ADR-0087 lifecycle machinery, offered only
when the CLI is **not installed**. `clilifecycle.ForMissing(cliID)` returns an
npm-backed plan for pi, codex and claude-code; the job runs the same npm argv
as reinstall through the same one-lane durable runner, terminal guard and
post-success recheck. Grok and Hermes Agent get a guided install — a card
linking the vendor's official guide; PiCode never executes their curl
installers. An install request for an installed CLI is refused (reinstall
covers that). The Agent CLIs surface shows an **Install** button on the
missing CLI's detail page and the guided card otherwise; the "Update check
failed" line no longer appears for installs PiCode does not manage, and the
real check error text is shown instead of a bare "failed".

This stays inside ADR-0087's accepted boundary: PiCode orchestrates vendors'
own package mechanisms and never edits their configurations. No new argv
class is introduced — install reuses reinstall's.

### Decision table (delta to ADR-0087)

| Conditions | Action / observable result |
|---|---|
| CLI missing + npm-backed (pi, codex, claude-code) | Install button; job runs `npm install -g pkg@latest`; success re-runs the setup check and the CLI shows Installed |
| CLI missing + npm absent on PATH | Job fails with "npm was not found" — the message names the prerequisite |
| CLI missing + vendor curl installer (grok, hermes) | Guided card with the vendor's install guide; no executable action |
| CLI installed + install requested | Refused — reinstall is the action for installed CLIs |
| Update check on an unmanaged install | No "failed" line; the surface communicates the install/docs path instead |

## Consequences

The lifecycle covers creation to removal for npm-backed CLIs. Cost: the
surface now offers a state-changing action on machines where npm may be
absent or stale — the failure mode is a failed job naming the prerequisite,
never a half-install (npm owns the filesystem). Vendor curl installers stay
manual; if a vendor later ships an uninstall/update-shaped command, adapters
extend per ADR-0087's mechanism.

## Alternatives considered

- **Execute vendor curl installers for grok/hermes**: runs remote scripts
  with PiCode as the actor; refuses on principle and offers the guide.
- **Reuse the `update` action for missing CLIs**: job records and UI copy
  would lie about what ran. A distinct `install` action keeps records honest.
