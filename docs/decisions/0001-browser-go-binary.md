# ADR-0001: Browser app served by a single Go binary

- **Status**: accepted
- **Date**: 2026-08-23

## Context

PiCode needs a rich UI for agent orchestration across workspaces, runnable
by terminal-averse users on any OS. Options considered:

1. **Desktop app (Tauri)** — native shell (Rust) + webview; free tray/global
   shortcuts; installer ceremony per OS; Rust glue layer required.
2. **Browser app + Go server** — single Go binary embeds the UI
   (`go:embed`), serves localhost; browser is the runtime.
3. **Browser app + Node server** — would allow the Pi SDK (`AgentSession`)
   in-process, but adds a second runtime for everything else the server
   does (PTY supervision, WS fan-out, static serving).

Pi's own integration surface is language-agnostic where it matters:
`pi --mode rpc` speaks JSONL over stdio, and the TUI is embeddable via any
PTY. Cross-platform distribution was a hard requirement; install friction
was a product-killer for the terminal-averse audience.

## Decision

PiCode is a **single Go binary serving the UI to the browser**. Go owns:
HTTP/WS serving, agent process supervision, tmux control, session file
parsing, the broker. UI is embedded via `go:embed`; distribution is one
static binary per OS (Go cross-compile).

## Consequences

- **Easier**: single-runtime deploy; goroutines fit the N-agents × N-sockets
  shape; `go:embed` kills asset pipelines at bootstrap; multi-machine story
  comes almost free later (agents run on the server, browser attaches).
- **Harder**: no native tray/global shortcuts initially (PWA can recover
  some); Node-based Pi SDK is unavailable in-process — we go through RPC,
  which is protocol-stable but less deep than `AgentSession` (custom
  compaction, per-agent ResourceLoader would need a Node sidecar later).
- **If wrong**: a Node sidecar for deep SDK control can be added beside the
  Go binary without rewriting the UI or the RPC paths.

## Alternatives considered

- **Tauri**: rejected — two-runtimes-plus-frontend stack; desktop installer
  friction conflicts with "no terminal fear" positioning; gained desktop
  integrations we don't need for MVP.
- **Node-only server**: rejected — WS fan-out and process supervision are
  stronger in Go; and stdlib HTTP/PTY story avoids the dependency surface.
