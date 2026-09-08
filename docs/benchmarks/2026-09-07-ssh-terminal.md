# Study: reaching the agent terminals over SSH

- **Date:** 2026-09-07
- **Owner request:** a benchmark scan found no ADE we track shipping SSH as a
  terminal transport; the owner asked for the full study — pros, cons, user
  utility — before anything is decided or built.
- **Scope:** reaching the tmux sessions PiCode already owns (managed Pi TUI +
  Agent CLIs) from SSH clients (OpenSSH, Termius, Tailscale SSH, mosh-style
  reconnects). Not: PiCode as an SSH *client* to remote machines (remote
  workspaces), not eval-harness design, not a bundled SSH server (refused
  below).

## What is true in the repo today

- Terminals are tmux sessions (`picode-…`) the server owns (ADR-0002); the
  browser xterm over WebSocket is one client, not the home of the PTY.
  **Anyone with shell access to the host can already `tmux attach` to the
  same sessions.** The SSH door exists de facto wherever the user has SSH;
  PiCode neither helps nor documents it.
- App auth is pairing + token + Host/Origin gate over HTTP (ADR-0049). SSH
  to the host bypasses all of it by design — the host's own identity rules
  apply instead.
- Remote-box deployments are a supported shape (ADR-0050/0051, tailnet;
  identity precedent: `tailscale whois` in the shared gateway).
- Agent CLIs are terminals (ADR-0069); PiCode never types into a CLI TUI
  (ADR-0078); the prompt door stages files in the cwd (ADR-0089). Over SSH
  the user is the one typing — ADR-0078 never applied to the user's hands.
- Copy-out is already the weak side of the browser terminal; OSC 52 is
  ignored over SSH remotes ([2026-08-30](2026-08-30-web-terminal-clipboard.md)).
  An attached TUI inherits that ceiling.
- Raw tmux is a footgun at scale: the 2026-09-06 sweep that killed 29
  sessions was `tmux ls | grep '^picode-'` — exactly what an SSH user is
  tempted to run.

## Benchmarks (live docs, 2026-09-07)

| Source | Fact | Take / refuse |
|---|---|---|
| Coder CLI | `coder ssh <workspace>` "start a shell into a workspace"; `coder config-ssh` emits OpenSSH config so workspaces read as ordinary hosts | The bar: a first-class `ssh` verb + config integration. Refuse building our own server to get it |
| GitHub Codespaces | `gh codespace ssh` — but "the codespace … must have an SSH server pre-installed" (`sshd:1` devcontainer feature); auto key-pair, `--config` OpenSSH integration | Even GitHub wires sshd per environment, not a host-wide daemon. Per-workspace sshd makes each workspace an island; not our shape |
| Ona / Gitpod Classic | "SSH is the basis for connecting to your Gitpod workspace" — VS Code Desktop, JetBrains Gateway, CLI all ride SSH | SSH as *the* transport, not an add-on. Their product is the environment; ours is the supervisor of local PTYs — do not invert |
| VS Code Remote-SSH | Connects to "any remote machine, virtual machine, or container with a running SSH server"; the integrated terminal runs remotely | The client-side world already assumes host sshd. We re-implement nothing |
| Tailscale SSH | Tailnet takes over port 22: WireGuard identity, zero `authorized_keys` management, ACLs + check mode, session recording | **The transport that matches ADR-0050/0051 boxes.** Same trust story as the shared gateway's `tailscale whois` |
| Terminal-Bench (eval) | "An execution harness that connects a language model to our terminal sandbox"; 2.0 runs in Harbor on Modal/Daytona sandboxes | Eval harnesses drive PTYs programmatically in-container; SSH appears nowhere in the path. No benchmark pressure |
| Daytona | Sandboxes are reached "programmatically using the Daytona SDKs, API, and CLI"; no SSH entry in the docs index (inference: SSH is not part of the sandbox contract) | Sandbox platforms expose programmatic I/O, not interactive SSH, by default |
| Herdr ([2026-08-27](2026-08-27-herdr.md)) | PTY runtime built to survive detach, lid-close "and SSH" | In our whole set, SSH appears only as a resilience scenario, never a feature |

## What users would get (utility)

