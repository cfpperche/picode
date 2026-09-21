# ADR-0172: session-browser

- **Status**: accepted
- **Date**: 2026-09-21
- **Boundary**: security model — an identified principal may open and drive the work-browser split bound to its own session at the act tier, on any http(s) URL, without a stored grant. Amends ADR-0134 (the read-only default no longer gates that tab) and ADR-0135 (the agent may open the split; the human closing it is still the revoke).

## Context

The work browser is the tab the human and the agent share: the agent drives, the human clicks, signs in and sees the result. ADR-0134 made the default read-only on whatever tab is on screen, and act (navigate, click, type) required a stored grant plus a domain list. ADR-0135 let only the human open the split. An agent that needed to open a login page was refused, and opened Chromium in the environment instead.

That refusal mixed two jobs. Headless browsing — `agent-browser` in this project, Playwright, a vendor's own browser — is unattended and stays outside this browser. The session tab is the other job: co-visible, bound to one principal, revoked by closing the split.

## Decision

An identified caller (an agent id, else `term:<id>`) may drive the split bound to its own session. The verbs `open`, `navigate`, `click`, `type`, `press`, `evaluate`, `snapshot`, `screenshot` and `events` run there at act, on any http or https URL. `open`, and any of those verbs when no split exists yet, asks the desktop page to open the split beside that session and select it. Commands never fall through to another tab.

The stored read/act grant and the domain list do not gate this tab. Closing the split ends the binding; the next call may open a new one. The machine switch still turns the built-in browser off. History and raw CDP stay on their own gates (ADR-0146, ADR-0144): raw CDP still needs Developer mode and the full tier.

A caller with no identity cannot open or drive a session tab. Its read verbs stay on the tab on screen, read-only (ADR-0134, that case only). Headless tools are not this decision and are not intercepted.

## Consequences

Easier: the agent opens the browser beside its session and drives the page the human is watching, including a login the human completes in that tab. Harder: a stored domain list no longer confines the session tab — the human's eyes and the close button are the boundary, and a compromised agent can navigate that one tab anywhere on the web while the split is open. The blast radius is the work profile's cookies for sites loaded in that tab, which the human can see. Turning the machine switch off, or closing the split, stops the next call. If the rule is wrong, the cost is one tab the human watched, not a second invisible Chromium.

## Alternatives considered

- Keep the stored grant and only auto-open the split. Rejected: a read-only tab, or a domain list that blocks the login page, is why the agent opens Chromium.
- Give the session tab the full tier (raw CDP, cookies, interception). Rejected: click, type and navigate are the job the human named. Raw protocol stays an explicit grant.
- Intercept `chromium` on PATH and retire headless tools. Rejected: headless work is a different job and stays on the tool the runtime already has.
