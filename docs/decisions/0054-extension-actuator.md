# ADR-0054: Extension actuator — the agent acts on the tab through a granted, visible loop

- **Status**: accepted
- **Date**: 2026-09-02
- **Amends**: nothing; fills the hole ADR-0043 deliberately left ("v1 is
  sensor-only… actuating the page is a later track")

## Context

Track A/B shipped: the side panel sends the current tab to an existing
agent. The loop ends there — the agent can *read* about the page but not
*act* on it. The roadmap's Track C is that actuator. Constraints that
shape it:

- **MV3 is not Playwright.** The service worker sleeps; content scripts
  are weaker than CDP. Long, unattended jobs stay on the isolated
  Chromium (`agent_browser`). The actuator exists for "the page I am
  looking at, right now".
- **The agent runs remotely.** Since ADR-0050 the extension may talk to a
  PiCode on another machine, so actions cannot be files in a workspace or
  CDP on localhost — they must travel the existing native-messaging →
  server channel.
- **Trust is the human's, not the model's.** ADR-0036's rule: a third
  party never draws the authorize button. Here, the extension — on the
  human's machine — owns the per-origin grant and the visible execution.
- `messageRun` (ADR-0045 amendment) already proved the shape "send turn →
  observe settle → use the final text"; `RunObserver.OnSettled` delivers
  the assistant's real final message.

## Decision

The agent acts through **action batches** parsed from its own reply, and
the human's extension executes them **visibly, on the granted tab**:

1. **The prompt declares the contract.** A send with "Let the agent act"
   appends a `[browser-act]` block: reply with one ```picode-act fenced
   JSON (`click`, `fill`, `press`, `read`, `scroll`; CSS selectors; ≤12
   actions). Omitting the block is a plain answer — loop ends.
2. **Server parses and stores, never executes.** After the turn settles,
   the last ```picode-act block is validated and stored as a batch in
   SQLite (`act_batches`, migration 021) — one pending batch per agent,
   claimed at most once, expiring unclaimed after 10 minutes. The server
   has no page access; it is a post office.
3. **The extension executes, one action per injection**, each with a
   ~250 ms highlight on the element before it fires — the human sees
   every step. `chrome.scripting.executeScript` on demand (no `<all_urls>`
   content script). A navigation aborts the rest with "page changed".
4. **The grant is per origin, in the extension.** First batch for an
   origin asks: "Let PiCode act on <origin>?" — Allow / Not now. Grants
   live in `chrome.storage.local`, revocable; a batch whose origin is not
   granted, or whose tab is elsewhere, is not executed and simply waits
   or is stopped. Not now ends the loop.
5. **Results feed back as one more turn.** Outcomes return to the agent
   as a follow-up (`[browser-act result]`), which may answer or emit the
   next batch. Hard cap: **3 rounds**. Stop (human) ends the loop
   immediately. The agent is never stopped by the loop — it is the
   person's agent, not the loop's.
6. **Honest lifetime.** The side panel drives the polling; panel closed
   means paused — a pending batch expires after 10 minutes and a reopened
   panel says so. No background scraping, no persistent port, no
   `chrome.debugger`.

## Refuse

| Asked for | Refused because |
| --- | --- |
| Server-side execution (CDP from the daemon) | The page lives in the human's browser on another machine; and server-driven DOM access is the `chrome.debugger` power without its consent UI |
| Persistent native-messaging port to keep the SW alive | Complexity now; one-shot `sendNativeMessage` polling from the open panel is enough and pauses honestly |
| `<all_urls>` declared content script | Inject on demand after a human gesture; never blanket-inject |
| Silent execution (no highlight) | A click the human cannot see is a trust bug, not a UX nit |
| Round cap higher than 3 / unattended loops | This is the boundary against MV3-as-Playwright; long jobs belong to `agent_browser` |

## Consequences

- **Easier:** "fill this form", "click through this wizard", "read this
  view" work in the browser the human already lives in, with their SSO.
- **Harder:** the model must emit well-formed ```picode-act JSON; bad
  blocks just end the loop (the text answer still arrives). The loop is
  slower than CDP automation and requires the panel open — stated in the
  guide, not hidden.
- **If wrong:** batches are inert rows; deleting the routes, the table
  and the panel section removes the feature without touching Track A.

## Alternatives considered

- **Agent-driven CDP against the user's Chrome** (`chrome.debugger`):
  strongest automation, but the yellow infobar is the only consent, and
  it is exactly the power ADR-0043 refused for v1.
- **Human-only actuation** (panel buttons "click this"): no leverage; the
  agent cannot chain steps, so the feature would not earn its grant UI.
- **WebSocket push to the extension**: the server cannot reach the
  extension; the extension must poll. Feed/SSE (ADR-0048) stays an
  in-browser client mechanism.
