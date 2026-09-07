# ADR-0044: The mobile shell is a supervision console, not the desktop shrunk

- **Status**: accepted; amended 2026-09-01 (same day) after the owner's first
  use on a phone — see *Amendment* at the end
- **Date**: 2026-09-01
- **Scope amendment**: file editor/tree and Git exclusions superseded by [ADR-0095](0095-mobile-files-git.md); other decisions unchanged.
- **Presentation/build boundary**: partially superseded by
  [ADR-0072](0072-independent-web-applications.md). The original rationale
  below is retained as history.

## Context

PiCode has had two shells in one Vite app since the phone split
(`web/src/desktop`, `web/src/mobile`, chosen at boot by `max-width: 767px`
or `?mobile=1`). The mobile one was a 381-line reduction of the desktop:
Agents | Chat | Term | More, no routes, only the first agent of each
workspace, a chat that ignored `extension_ui_request` (so a permission
prompt never showed — "Mobile has no waiting card" sat in the handoff
twice), no abort or steer, no way to create an agent, no Inbox
(ADR-0037's whole data plane was unreachable), and a More menu that
mounted Providers and MCPs with empty props. The backlog called the gap
"mobile parity".

The owner reframed it on 2026-09-01: the phone does not need to mirror
the workstation. Its job is to **observe the system, answer the agents'
calls, decide, and act quickly** — create an agent, talk to it, interrupt
it — while away from a desk. The benchmark study
(`docs/benchmarks/2026-09-01-mobile-agent-supervision.md`) found every
shipping product in this space converging on the same shape: Claude
Code's Remote Control, Codex in the ChatGPT app and Cursor for iOS are
all supervision surfaces (approve, redirect, reply, review), none is a
mobile IDE; the on-call reference (PagerDuty) puts the decisions that
need you at the top, sorted by urgency, with the actions under the thumb.

## Decision

**Four tabs and one pushed screen.** `Now` (home): the "Needs you" queue,
then who is running, then today's spend/activity/sessions, then the last
finished runs. `Inbox`: the ADR-0037 app through the same host surface
the desktop uses. `Agents`: every agent, grouped by workspace, with
Start/Stop and a "+" sheet (workspace, free agent, agent in a workspace,
or adopt a pi session). `More`: the rest of the product one tap deep,
mounting the desktop page components with real props. The **agent
screen** is pushed, not a tab: name · state, model · cost · context, the
shared `Conversation` with its ask card, the shared `Composer` (its Stop
is the abort; dictation comes with it), and a Chat | Terminal segment
only for an agent that lives in a tmux TUI.

**Needs-you is a merge of two sources.** Live dialogs first — the agent is
blocked, and pi's dialogs expire — then blocking inbox items, newest
first. Both are answerable in place: a dialog through
`POST /api/agents/{id}/ui`, an inbox item through
`POST /api/inbox/{id}/respond`.

**One server change.** `agentView` (`GET /api/workspaces`,
`GET /api/agents?free=1`) carries `streaming`, `waiting` and the open
`dialog` from the managed runtime's snapshot. The fleet poll the phone
already needs now says who is waiting and what they are asking; no
socket per agent, no new endpoint. The desktop ignores the fields for
now (its sidebar can adopt them later in place of `waitingId`).

**A pure reducer, not a shared handler.** `lib/agentEvents.js` carries the
desktop `handleEvent`'s item-level semantics (snapshot with dialog,
deltas, tools, bash, enqueue, message/turn end, ui request/timeout,
exit, task failed, rejected) as `reduceAgentEvent(state, ev) → {state,
effects}`, on the helpers both shells already share (`lib/askForm.js`,
`assistantMsg.js`, `diff.js`, `searchCards.js`, `piError.js`). The
desktop's inline switch is untouched: zero regression surface there,
and the reducer is the first unit-tested version of that logic.
Recorded debt: the desktop may adopt it.

