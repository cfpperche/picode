# Study: adopt t3code, paseo, and Cursor as PiCode benchmarks

- **Date:** 2026-08-24
- **Sources:** AgentDeck's studies (same owner, same machine —
  `~/agentdeck/docs/benchmarks/`) plus PiCode ADRs 0001–0009. We did
  **not** re-clone t3code/paseo in this session; file-path receipts
  below are from those studies. Verify against HEAD before relying on
  a specific symbol.
- **Scope:** what those three teach PiCode, and what we refuse.

## Why now

PiCode already held Cursor as the aesthetic/product north star
([benchmark-cursor.md](../benchmark-cursor.md)). AgentDeck separately
watches **t3code** and **paseo** as *architecture* receipts. The owner
asked PiCode to consider the same set. Cursor stays the design bar;
t3code/paseo join as architecture + composer-depth bars.

## What each one is (for us)

### Cursor — design / UX / product

Already documented. Strongest reference for: one composer that owns
the controls, named modes, model+thinking picker, inline diffs,
checkpoints, `@` context. Closed source — no file receipts.

PiCode adaptation already in force: agent tabs, dock, per-agent
provider/model/thinking bar, view-only diffs, `Ctrl+K`. We still
reject becoming an editor.

### t3code — harness control surface

Receipts (via AgentDeck study 2026-08-23):

- Normalized runtime events (`packages/contracts/src/providerRuntime.ts`)
  — SDKs + ACP, not CLI stdout scraping.
- Session state includes **`waiting`** (permission/question), not a
  boolean `running`.
- Composer: draft store, prompt stash, `@`-mentions,
  `pendingUserInput` anchored on the composer, URL-routed threads
  (`/_chat/$environmentId/$threadId`).

**Take for PiCode:** `waiting` as a first-class agent mode (we only
have stopped / interactive / managed). Draft persistence. Queueing
already exists in pi RPC (`steer` / `follow_up`) — expose it instead
of failing send. Hash routes we started (`#/providers`, `#/mcps`)
should grow toward reload-safe agent URLs.

**Refuse:** embedding Claude/Codex SDKs. ADR-0003 — we drive
user-installed `pi`. ACP is a future maybe, not a v1 bet.

### paseo — daemon + PTY + Pi

Receipts (same study):

- Local daemon, many clients; E2EE pairing relay instead of Tailscale.
- Agent TUI in a PTY with **provider hooks**
  (`packages/server/src/terminal/agent-hooks/…`).
- Sessions as a **task graph** (dependencies, parallel runs).
- They list **Pi** as a first-class agent.

**Take for PiCode:** PTY+TUI is a sibling of our interactive dock
(ADR-0002/0006), not a rival. Task-graph / multi-agent is M4 broker
territory — keep in mind, do not copy the graph UI yet. Voice and
pairing-relay are out of scope (we stay tailnet + HTTPS, ADR-0007).

**Refuse:** replacing pi RPC with screen-scraping. All three projects
(and us) already rejected that.

## Convergence

| Channel under the TUI | Who |
|---|---|
| Embed agent SDKs / ACP | t3code |
| PTY + provider hooks | paseo |
| CLI JSONL (`pi --mode rpc`) + optional TUI in tmux | PiCode (ADR-0002, 0006) |

We keep our channel. We steal *UX states* (`waiting`, drafts, queue)
and *composer shape* (Cursor / t3code), not their runtimes.

## Gap list (PiCode)

Priority: P0 dogfood · P1 leverage · P2 later.

| # | Gap | Priority | From |
|---|---|---|---|
| G1 | Agent **waiting** state (permission / question) — today only stopped / interactive / managed | P0 — **Track C1** | t3code `waiting` |
| G2 | Composer **draft persistence** across tab/reload | P0 — **Track C2** | t3code `composerDraftStore` |
| G3 | Make queueing visible (steer/follow_up already on the wire; busy should not feel like a dead Send) | P0 — **Track C3** | pi RPC + paseo queue |
| G4 | Reload-safe agent URL (`#/agent/<id>` or similar) | P1 — next roadmap | t3code thread routes |
| G5 | `@file` shipped; `@skill` / `@agent` later (mentions, not agent-to-agent chat) | P1 — next roadmap | Cursor + t3code |
| G6 | Streaming markdown in the transcript | **shipped** | Cursor bar #10 |
| G7 | Checkpoints / rewind via pi session tree | P2 — next roadmap | Cursor; deferred (not an editor) |
| G8 | Task graph / multi-agent | P2 — next roadmap | paseo; M4 broker |
| G9 | ACP / extra SDKs | P2 — next roadmap | t3code; conflicts with ADR-0003 unless pi itself speaks ACP |

Track C plan: [conversation-control-roadmap.md](../design/conversation-control-roadmap.md).

## What we keep (not worth copying)

- Single Go binary + `go:embed` (ADR-0001). Neither t3code nor paseo
  ships that way.
- User-installed `pi` (ADR-0003).
- Tailnet HTTPS (ADR-0007) over a pairing relay.
- Door, not cage: the real Pi TUI stays one Terminal click away.

## Ritual

Substantial UI or lifecycle work cites at least one row from this
watch list in the PR/commit or an ADR. New studies go in this folder,
dated, with receipts.
