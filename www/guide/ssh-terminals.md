# Agent terminals over SSH

Every PiCode terminal — a managed Pi agent, a project shell, or an Agent CLI
like Claude Code or Codex — lives in a tmux session on the machine that runs
PiCode. The terminal pane in the browser is one client of that session, not
the session itself. If you can open an SSH connection to that machine, you
can attach to the very same terminal from your own emulator: same TUI, same
scrollback, same running CLI.

## What you need

- SSH access to the host, **as the same Linux user that runs PiCode**. The
  sessions live in that user's default tmux server.
- Any SSH client works — no tmux needed on the device you type on.
- On a tailnet box, [Tailscale SSH](#prefer-tailscale-ssh-on-a-server) is
  the comfortable path.

## Find the session

On the host, `tmux ls` lists every session PiCode owns. PiCode only ever
creates names under one prefix:

| Name | What it is |
|---|---|
| `picode-<id>` | a managed Pi agent's terminal; `<id>` is the agent id in the app's URL (`#/agent/<id>`) |
| `picode-sh-<id>` | a project shell or an Agent CLI terminal |

## Attach

```sh
ssh <host>                        # from any device with an SSH client
tmux attach -t picode-sh-a1b2     # the terminal, as if you never left
tmux attach -r -t picode-a1b2     # read-only: watch an agent work
```

- Leave with the tmux detach key (`Ctrl-b` then `d`). The terminal keeps
  running; the browser pane picks it up unchanged.
- `-r` (read-only) is the polite default for supervision: your keystrokes
  do nothing, so you cannot fight the agent for the prompt. Attach without
  `-r` when you want to share the TTY and type into it yourself.

## Prefer Tailscale SSH on a server

If your PiCode runs on a tailnet box — [On a server](./remote-server) or
[Share one server](./shared-server) — enable Tailscale SSH instead of
managing key files:

```sh
sudo tailscale up --ssh
```

From any approved device on your tailnet, `ssh <box-name>` then attach as
above. Authentication rides your tailnet identity: no `authorized_keys` to
maintain, access controlled by your tailnet ACLs, and optional session
recording. See [Tailscale SSH](https://tailscale.com/kb/1193/tailscale-ssh).

## The one rule: never kill by pattern

Closing a terminal means one exact name, never a sweep:

```sh
tmux kill-session -t picode-sh-a1b2   # one session, on purpose
```

A `tmux ls | grep '^picode-' | xargs -n1 tmux kill-session -t` killed 29
sessions on 2026-09-06, six of them real work. If you want a terminal gone,
close it from the PiCode UI — that kills exactly its own session.

## What the browser pane still owns

SSH attach carries the raw terminal only. These stay browser-only:

| Feature | Where |
|---|---|
| Working / Needs-you chips | the terminal pane, see [CLI activity reporting](./terminal-status) |
| Terminal checklists | the sidebar and pane cards |
| Inbox replies and the attach door | the chat composer |
| Reliable copy out of the TUI | the pane (clipboard over SSH remotes is unreliable) |

A good split: supervise over SSH from a phone or a second machine; do the
copying, file drops and replies from the PiCode UI.

Pairing (see [Security and pairing](./security)) protects the PiCode web
app; it does not govern SSH. Whoever can reach the shell of the user that
runs PiCode can reach its terminals — keep that shell as locked down as you
mean it to be.
