# ADR-0134: browser-default-policy

- **Status**: accepted (owner, 2026-09-13 — "vamos de A", agreed in session)
- **Date**: 2026-09-13
- **Boundary**: security model — the access an agent has to the work browser
  by default, and what identity that decision rests on. Amends ADR-0128
  (its "deny by default" stands for everything unnamed; this fixes the
  *default value* of an agent's policy).

## Context

ADR-0128's work browser now has both halves: the shell's bridge with the
method catalog per tier (`feat/browser-cdp`) and the daemon↔shell command
channel (`feat/browser-agent`, ADR-0132). What has no answer yet is what an
agent may do *before anyone grants it anything*, and that answer decides
whether the first `browser` tool is usable on the day it ships or inert until
a policy editor exists.

Facts that shape the choice:

- The page the human is looking at is one the **human opened**; reading it is
  the product's core promise (the ChatGPT Work benchmark does exactly this,
  and its default needs no grant).
- The tier catalog already denies everything unnamed (ADR-0128 item 3): read
  has no execute path, so a read default cannot act, navigate or inject.
- Identity is an assertion, not proof: a Pi tool reports `PICODE_AGENT_ID`
  from its process env (`internal/store`, ADR-0037) and authenticates with the
  install token. Any local process holding that token can claim any agent —
  the token, not the id, is the trust boundary. A per-agent policy therefore
  shapes **cooperating** agents (which agent may do what, as product control);
  it is not a sandbox against a hostile local process. A per-agent token would
  be the fix if that ever matters, and it is a new credential — its own ADR.

## Decision

1. **An agent with no explicit grant has `read` on the work-browser tab the
   human has on screen** — and nothing else. This is the default value of the
   `{tier, domains}` policy ADR-0128 defines; the deny rules themselves are
   unchanged.
2. **`act` and `full`, and any origin other than the tab on screen, require an
   explicit per-agent grant.** Grants persist as one setting per agent; the
   editor that writes them (Settings ▸ Browser, ADR-0128's table) is slice 4.
3. The read default is applied by the daemon when it resolves a tool call:
   the target is the active tab (the command carries no tab id, ADR-0132), and
   the shell re-checks the method catalog for `read` as it already does.
4. The tool's own copy says what reading means: the page's content reaches the
   model. A page can hold secrets the human did not intend to share with an
   agent, so the first call names the page it read.

## Consequences

- The feature works the day the tool ships; no inert period waiting for a
  settings screen, and no "grant everything" prompt in the way of the one
  thing the product is about.
- Read again cannot act: no `Runtime.evaluate`, no `Input.*`, no navigation
  (ADR-0128's catalog). A grant is the only path upward, and it is visible and
  revocable where the agent is configured.
- Accepted risk: an agent can read whatever the human happens to have on
  screen. The remedy is one keystroke — the human closes the tab or switches —
  and the tool reports which page it read.
- Accepted limit: an agent that lies about its `PICODE_AGENT_ID` gets another
  agent's grants. The token is the boundary today; if per-agent proof is ever
  required, it arrives as a per-agent token in its own ADR.
- The navigation gate (ADR-0128 item 4) stays required for the grants: with
  `act` granted for a domain, a click that redirects off it must cancel before
  the page loads.

## Alternatives considered

- **Deny everything until the policy editor exists** (ADR-0128 read literally).
  Rejected by the owner: it makes the tool inert and the slice-4 editor a
  prerequisite for the first useful call.
- **Read any work-browser tab, not only the one on screen.** Rejected: the
  human's attention is the consent signal, and a background tab is not
  something they are looking at.
- **Mint a per-agent token now** so the id cannot be claimed. Rejected for
  this step: a second credential and a second authentication path for the
  same privilege, to defend against a local process that already holds the
  install token. Available later, deliberately.
