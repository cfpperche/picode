# Conversation control — implementation roadmap

- **Date:** 2026-08-27
- **Status:** plan. Track A (composer files) and Track B (MCP manager) shipped.
- **Why:** the ADE still goes mute when the agent needs a human. Architecture
  already names it: extension dialogs (`select` / `confirm` / `input`)
  surface on RPC; **UI mapping is open**. t3code calls this `waiting`.
  Busy Send and drafts are the other two holes in the same surface.

Canonical pi: [RPC](https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/rpc.md)
(`extension_ui_request` / `extension_ui_response`, `steer` / `follow_up`) ·
[architecture RPCBridge](../architecture.md) ·
[benchmarks study](../benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md)
(G1–G3).

## Sequence

| Order | Track | Start when |
|---|---|---|
| 1 | **C1 — waiting** | **shipped** |
| 2 | **C3 — visible queue** | **shipped** |
| 3 | **C2 — draft persistence** | **shipped** |

Do not start the [next-roadmap](#next-roadmap-not-this-track) list, Radius,
OpenRouter PKCE, or llama.cpp installer in this track.

## Refuse (this track)

| Temptation | Why not |
|---|---|
| File tree / code tabs / LSP (z.ai Coding Plan, Cline, Kilo) | Cursor adaptation rule: we are an ADE, not a worse IDE |
| Auto-approve every dialog | architecture: policy undecided; C1 is *visible* waiting |
| Modal wizard for `select` / `confirm` | answer lives in the conversation, like tool pills |
| Second pi process to “queue” | `steer` / `follow_up` already on the wire |
| Cloud-synced drafts | this machine; localStorage / SQLite overlay is enough |
| Replacing the kind chip with a hidden default | prompt / steer / follow_up stay explicit |

## C1 — waiting

Agent cards and the composer today know **stopped / interactive / managed
(+ streaming)**. A blocking `extension_ui_request` is none of those: the
process is alive, the turn is stuck on the human.

Go already has `ManagedAgent.ReplyUI`. The GUI does not map the event.

**Ship:** a first-class **waiting** state. The question is a conversation
card (title + options / confirm / one field). Answer or Cancel sends
`extension_ui_response` with the matching `id`. Empty/blocked: one line
(“The agent is asking something”) + the card. Fire-and-forget
(`notify`, `setStatus`) is toast/status — not waiting.

**Not:** `editor` as a full page; a `<textarea>` in the card is enough.
Auto-approve. TUI interactive mode (the dock already is the UI).

| # | run mode | RPC | action |
|---|---|---|---|
| 1 | managed, no dialog | * | unchanged (idle / streaming) |
| 2 | managed | `select` / `confirm` / `input` / `editor` | state **waiting**; card in chat; agent row says waiting |
| 3 | waiting | user picks / types / confirms | `extension_ui_response`; waiting ends |
| 4 | waiting | user Cancel | `cancelled: true`; waiting ends |
| 5 | waiting | request has `timeout` | pi auto-resolves; GUI dismisses the card |
| 6 | managed | `notify` / `setStatus` / `setWidget` / `setTitle` | no waiting; toast or status only |
| 7 | interactive (tmux) | * | TUI; no RPC dialog UI |
| 8 | stopped | * | no card |

C1 **shipped** (2026-08-27). visual-review: PASS (`chat-waiting.png`,
`chat-ask-cancel.png`).

## C3 — visible queue

`steer` and `follow_up` already exist (composer kind chip, task table,
RPC). Busy Send can still feel dead: no chip in the thread for “this
goes next”.

**Ship:** while streaming or waiting, Send still works. The message
lands in the conversation as a **queued** user block (steer vs follow-up
from the kind chip). Abort/stop of the live turn does not delete the
queue unless we say so (default: keep follow-up, drop in-flight steer
with the turn — name the row in the table and test it).

**Not:** a second agent. Hiding the kind chip.

| # | agent | kind | Send | result |
|---|---|---|---|---|
| 1 | idle / settled | prompt | now | `prompt` |
| 2 | streaming | steer | now | RPC `steer`; queued chip until injected |
| 3 | streaming | follow-up | now | RPC `follow_up`; chip until the turn ends |
| 4 | waiting (C1) | follow-up | now | queued; dialog stays until answered |
| 5 | waiting | prompt with no kind change | now | follow-up (do not error) |
| 6 | stopped | * | now | start, then `prompt` (today) |
| 7 | streaming | empty | * | no-op (today) |

C3 **shipped** (2026-08-27). Busy Send stays enabled (prompt → follow-up).
Follow-up is held in PiCode until the agent is idle (**Edit** / **Remove**).
Abort drops queued **Steer** (already on the wire).

## C2 — draft persistence

Composer `draft` is React state. Reload or agent-tab switch drops it.

**Ship:** persist the open composer (text + kind, not image bytes) per
agent id across reload. Restore on select. Empty draft deletes the
slot.

**Not:** pinning drafts as Pins. Cross-device sync.

| # | event | stored draft | action |
|---|---|---|---|
| 1 | type in composer | * | write-through (debounce ok) |
| 2 | reload, same agent | non-empty | restore |
| 3 | switch agent and back | non-empty | restore that agent |
| 4 | Send succeeds | * | clear slot |
| 5 | empty composer | leftover | delete slot |

C2 **shipped** (2026-08-27). localStorage `picode-drafts` per agent (text + kind).

## Next roadmap (not this track)

Recorded so the next plan does not rediscover them. **Do not start here.**

| Gap | Why later |
|---|---|
| Session rewind / checkpoints (pi JSONL `id` / `parentId`) | Cursor #2; study G7 P2; needs waiting stable |
| Cost / tokens in the session chip | Cursor #8; M4 |
| Reload-safe `#/agent/<id>` | **Track D1** — `docs/design/session-surface-roadmap.md` |
| `@agent` / `@skill` (`@file` already ships) | Cursor #6 remainder. Same as `@file`: tag context into **this** prompt. Not agents talking to each other. |
| Diff per-hunk accept/reject | we review, we do not author; still not an editor |
| Task graph / multi-agent broker UI | paseo; M4. **This** is agents talking to each other. Separate from `@agent`. |
| ACP / extra SDKs | conflicts with ADR-0003 unless pi speaks ACP |
| IDE chrome (explorer, tabs, LSP) | refused |

Handoff **Backlog** is unchanged: llama.cpp installer/router, mobile
parity, `/tree` `navigate_tree` ([pi#8645](https://github.com/earendil-works/pi/issues/8645)).
Parked with A/B: Radius gateway, OpenRouter PKCE, MCP Sign out on
macOS/Windows.

Owner Voice V1 dogfood: done (Chrome Windows mic).

## Where it lives

| Thing | Path |
|---|---|
| This plan | `docs/design/conversation-control-roadmap.md` |
| Predecessor | `docs/design/composer-mcp-roadmap.md` (A+B shipped) |
| RPC runtime | `internal/rpc/runtime.go` (`ReplyUI`, `SendTurn`) |
| Panel / composer | `web/src/desktop/App.jsx`, `web/src/components/Composer.jsx` |
| Agent snapshot | `ManagedAgent.Snapshot` (streaming only today) |