| User story | Without an SSH door | With a documented SSH attach |
|---|---|---|
| Phone supervision (Termius + tailnet) | Browser UI only; needs-you behind WebSocket reconnects | Native SSH client, mosh-grade reconnects, familiar scrollback |
| Restrictive networks | WSS to the box blocked or flaky | SSH egress is often allowed where arbitrary WSS is not |
| Terminal-native users | Browser keybindings, second-class copy/paste | Their own emulator: fonts, macros, split panes on one session |
| Watching an agent work | Single xterm pane in the UI | `tmux attach -r` shares the exact TUI the agent sees, read-only |
| Long-running sessions off-box (ADR-0050) | UI already works from any device | Same, plus the host's own tooling (scp, port forwards) |

Honest limits: SSH attach carries the raw TUI only — no Working/Ready chips,
no checklists (ADR-0081), no Inbox, no attach door (ADR-0089). Every PiCode
affordance is a side channel over RPC that does not travel through a bare
tmux client.

## Pros and cons

**Pros**

1. Zero new server code today: ADR-0002 already puts every terminal in
   tmux, so the feature is documentation (plus, optionally, a "copy attach
   command" affordance later).
2. The transport belongs to the user, with the user's auth: no second
   credential store, nothing new to pair or revoke; ADR-0049 untouched.
3. Tailscale SSH on ADR-0050/0051 boxes adds identity, ACLs and session
   recording for free — the trust story the shared gateway already uses.
4. A real resilience win for phone and off-box users (Herdr's own bar:
   survive SSH).

**Cons**

1. Support surface we do not control: client matrix (OpenSSH, Termius,
   Windows), TERM quirks, tmux version drift on user hosts.
2. Footgun amplifier: raw tmux invites `kill-session`-by-prefix sweeps
   (2026-09-06 incident). Docs must teach exact session names and `-r`.
3. Feature asymmetry confuses: a user who lives in SSH never meets the
   Working chip or the prompt door, then reports "PiCode lost my agent".
4. Clipboard ceiling: OSC 52 is ignored on SSH remotes (2026-08-30);
   expectations must be set in docs.
5. The expensive version — bundling an sshd or SSH gateway — means a new
   dependency (`golang.org/x/crypto/ssh`), a second auth system, and a
   security-model ADR.

## How it lands per deployment mode

| Mode (ADR-0049/0050/0051) | SSH door |
|---|---|
| Loopback, single user | User's own sshd or none; a docs note only |
| Remote box on a tailnet | Tailscale SSH recommended; `tmux attach -t picode-…` |
| Shared box, `picode gateway` (0051) | Each daemon runs as its own Linux user, so tmux sockets do not cross users; needs a docs pass before promised |
| Windows host | Out of scope (uneven sshd; matches the CLI-lifecycle Windows refusal) |

## Recommendation (owner call; no ADR filed)

1. **Document, don't build.** A `www/` page — "Reach your agent terminals
   over SSH" — with exact session names, `tmux attach -r` for supervision,
   Tailscale SSH on tailnet boxes, and the incident-grade warning against
   prefix sweeps.
2. **Refuse** an in-process SSH server, an SSH gateway, and RPC-over-SSH
   for v1. If ever wanted, that is an ADR (security-model boundary) and the
   owner's call.
3. Optional, later, tiny: a terminal-card affordance that copies the
   per-session attach command. A UI decision, not this study.

## Refuse

| Temptation | Why not |
|---|---|
| Ship `sshd` inside picode | Second auth system + new dependency + audit surface; Coder/Codespaces precedent wires per-env sshd, and we do not need even that |
| Make SSH the primary transport (Ona-style) | We supervise local PTYs from any browser; inverting makes terminal-emulator quirks the product |
| An agent-specific SSH gateway | ADR-0069: CLIs are the user's terminals, not ours to proxy |
| Promise clipboard parity over SSH | The 2026-08-30 refuse stands; OSC 52 fails on remotes |

## Open questions

- Shared-box mode (0051): does tmux socket isolation need verification
  before docs promise per-user attach?
- Is `tmux attach -r` safe as the documented supervision default — any pane
  write that misbehaves under read-only clients?
- The flight recorder (ADR-0085) cannot see SSH attaches; accept invisible
  clients as a documented blind spot?
