# ADR-0059: Inbox replies use a transient RPC burst behind the TUI surface

- **Status**: superseded by ADR-0060 (Inbox replies are delivered into the
  running TUI). Was accepted 2026-09-03; it superseded the "consented switch"
  amendment in ADR-0037 and narrowed ADR-0006.
- **Date**: 2026-09-03

## Context

ADR-0037 originally parked an Inbox reply until a managed agent could drain
it. Its 2026-09-02 amendment tried to make a TUI agent reachable by ending the
tmux session, starting the ordinary managed runtime, navigating the operator
to the normal chat surface, and reopening the TUI after the turn. Live dogfood
rejected both the product behavior and its lifecycle:

- opening chat was the opposite of the requested interaction; the operator
  asked to stay on the agent's terminal tab while a temporary state showed the
  reply being received and processed;
- `Settled()` is true before a new RPC process accepts its first prompt, so the
  first flip-back watcher reopened the TUI before the turn began and killed the
  reply;
- `follow_up` acknowledges an early command by placing it in pi's in-memory
  queue, which can disappear while a session is still being resumed;
- killing the whole tmux session detached the terminal surface even though
  only the Pi process needed to yield its single-writer lease.

ADR-0006's important invariant still applies: a Pi TUI and a Pi RPC process
must never write the same append-only session JSONL concurrently. "Background"
therefore cannot mean two live Pi writers. It means an ephemeral process swap
hidden behind one stable terminal tab.

The benchmark adaptation is narrow: t3code's first-class waiting state becomes
a first-class reply-burst state; paseo's persistent PTY container motivates
keeping tmux alive while its child changes; Cursor's 100–200 ms state motion
sets the visual bar. PiCode keeps its own RPC protocol and does not copy their
runtime stacks.

## Decision

A reply to an idle TUI agent runs through a per-agent, generation-scoped
`BurstCoordinator`:

1. The Inbox action captures the exact session file that filed the question,
   reserves one generation, and durably parks the reply as one task.
2. The existing tmux session and browser terminal attachment stay alive. Its
   Pi process is replaced by a small holder process, so the session has no
   writer while the burst runs. A lease marker records daemon and RPC PIDs;
   the holder terminates the leased writer before restoring the TUI if the
   daemon dies or a same-PID binary re-exec releases the marker.
3. One transient `pi --mode rpc --session <exact-file>` process starts outside
   the ordinary managed-chat workflow. After `get_state` proves startup, the
   parked reply is sent as `prompt`, not as an idle `follow_up`.
4. `agent_start`, newly appended user-message bytes, and `agent_settled` are
   the lifecycle truth. RPC acknowledgement alone is never delivery truth.
5. The same agent tab renders a dedicated state — receiving, processing,
   restoring, done, or failed — over the parked terminal. It may stream the answer
   and compact tool activity, but it never renders the ordinary chat composer
   or navigates to the chat surface.
6. Once the turn settles, the RPC process stops, the holder resumes the TUI on
   the same session file, and the dedicated state exits. Restoration readiness
   requires the holder lease to disappear while the pane remains live; it does
   not compare `pane_current_command`, which is `node` for npm's script-backed
   `pi`. The restored TUI reads the complete user/assistant turn. The agent's
   persistent run mode remains interactive; `burst` is transient orchestration
   state, not a mode change.

ADR-0006 remains authoritative on one live Pi process. This ADR narrows its
"mode switching is explicit" consequence: the temporary swap is allowed only
inside the coordinator, leaves tmux and the interactive product mode intact,
and is never exposed as managed chat.

On Linux, the transient child also uses `Pdeathsig: SIGKILL`; the marker is the
holder's cross-restart lease and contains the child PID. Startup releases and
waits for holders **before opening SQLite**, so an exec-style update that keeps
the daemon PID cannot race recovery's final JSONL scan. A marker created before
its holder has no child PID and is removed after a bounded wait; a possibly live
leased PID remains fail-closed and prevents daemon startup rather than allowing
a second writer. Orderly shutdown cancels every coordinator generation and
waits for a terminal phase before closing the store.

Store recovery distinguishes three truths. A queued task never reached Pi and
is failed/reopened. A claimed task with no timestamp-correlated, full-payload
user row is also failed/reopened. A claimed task whose exact session already
contains that durable row is finalized as delivered and is **not** reopened,
even if the daemon died before recording `agent_settled`; replaying it would
create a duplicate turn.

### Decision table