**Hash routes on the phone.** `#/`, `#/inbox[/id]`, `#/agents`,
`#/agent/<id>`, `#/more[/section]`. The agent route is the desktop's, so
a QR scan or a pasted link opens the same agent; other desktop hashes
map to the closest section. Tabs `replace`, pushed screens `push`, so
Android's Back closes the agent screen instead of leaving the PWA; the
on-screen Back always goes to the parent. No remembered tab — Now is
the home because the question is always "what needs me".

**Explicitly not on the phone.** Editor, file tree, git graph, terminal as
a tab, the desktop's tab strip, and any per-agent context-window fetch
from the home (one `/status` call per running agent; refused until an
aggregate exists). Push notifications are the next ADR (Web Push over
VAPID, Go stdlib), not this one.

## Consequences

- **Easier**: a permission prompt can finally be answered from a phone;
  the Inbox exists on the phone; a stopped agent can be started, a
  running one stopped, a runaway turn aborted, a new agent created —
  each in one or two taps in the thumb zone.
- **Harder**: the mobile shell is now a small app (five screens, four
  hooks, seven components, ~350 lines of CSS) rather than one file; two
  shells share more components with different frames, and a change to
  `Conversation`/`Composer`/`AppSurface` must be looked at on 390px too.
- **Accepted cost**: the agent screen replays the last 200 transcript
  events on open (the desktop's own tail); a very long session on a slow
  link shows the socket's live events first and the history a moment
  later. The fleet poll runs at 5 s on Now/Agents (15 s elsewhere) — the
  ADR-0036 badge poll's own "seconds of latency is fine" reasoning.
- **If wrong**: `main.jsx` still picks the shell by viewport; the old
  `?desktop=1` escape and the More → Desktop layout row remain.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Finish "mobile parity" (file tree, git graph, editor at 390px) | The products people actually use from a phone are all supervision surfaces; editing on a phone is the one thing every one of them refuses (Cursor: "not a mobile IDE"). |
| A native app (AgentWatch / Nimbalyst have Live Activity, watch) | ADR-0003's one-binary posture; the PWA already installs to the home screen, and push (next ADR) covers the alert. Live Activity is the one thing a PWA cannot do — accepted. |
| Extract the desktop `handleEvent` into a hook both shells call | Entangled with six refs and five side effects; a shared mutation would put the desktop at risk to ship a phone feature. The pure reducer gets the tests without touching it. |
| A `GET /api/agents/waiting` endpoint | The phone polls the fleet anyway; three fields on the view it already reads is smaller than a new route and a second poll. |
| Keep Chat and Terminal as tabs | A tab must always have something to show; a chat tab with no agent picked is a dead end (the old shell's "Pick an agent."). Pushed screens carry their agent in the URL. |
| Remember the last tab (PagerDuty does) | The home is the queue of decisions; coming back to Agents when something is waiting is the wrong default for this product. |

## Amendment (2026-09-01, owner's first use)

Two findings from the phone, both accepted:

- **No header.** "PiCode [QR]" spent 48px on every screen and did nothing
  the tab bar does not. Removed; the QR stays in More → *Open on another
  phone*. Pushed screens keep their own header (Back · title · action).
- **Work, not Agents.** One flat Agents tab hid the structure the desktop
  rail already has and dropped terminals entirely. The tab is now
  **Work** with three segments mirroring the rail: *Workspaces* (a card per
  folder with its agents **and** terminals, "+ Agent" / "+ Terminal"),
  *Agents* (free agents — an agent need not belong to a workspace) and
  *Terminals* (free terminals only, New / Remove — a workspace's terminal
  is listed once, on its card, exactly as the desktop rail splits them). The last segment is remembered
  per-viewer. A terminal opens as a pushed `#/term/<id>` screen: the same
  `TermSurface`/xterm the desktop attaches to the tmux session, plus a key
  bar (Esc, Tab, Ctrl+C/D/Z/L, arrows, `/ | - ~`) sending byte sequences
  straight to the pane's socket — without it a phone terminal is
  read-only. Live terminals join Running on Now.

Routes: `#/work[/section]`, `#/term/<id>` (the desktop's); `#/agents` and
`#/terminals` map onto Work.

Second pass, same day: pushed screens (agent, terminal, a More section)
hide the tab bar — the ← header is the way out, and the pane or the
conversation gets the height; the terminal's key bar sits behind a
floating keyboard button rather than always on; the phone shell locks
zoom (viewport meta rewritten on mount so the desktop keeps the browser
default, `touch-action: manipulation`, pinch gestures cancelled); row
actions are icons (▶ ■ 🗑) with `aria-label`s, not words. Back from an agent lands on Work; from a
terminal on Work → Terminals.

## Phase 3 (2026-09-01): polish from the phone

- **Pull-to-refresh** on Now, Work and Inbox (`usePullToRefresh`, a one-line
  indicator, no spinner theatre); it refetches everything the screen reads.
- **Swipe** left on an Inbox row reveals its actions (Done, Snooze, Delete)
  — touch has no hover, and those buttons were hover-only. Desktop unchanged.
- **Changes**, read-only: `#/changes/<a|t|w>/<id>` shows an owner's
  uncommitted working tree through the git graph's own `UncommittedDetail`
  (which learned the workspace owner kind). Entry points: the agent screen's
  state row, the terminal header and the workspace card, each only when the
  tree is dirty, with the count. Nothing here stages, commits or edits —
  Cursor's line ("not a mobile IDE") holds.

Owner's phone pass on phase 3: pull-to-refresh did not fire on the Inbox
(the host surface scrolled inside a non-scrolling screen) and the stacked
list-over-detail overflowed a phone. The Inbox is now two screens, like
everything else pushed on the phone: the tab is the list (tabs, search,
rows — the screen itself scrolls, so pull works), and an item is
`#/inbox/<id>` with the shell's Back header and only that item's panes.
`AppSurface` gained `paneMode: "list" | "detail"` (+ `onOpenItem`) for it;
the desktop's split is untouched. A notification tap lands on the item
screen, and answering it pops back to the list.

## Amendment 2026-09-02 — Safari without the installed web app

The sticky heads (screen head, pushed-screen head, the Inbox filter
head) used a negative top margin plus `top: calc(-12px - env(safe-area-
inset-top))` so the head's own padding would sit under the translucent
status bar of the installed web app. Chrome and Safari place a sticky
box with a negative margin differently, and in Safari's own tab nothing
covers that strip: the head's top was clipped. Now a screen that has a
sticky head hands it the top padding (`:has()`), the head has no
negative margin and sticks at `top: 0`; the safe-area strip is the
head's padding in both modes. The terminal's keys toggle moved from a
floating button (it sat on the TUI's status lines) into the header.

## Amendment 2026-09-02 — terminal keys: the benchmark, not a library

The owner asked for a ready library before more home-grown code. The
research (Termux extra keys, Blink smart keys, mtmux, terminal-web —
xterm.js + tmux like PiCode) found one convergent design and no
package: the popular virtual keyboards (simple-keyboard, KioskBoard)
draw a whole QWERTY for forms and know nothing of terminals or sticky
modifiers. So the bar stays ours and matches the benchmark: one row
that scrolls sideways (nothing shrinks or clips — a flex `<button>`
does not shrink below its label in Safari), **sticky Ctrl/Alt**
(`lib/termSticky.js`, pure: a letter becomes its control byte, Alt
prefixes ESC, bar arrows take the xterm modified form, a five-second
expiry), keys that never summon the phone keyboard with a ⌨ key that
does, the screen sized to the `visualViewport` so the bar rises above
the iOS keyboard and xterm refits, and a hardware-keyboard heuristic
(focus with no viewport shrink) that hides the row.

Follow-up the same evening: the owner chose Termux's layout outright —
two rows of seven equal cells (`ESC / — HOME ↑ END PGUP` over `⇆ CTRL
ALT ← ↓ → PGDN`), flat labels on the terminal's background, nothing
that can overflow. The ⌨ and × keys went: tapping the terminal opens
the phone keyboard, the header icon toggles the grid. Sticky Ctrl/Alt,
the viewport lift and the hardware-keyboard heuristic stay underneath.

## Amendment 2026-09-06 — extra keys are an IME accessory

Owner screenshots on iOS 26: the Termux 2×7 grid and the chat composer
sat under the software keyboard, so typing was invisible, and the grid
could be on while the keyboard was off. Native apps (Fable, Blink Smart
Keys) dock extra keys *to* the IME via `inputAccessoryView`. A PWA cannot
do that, so the bar is faked by pinning `#m-app` to `visualViewport`
(`--vv-height`, `--vv-offset-top`, `--kb-inset`) and showing the keys
only while the terminal host holds focus.

The grid reverts to one horizontally scrolling row (the 2026-09-02
benchmark, Fable-first order: `esc tab ctrl alt arrows ^C`, then
home/end/pages and `| ~ / -`). Hide is pinned on the right and blurs
xterm, so the row and the OS keyboard open and close together. Sticky
Ctrl/Alt and the home-grown bar (no simple-keyboard) stay. The same
row is mounted on the agent Terminal segment; that attach now carries
`createSticky()` like `ShellTerm`. Chat gets the viewport lift only —
no extra keys.

The previous `innerHeight - vv.height > 120` gate is gone: it is why
the lift did not fire when iOS reported equal heights. Pinch-zoom
(`scale !== 1`) does not rewrite the variables. A hardware keyboard is
detected only with a fine pointer, no inset, and a 300 ms wait; a
coarse pointer with a 0 inset still shows the row. iOS's own undo/Done
pill cannot be replaced from the web; if it still covers the prompt
after a correct `--vv-height`, pad from a measured value on the owner's
phone, do not guess it.

## Amendment 2026-09-06 (evening) — pin only while the IME is up

Owner screenshots after deploy: a black strip under the TUI with the
keyboard closed (`#m-app` was always sized to `visualViewport.height`,
plus safe-area padding below xterm); the prompt stayed behind the extra
keys when the IME opened (xterm's canvas overflowed the pane); scrolling
the row showed TUI ink through the gaps. Safari's undo/Done pill is
system chrome over the web view — a PWA cannot hide it.

`#m-app` is `position: fixed; inset: 0` at rest. `--vv-height` is written
only while the IME covers pixels (inset vs `innerHeight`, or a shrink
against the unfocused visual-viewport baseline). The extra-keys row is
opaque (`--bg-base`) with `z-index` above xterm, and the pane clips
overflow then refits. The TUI draws to the bottom of the pane (no extra
home-indicator padding) so a closed keyboard is not a dead strip.

A sideways drag on the row must not show iOS's overlay scrollbar on the
screen: the row is `touch-action: pan-x`, the terminal screen turns off
`-webkit-overflow-scrolling`, and overlay scrollbars are hidden on the
row, the screen and xterm's viewport.

## Amendment 2026-09-07 — first open does not arm the accessory

Owner screenshots on iOS 26: opening a terminal focused xterm without a
user tap, so the extra-keys row appeared and iOS did not open the
software keyboard; a black strip sat under the TUI (`innerHeight` minus
`visualViewport.height` at rest can exceed the pin threshold because of
`100vh` vs the home indicator).

Attach no longer calls `term.focus()`. Extra keys require a user tap
(`userArmed`: pointer on the pane, or the header keyboard icon). The
header icon does not `preventDefault` on open, so the tap can summon the
IME. `#m-app` pins only when a focused field actually shrank the visual
viewport against the unfocused baseline — a rest-state inset is not the
IME.
