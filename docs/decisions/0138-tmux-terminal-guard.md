# ADR-0138: tmux-terminal-guard

- **Status**: accepted (owner, 2026-09-15 — "aprovado pode tocar", with the
  recommendations presented alongside the spec)
- **Date**: 2026-09-15
- **Boundary**: process — what processes inside PiCode terminals may do to
  the shared tmux server; extends the ADR-0056 intercept (session-scoped
  PATH) with a fourth wrapper class that is a policy gate, not a launcher.

## Context

Three measured incidents, one class: an agent inside a PiCode terminal ran a
tmux command whose blast radius was the whole server.

- 2026-09-06: a prefix sweep (`kill-session` by pattern) killed 29 sessions
  (recorded in `docs/handoff/open/terminal.md` and AGENTS.md's pkill rule).
- 2026-09-14: an agent ran `tmux kill-server`; every session on the socket
  died, including PiCode's own terminals and the operator's shells
  (screenshot in the session record; the agent apologized and self-restricted
  — after the damage).
- The same document already records the prompt-level guard: "never kill by
  prefix … exact names from a fixture's API only". It leaked twice.

Prompt rules are trust; this repo's own philosophy is mechanism ("enforced
by git, not by trust", AGENTS.md §5). tmux offers no command-level
permission: `server-access` (tmux 3.x) is a user-attach ACL, and
`command-alias` only shadows names inside tmux's command mode, not shell
invocations. The chokepoint that exists is the shell's PATH — and ADR-0056
already puts a PiCode-owned directory first on the PATH of every session the
daemon creates (`new-session -e PATH=…`, re-asserted by the session rcfile).

Prior art checked (2026-09-15): `libtmux`/`gomux` are programmatic wrappers,
`tmuxinator`/`tmuxp` are declarative session managers, `byobu` is a
user-facing front — none is a destructive-command guard; Claude Code's
allow/ask/deny permission rules and PreToolUse hooks are the tool-boundary
guard pattern, applied here at the shell boundary instead.

The failure mode behind all three incidents was measured during this
branch's own testing, and it is worth recording before the decision:
**`$TMUX` outranks `TMUX_TMPDIR`.** A tmux client started inside an
existing session talks to the server named in `$TMUX` no matter what
`TMUX_TMPDIR` says — so "isolated" scratch work (the 2026-09-14 incident:
"I ran `tmux kill-server` thinking `TMUX_TMPDIR` protected [production]")
runs against production. The same trap hit this branch's integration
test, which ran `kill-server` on the live server twice before the fixture
scrubbed `$TMUX` and asserted `#{socket_path}` — the guard's own test had
reproduced the incident it exists to prevent.

## Decision

PiCode installs a `tmux` wrapper into `<dataDir>/bin` (the ADR-0056
intercept directory), **on by default** for managed terminals. Every `tmux`
invocation typed inside a managed session passes the policy gate:

| Command | Condition | Action |
|---|---|---|
| `kill-server`, `kill-window`, `kill-pane` | any | refuse, actionable copy |
| `kill-session -t X` | X is a glob/regex/pattern | refuse ("exact names only") |
| `kill-session -t X` | X carries our terminal's marker (`PICODE_TERM_ID` read from the session environment) | allow |
| `kill-session -t X` | X is a `picode-*` session owned by another terminal, or a session without our marker (the user's own) | refuse |
| `kill-session -a` (all but target) | any | refuse (mass kill by another name) |
| `send-keys …` | payload contains `kill-server`/`pkill`/`killall` | refuse (closes the type-into-another-pane bypass) |
| `new-session` | no `PICODE_TERM_ID=` in the arguments, simple `tmux new-session …` form | add `-e PICODE_TERM_ID=<this terminal>` — transparent ownership stamping, measured: a plain child session does **not** inherit the marker, so without the stamp an agent could never clean up its own scratch sessions |
| anything else | — | passthrough, unmodified |
| outside managed sessions | wrapper not on PATH | unaffected; if invoked directly, passthrough |

The marker is the receipt — the same rule the tmux read model already enforces
("the name is a hint; the marker is the receipt", `internal/tmux/server.go`).
Each refusal appends one line to `<dataDir>/tmux-guard.log` and prints the
reason on the pane so the agent self-corrects in the same turn. Utilities
(`tmux ls`, `tmux mine`) give agents a safe spelling for the operations they
actually need, so the guard removes the dangerous path without removing the
capability. The wrapper is POSIX sh, installed and toggled through the
existing intercept mechanism (`enabled.json` key `tmux-guard`); the daemon's
own tmux paths keep going through `internal/tmux`, which enforces exact names
and markers in Go.

Honest limit, accepted: a PATH wrapper is a guardrail, not a security
boundary — `/usr/bin/tmux kill-server` typed verbatim still bypasses it. It
converts the observed accident class (plain `tmux …` from an agent that
believes the session is its own) into a refused command with feedback.

## Consequences

Easier: agents can investigate tmux state freely (`ls`, `capture-pane`,
`send-keys` for their own panes) without one bad inference wiping the
operator's fleet; refusals are legible and logged; the policy is one table,
testable end to end against a real tmux fixture.

Harder / accepted cost: a legitimate whole-server restart from inside a
managed terminal now needs the operator (or the daemon, which is unaffected);
a wrapper is one more shell script in the intercept set to keep honest; the
send-keys payload check can false-positive on a literal string that merely
mentions `kill-server` — acceptable, the refusal copy says why.

Who breaks if we're wrong: an agent that *must* kill the server (none known)
is blocked until the operator does it; a user who disables the guard in the
UI gets today's behavior back, deliberately.

Two rules this branch paid for and the tests enforce:

- **Isolation is asserted, never assumed.** A test that runs a destructive
tmux verb under `TMUX_TMPDIR` must scrub `TMUX`/`TMUX_PANE` from the child
environment and read `#{socket_path}` back before acting; the guard's
integration test skips if the socket is not under its temp directory.
- **The guard resolves its own path in pure shell.** The shared
`wrapperFindReal` shells out to `dirname(1)`; under a minimal PATH the guard
found itself in its own bin dir and exec'd in an endless loop
(2026-09-15). The guard uses `${0%/*}` and every test exec carries a
deadline, so a loop fails the suite instead of orphaning a process.

## Alternatives considered

- **Prompt rules only** (status quo): failed twice under measurement; trust
  is the thing this repo refuses to rely on for enforcement.
- **tmux `command-alias` shadowing** in a tmux.conf: only affects tmux's own
  command mode; `tmux kill-server` from a shell never touches it.
- **Dedicated socket (`tmux -L picode`) now**: the stronger mechanical bound
  (a raw kill of the default server cannot reach PiCode sessions), but it
  migrates live sessions, changes the user's manual `tmux attach` flow, and
  touches session discovery — kept as the named next step if incidents
  continue, not this slice.
- **Full pty ownership / reimplementing session control**: out of scope; the
  daemon already owns the sessions it creates via markers.
