# ADR-0115: Browser surface input — control behind session consent

- **Status**: superseded by ADR-0117 (feature removed 2026-09-10)
- **Date**: 2026-09-10
- **Boundary**: security model (an authenticated client can now drive the
  agent's browser — mouse, keyboard, touch — not only watch) and protocol
  (the proxy forwards a second engine message family). Amends ADR-0114's
  read-only constraint; everything else in ADR-0114 stands.
- **Driver**: `packages/pi-browser-capture` (consent), `internal/server/browser_ws.go` (forwarding).

## Context

ADR-0114 shipped the browser surface watch-only and named its condition for
more: interactive control is the same trust level as terminal attach
(ADR-0085), and it deserved its own consent decision. The engine's stream
protocol already accepts `input_mouse`, `input_keyboard` and `input_touch`
(no wheel — remote scrolling is not in the engine's protocol yet). The
remaining question was who says "this session may be driven".

The repo already has a consent pattern: `/browser-captures` (ADR-0082) is a
branch-persisted session entry written by a pi extension, with a CLI flag and
a slash command, so consent follows the session tree and dies with it.

## Decision

**Consent lives in the session's capture directory as `input.json`; the
proxy forwards input only while it says on; two writers share one format.**

1. `pi-browser-capture` gains `browser-input` (flag + `/browser-input
   on|off|status`), persisted as a `browser-input-consent` branch entry and
   restored on session start/tree — the TUI path (and fork/restore
   semantics), exactly like captures. It mirrors the live state to
   `<sessionFile>.capture/input.json` (`{"on":bool,"ts"}`, atomic write).
2. The browser surface's toggle goes through a daemon route,
   `POST /api/agents/{id}/browser-input {on:bool}` (behind the one auth
   gate), which writes the same mirror. Managed-mode prompts do not run
   slash commands (verified: a `/browser-input on` prompt reached the model
   as plain text), so the composer path cannot be the toggle.
3. The proxy forwards `input_mouse` / `input_keyboard` / `input_touch` only
   while the mirror says on (checked with a short TTL so mouse-move bursts
   do not hammer the disk); otherwise the read-only refusal envelope from
   ADR-0114 is returned.
4. The mirror is keyed to the session file: a fresh or forked session has
   no mirror, so control starts off — the safe default every time the
   session changes.

## Consequences

**Easier:** the human can unstick the agent (CAPTCHA, MFA, login) inside the
surface, closing the loop Browserbase/Devin/Manus all ship. Consent follows
the session (branch-persisted), so a forked or restored session starts
watch-only again — the safe default.

**Harder / accepted costs:**

- Two writers share one mirror file (extension and daemon). Both write
  atomically, same shape; last writer wins. The extension additionally
  branch-persists for fork/restore; the daemon route's consent is scoped to
  the current session file and disappears with it.
- Consent state lives in the session; the daemon's mirror read can be up to
  one TTL stale. Turning control off in the terminal can take up to a second
  to reach a driving client. Accepted: the window is bounded and local.
- No remote scrolling: the engine's protocol has no wheel event yet. Control
  covers click, type and touch; scrolling needs an upstream addition, not a
  proxy hack.
- The keylogger-shaped risk is real: with control on, keystrokes go to the
  agent's browser — the same surface the user already types into when they
  "take over". The chip says so.

**Who breaks if we are wrong:** a consent mirror forged inside the capture
directory requires the session's own uid and dir trust the sidecar already
relies on (ADR-0082); the proxy adds no new writer.

## Alternatives considered

- **Toggle via a prompt (`POST /prompt` with "/browser-input on")**: tested
  live — managed-mode prompts hand the text to the model; slash commands do
  not run. The composer-only path would also need the agent's UI open.
- **Toggle via `POST /api/agents/{id}/command`**: that route is the TUI door
  (ADR-0060) — it stops the managed runtime and types into tmux. Wrong tool.
- **PiCode-level setting (store + feed event)**: puts agent-capability
  consent in the daemon and adds a store mutation; the ADR-0082 split
  (pi owns agent capability, PiCode supervises) keeps it beside the session.
- **Consent prompt per input session (Dialog each time)**: noisier than the
  session-level toggle and repeats what the chip already states.
