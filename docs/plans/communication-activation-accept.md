# Activation acceptance — 13 September 2026

Scratch `localhost:8473`, six real Agent CLI TUIs, isolated `PICODE_DATA`, vendor homes
symlinked. Evidence: `var/qa/activation-accept/` (uncommitted). No production
terminals were used. `qa-scratch stop` removed the six sessions.

Activate now is one explicit, non-retrying native pointer
(`peerActivationPointer`). It requires an enabled participant, a recorded
session key, an active connection, phase `waiting-conversation`, and an idle
empty composer.

## Decision table (fresh launch)

| CLI | Model shown | Live | Identity | Session | Phase after enable | Activate now | Turn |
|---|---|---|---|---|---|---|---|
| Pi | xai grok-4.6 · high | idle after Trust | confirmed | yes | connected | 409 already connected | $0.000 |
| Claude Code | Opus 5 Max | idle empty | confirmed | yes | waiting (first message) | 409 already connecting | none |
| Codex | gpt-5.6-luna low | open welcome | unobserved | no | waiting-conversation | 409 identify first | none |
| Grok | Grok 4.6 xhigh · [stable] | idle welcome | confirmed | yes | connected | 409 already connected | none |
| Hermes | glm-5.3-flash | open, auto draft | unobserved | no | waiting-conversation | 409 identify first | none |
| OpenCode | Glm 5.3 Flash Cloudflare | open welcome | unobserved | no | waiting-conversation | 409 identify first | none |

The Messages UI showed **no** “Activate now” button (`activate: false` in the
DOM). Overlay was not the subject of this run.

## What this means

- Grok and Pi connected without Activate now and without a billed model turn
  (Pi only needed the Trust folder dialog).
- Claude has a session from SessionStart but waits for a saved first
  conversation; Activate now is refused as already connecting.
- Codex, Hermes and OpenCode still have no session key on a fresh welcome, so
  Activate now cannot mint a connection or send a pointer. Hermes also had a
  curator-filled draft, which would block the empty-composer gate even after
  identity existed.
- **No Activate now request consumed a model turn.** Remote mode was not
  exercised (setup remains the local address).

Widening Activate now to cold-start (no `sessionKey`) would be a new product
cut, not this measurement. The current gate matches ADR-0110: unknown
identities never mint a capability.
