# ADR-0135: agent-browser-binding

- **Status**: accepted
- **Date**: 2026-09-14
- **Boundary**: protocol — the work-browser command channel (ADR-0132) resolves
  its target tab by an explicit agent↔tab binding instead of "the active tab of
  the shell window"; interaction — a split pane keeps the bound tab co-visible,
  which amends ADR-0134's on-screen consent rule.

## Context

ADR-0132/0134 route a work-browser command to the work-browser tab on screen.
Real use broke that contract: for an agent to work on a page, the human must
park on the browser tab — the agent stops receiving messages the moment the
human returns to the terminal (verified live on 2026-09-14: a snapshot request
returns 502 "no work-browser tab is open" while the human reads the terminal).
The market pattern is side-by-side
(docs/benchmarks/2026-09-02-live-browser-preview.md): Cursor, Antigravity and
Devin all run the agent and the browser it drives as two visible panes.

## Decision

An agent's terminal pane offers "Open browser" in its context menu. Choosing it
splits the agent's tab: the agent pane and a work-browser pane, both visible.
The browser pane binds 1:1 to that agent: while the binding exists, ADR-0132
commands from that agent resolve to the bound tab — not to the window's active
tab. Closing the browser pane ends the binding (closing is revoking). Closing
the agent's tab detaches the binding but leaves the browser tab alive as a
plain web tab. Tabs without a binding keep ADR-0134's on-screen rule.

## Consequences

Easier: the obvious workflow works — the human watches the agent drive a page
while talking to it. The consent story gets tighter, not looser: the human
sends the agent to the tab explicitly, and the tab stays co-visible in the
split, so "the agent sees what I see" survives as an invariant. Harder: the
channel gains a second target-resolution rule (binding first, active-tab
fallback) that every tool call must respect, and a split pane is a second live
WebView2 rect to keep bounded on resize. If the 1:1 rule is wrong, multi-tab
binding is additive — a binding map, not a schema change.

## Alternatives considered

- Keep ADR-0134 and ask the human to stay on the browser tab — rejected: it
  makes agent-and-browser mutually exclusive with conversation, which is the
  exact failure observed.
- Floating browser window bound to the agent — rejected: a separate window
  loses the co-visibility that justifies the grant, and WebView2 child-window
  bounds across windows complicate the shell.
- Detach the chat (let the agent answer while its browser tab is active) —
  rejected: it does not compose with ADR-0134's consent story and hides the
  work the agent is doing.
