# pi-inbox

Opt-in inbox tools for [pi](https://github.com/badlogic/pi-mono): the
agent files messages into [PiCode](https://github.com/cfpperche/picode)'s
inbox (ADR-0037) instead of interrupting the human.

- **`notify_human {title, body?, reason?}`** — a non-blocking FYI. The
  human sees it in the Inbox app's feed; the agent keeps going.
- **`ask_human {question, context?}`** — a blocking question. The turn
  **ends** (terminating tool result); the human's reply arrives later as
  a follow-up message through PiCode's durable task queue — the agent
  parks, a stopped agent picks the reply up on its next start.

## How it reaches PiCode

Per tool call the extension re-reads `<PICODE_DATA|~/.picode>/server.json`
(PiCode rewrites it on every bind, so port changes are picked up) and
POSTs to `POST /api/inbox` over loopback. The self-signed/mkcert cert is
accepted for that one localhost request only. Identity comes from
`PICODE_AGENT_ID`, which PiCode sets on every managed and TUI spawn; a
raw terminal `pi` files as "pi (unmanaged)". With no reachable PiCode the
tools return a soft explanatory result — a plain `pi` outside PiCode is
unaffected.

## Install

```sh
pi install -l /path/to/picode/packages/pi-inbox   # this workspace
pi install /path/to/picode/packages/pi-inbox      # this machine
pi -e /path/to/picode/packages/pi-inbox/extensions/inbox.ts  # this run
```

Nothing is installed by default (ADR-0010). MIT licensed — see LICENSE;
the rest of the PiCode repository is licensed separately.

## Tests

```sh
node --test test/*.test.ts
```

Pure logic lives in `src/logic.ts` (no pi imports); `extensions/inbox.ts`
is the pi glue.
