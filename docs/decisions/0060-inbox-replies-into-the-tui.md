# ADR-0060: Inbox replies are delivered into the running TUI

- **Status:** Proposed (supersedes ADR-0059)
- **Date:** 2026-09-04
- **Owner decision:** abandon the transient RPC burst entirely; the reply must
  land directly in the agent's terminal TUI. Fallback delivery types into the
  pane via tmux (tradeoffs accepted); replies that arrive mid-turn are queued
  natively by pi; the historical burst-era worktrees are removed.

## Context

ADR-0059 made an Inbox reply borrow the asking session for one transient
`pi --mode rpc` turn: a holder swapped the TUI child for an RPC writer, the
coordinator proved durable JSONL delivery, and the holder restored the TUI.
The machinery was correct but heavy — pane surgery, writer leases, crash
recovery, fail-closed startup — and the live validation kept finding new
edges (most recently: passive `extension_ui_request` decoration from benign
extensions aborting every attempt). The owner chose simplicity: **the TUI
never stops being the writer.**

The enabling fact is in Pi 0.84's extension contract: an extension running
inside the interactive process can submit a user message through the TUI's
own path — `pi.sendUserMessage(content, { deliverAs, triggerTurn })` — with
native steer/follow-up queueing while a turn is streaming. PiCode already
generates and injects extensions into the interactive agents it spawns
(ADR-0056 pattern), so the TUI can have a receiver without touching the
user's `~/.pi`.

This study's tier model (`docs/benchmarks/2026-09-03-guest-tui-agent-state.md`)
calls this level 2 — TUI plus side channel, the paseo/Conductor shape. The
tmux fallback is the level-3 screen hack, allowed only as a fallback.

## Decision

1. **Receiver extension, always on for spawned TUIs.** Interactive agents
   spawned by PiCode get a generated `-e` extension (`pi-inbox-reply.ts`,
   written under `<dataDir>/intercept/`) alongside any user extensions. It is
   separate from the opt-in terminal-status wrapper. On start (and every
   5 minutes) it posts `POST /api/agents/{id}/tui-hello`; the daemon records
   the hello time in memory. A hello younger than 10 minutes means a live
   receiver exists in the TUI process.
2. **Delivery channel — receiver.** The daemon writes a one-shot reply file
   `<dataDir>/tui-inbox/<agentID>/<nonce>.json` (`nonce`, `sessionPath`,
   `payload`). The receiver consumes it, requires
   `ctx.sessionManager.getSessionFile() == sessionPath`, submits
   `pi.sendUserMessage(payload, { deliverAs: "followUp", triggerTurn: true })`
   (owner decision: queue natively mid-turn, start the turn when idle), posts
   `POST /api/agents/{id}/tui-ack {nonce, ok, reason}`, and deletes the file
   either way. The file is the whole transport: no sockets, no polling of
   `/api` state, nothing to leak into chat.
3. **Delivery channel — tmux fallback.** With no fresh hello (old TUI,
   pre-upgrade daemon), the daemon types the reply into the pane:
   `load-buffer` + `paste-buffer -p` (bracketed paste, so pi's editor inserts
   it wholesale) + `send-keys Enter`. Tradeoffs the owner accepted: the paste
   can land in a draft the operator had open, and the daemon cannot verify
   which session the pane is showing. Delivery proof still gates success.
4. **Durable proof is unchanged.** A reply counts as delivered only when the
   exact session JSONL gains a new `type:"message"` user row whose normalized
   payload equals the reply (`internal/rpc/delivery_verify.go`). The
   receiver's ack accelerates confidence for the extension channel (the TUI
   owns the queued message and renders it); the JSONL row remains the
   reconciliation truth at boot and on timeouts. Mid-turn queueing can delay
   the row until pi processes it, so the fallback channel waits up to 10
   minutes in the background before reopening the item.
5. **One attempt, honest reopen.** No retry loop and no generations: a reply
   is either consumed (ack/row) or the item reopens prefilled
   (`store.EndInboxReply`). One pending reply per agent; terminal/session
   mutations keep the per-agent control guard and refuse or wait on it.
6. **Trivial startup.** Holders, leases, `*.hold` markers, `Pdeathsig`
   writers, and fail-closed startup checks are deleted. At boot, any
   `inbox-tui:` (or legacy `inbox-burst:`) task still queued/delivering is
   reconciled by the JSONL probe: row → delivered; no row → drop the reply
   file, fail the task, reopen the item.
7. **ADR-0059 is superseded, not rewritten.** The burst surface (coordinator,
   holder swap, transient runtime, BurstState feed, web card) is removed.
   Migration 023 (`inbox_items.session_path`), the exact-session rule, the
   prefill-on-failure reopen, and `open?restart=1` for dead panes all stay.

### Decision table

| Conditions | Action | User-visible result |
|---|---|---|
| TUI live with fresh receiver hello; session matches | Write reply file → receiver submits → ack | Reply appears in the TUI (queued if busy); item done |
| TUI live, no fresh hello (legacy) | tmux bracketed paste + Enter; background JSONL wait ≤ 10 min | Reply typed into the pane; item done when the row lands |
| Receiver says session mismatch | Fail the task, delete file, reopen item | Prefilled reply; operator re-checks the terminal |
| No ack and no row inside the window | Fail the task, reopen item | Item back in Active with the reason |
| Boot with a pending reply | JSONL probe: row → delivered; else file dropped, task failed, item reopened | No duplicates, no stuck items |
| Reply already pending for the agent | Refuse before parking | Item stays open; try when the send finishes |
| Terminal/session mutation in flight | Control guard refuses the reply, replies block the mutation | Mutually exclusive, no interleaving |
| Pane dead / agent stopped / managed mode | Ordinary durable delivery path (ADR-0037) | Existing surfaces, no TUI injection |

## Consequences

- **Easier:** no second Pi process, no tmux pane replacement, no writer
  leases, no `KillMode` interaction, no burst card — the operator watches the
  answer happen in the tab they never left.
- **Accepted costs:** the receiver only exists in TUIs spawned after this
  ships (legacy panes take the paste fallback); a paste can land in an open
  draft; `sendUserMessage` delivery depends on the extension host staying
  alive (a crashed TUI is caught by the boot/timeout reconciliation); the
  hello is a capability lease, so a stale daemon cannot mistake a dead TUI
  for a live receiver for more than 10 minutes.
- **Security:** reply files live under PiCode's data dir with `0600`, are
  consumed once, and name the exact session; the receiver posts only to the
  loopback daemon with the install token, like pi-inbox.
