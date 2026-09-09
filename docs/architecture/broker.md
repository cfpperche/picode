# Broker

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

A Pi extension (`picode-extension`, TypeScript, installed per workspace)
registers tools `send_message` / `read_inbox` that call PiCode's local HTTP
API. The broker delivers messages as `follow_up` prompts to the target agent.
Agents communicate through their native protocol — no internals hacked.