| Conditions | Action | User-visible result |
|---|---|---|
| Item is not an agent question/approval | Use ordinary Inbox action | No burst |
| Agent is stopped or already managed | Use ordinary durable delivery | Existing surface stays put |
| TUI exists, is idle, exact session is safe, no burst active | Reserve generation, park exact task, start burst | Same tab: receiving → processing → TUI |
| TUI reports a turn in flight | Refuse before parking | Item stays open; retry after it stops |
| Another generation is active | Refuse before parking | Item stays open; current burst is untouched |
| A passive extension UI update (status/widget/title/notify/editor text) arrives | Ignore it; the turn continues | Extensions that decorate the session cannot kill the reply |
| A previous holder lease still exists | Refuse before parking | Item stays open until exclusive-writer recovery finishes |
| Session path is absent, outside the agent session roots, or gone | Refuse before parking | Item stays open; open the TUI and retry |
| RPC starts and prompt materializes after the captured baseline | Mark exact task delivered, wait for settle | Stream answer, then restore TUI |
| RPC accepts but no `agent_start`/new user message appears | Retry the exact task, at most three attempts | Receiving state keeps moving; no false delivered |
| Cancel races a newly appended user row | Stop the writer, probe once more, let durable JSONL win | Delivered reply is never reopened for a duplicate |
| All attempts fail before materialization | Mark exact task failed, restore TUI | One-line error + Return to terminal |
| A blocking extension dialog (select/confirm/input/editor) opens | Stop and fail like any interactive need | Item reopens with the reply prefilled; consent stays with the human |
| Holder and bounded direct respawn cannot restore Pi | Mark the terminal unavailable; Return force-replaces the stale tmux session and starts the exact agent TUI | No dead terminal is presented as interactive |
| Explicit Return restart fails | Keep the terminal-unavailable card and its retry action; refuse duplicate restart clicks while one is running | A failed recovery never reveals a dead terminal with no way forward |
| Pane-holder install fails after selecting the exact session | Restore the prior selected-session pointer before surfacing failure | Item reopens; unchanged TUI remains authoritative |
| A new reply races an explicit pane/session mutation | Refuse reservation under the per-agent control guard | Item stays open; the mutation cannot acquire a competing writer |
| User cancels or an explicit process/session takeover begins | `cancelBurstAndWait` restores the holder/TUI while the control guard blocks new bursts, then the requested control continues | User control wins; no automatic chat or overlapping restart |
| Daemon exits/re-execs with a queued or unproved delivering task | Release writer before opening SQLite; fail task and reopen exact item | tmux/browser attachment survives; reply remains prefilled |
| Daemon exits/re-execs after a correlated user row exists | Finalize the task as delivered; do not replay it | Restored TUI owns the interrupted turn without a duplicate prompt |
| Startup cannot disprove a live leased PID | Retain the marker and fail daemon startup closed | No second writer starts against uncertain state |
| Late event from an older generation | Ignore it | Newer burst state cannot be overwritten |

Tests cover every row either directly through the decision function or through
store/runtime/tmux/reducer integration. A row not covered is recorded as FAIL
in `docs/handoff.md`, not implied green.

## Visual evidence

Curated captures cover desktop receiving, processing, ordinary failure, failed
pane restoration, and mobile receiving. Focused post-hardening passes also read
desktop and mobile completion, exit, restarting, and failed-restart retry
captures; clicked Return through cancel and exact-agent force-restart; confirmed
that retry remains actionable after restart failure; and confirmed reduced
motion removes the exit animation. Both viewports passed
`window.__picodeOverlayAudit()`.
See `docs/screenshots/adr-0059-burst-*.png`.

## Consequences

- **Easier:** the interaction now matches the operator's mental model: Inbox
  briefly borrows the terminal tab, never redirects them into a different
  workflow.
- **Easier:** lifecycle has one owner and one correlation key. A stale watcher,
  rapid click, late RPC event, or concurrent Runtime restart cannot restore or
  overlap another generation.
- **Easier:** the exact session comes from the asking extension instead of
  guessing from the workspace's latest file.
- **Harder:** tmux must support safe pane respawn and a crash-safe holder; this
  is more machinery than killing and recreating a session.
- **Harder:** the transient surface needs its own compact event projection. It
  intentionally does not become a second full chat implementation.
- **Accepted cost:** the Pi TUI process restarts under the overlay because Pi
  cannot reload another process's appended session state in place. The tmux
  session, web terminal attachment, tab, session file and product run mode stay
  continuous.
- **If wrong:** disable the burst action and fall back to ADR-0037's durable
  parked reply. Direct `tmux send-keys` remains a separately approved fallback,
  not a hidden branch in this design.

## Alternatives considered

| Alternative | Why it lost |
|---|---|
| Navigate to ordinary chat for one turn | Rejected explicitly in live dogfood; it changes the user's workflow and was the defect this ADR exists to remove. |
| Keep TUI Pi and RPC Pi alive together | Violates ADR-0006's measured single-writer invariant and risks session corruption. |
| Suspend TUI with `SIGSTOP`, then resume it | Avoids concurrent writes, but the resumed process keeps stale in-memory session state and cannot honestly reflect the RPC turn. |
| Kill and recreate the tmux session | Technically simple, but breaks the stable terminal attachment and visibly closes the TUI. Pane respawn preserves the container. |
| Inject the reply with `tmux send-keys` | Avoids the temporary RPC process but is screen/input timing, not a delivery protocol. It remains the explicit fallback if the RPC burst fails dogfood. |
