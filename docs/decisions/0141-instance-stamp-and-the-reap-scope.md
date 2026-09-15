# ADR-0141: The instance stamp — one PiCode does not reap another's sessions

- **Status**: accepted (narrows ADR-0133's "no read can settle it" clause about
  the reap; its receipt model and its namespace rule otherwise stand)
- **Date**: 2026-09-15
- **Boundary**: security model — which sessions an instance may remove (the
  reap scope), and one new marker in every session environment
  (`PICODE_INSTANCE`) that carries the identity the comparison is made on.

## Context

ADR-0133 built the inspector and named its hard question: *"which of those
sessions may PiCode remove?"* Its own context had already measured the
counter-example — seven live sessions read as "claimed by nobody" that in
fact belonged to a **second PiCode instance on the same machine** — and its
decision accepted the ambiguity in the open: a PiCode-shaped session missing
from the store may belong to another PiCode, the row says so, and the human
decides.

Two things changed the cost of that acceptance.

1. **ADR-0139 merged the reads.** Every instance now reads the default
   server's sessions beside its own socket's, so a scratch instance on
   `:8475` shows the production instance's whole fleet in *Not in PiCode's
   records*. On 2026-09-15 that was seen on a scratch screenshot: twelve of
   production's live terminals, each one click from a **Remove** button that
   this instance has no right to press. The copy hedged ("another PiCode on
   this machine — PiCode cannot tell those apart") and the very next block
   offered the removal. A hedge followed by a doorway is not a safety model.
2. **The identity was already in the session.** Every shell terminal carries
   `PICODE_TERM_URL=https://localhost:8445` (ADR-0056 tier 1) in the session
   environment, written at creation with `-e`. Measured on the same machine,
   2026-09-15: all twelve production sessions carry it, and the port in it
   identifies the instance — nothing else on the machine can hold that port.
   Agents carry `PICODE_AGENT_ID` and no URL.

So "no read can settle it" was true of the *store*, and false of the
session: the session carries its creator's address, and from this version it
can carry its creator's name.

## Decision

**The Manager stamps every session it creates with `PICODE_INSTANCE` = the
data directory its socket lives in — injected in the one funnel
(`NewSessionEnvSize`), after the caller's environment, so a caller cannot
claim another identity. `SessionReceipt` reads it, beside `PICODE_TERM_URL`.
When an unclaimed session's stamp names a different data directory — or, for
a session created before the stamp existed, its URL names a different port on
this machine — the app states *"this belongs to another PiCode instance"*,
names it, and offers **no removal door**; `reap` refuses the same case behind
the receipt, so the rule holds whether or not a human is looking at the
screen. A session stamped by this instance is a leftover in the old sense and
is the only PiCode-shaped session this instance may remove; a session outside
the `picode-` namespace is still never actionable.**

The stamp is derived from the socket path (`filepath.Dir`), so `NewWithSocket`
carries the identity and `New()` — tmux's shared default server, the drain's
read-only legacy manager — carries none and stamps nothing.

The residual ambiguity is named, not hidden: a session with neither stamp nor
URL (an agent session created before this version) keeps ADR-0133's sentence
and its human judgement. The list keeps one call per screen (ADR-0133 rule 1)
and therefore carries no per-row verdict; the detail screen, which already
reads the receipt, is where the answer is.

## Consequences

- A second PiCode on the machine can no longer kill the first one's work
  through its UI. The reverse also holds: production cannot reap a scratch's
  sessions, which is correct — they are someone's live work too.
- The eleven-seconds-earlier failure mode is gone: a session's owner being
  *down* (a restart, a stopped instance) no longer makes its sessions look
  reapable, because the comparison is identity, not liveness. A liveness
  check was rejected for exactly this: it fails in the window where the
  sessions matter most (Alternative B below).
- Cost, accepted: move an instance's data directory, or change its port for a
  session created before the stamp, and that instance's own old leftovers
  look foreign — refused, with the reason on screen. They are still removable
  from tmux itself, which is the honest place for "I know better than the
  tool".
- If the stamp is ever written wrong, the failure is safe in one direction:
  a false "foreign" refuses a removal (nothing is destroyed, the screen says
  who it thinks owns the session); a false "ours" requires a session stamped
  with this exact data directory, which is only reachable by copying a data
  directory onto a second instance.
- Not decided here: a machine-wide view of *other* instances' data
  directories (no registry exists, and ADR-0139's socket listing deliberately
  does not invent one), and grouping foreign sessions in the list itself
  (would need one receipt read per row).

## Alternatives considered

- **Keep the hedge and trust the operator.** Its cost is now measured rather
  than hypothetical: the hedge sat directly above the doorway, and the
  doorway is a one-click destructive action on another instance's live
  terminals. Rejected.
- **Ask liveness instead of identity** (dial the session's URL; if nothing
  answers, allow the reap). Fails in the window that matters: a restarting or
  stopped instance's sessions answer nothing while being exactly the work a
  human will re-attach to. Rejected — identity, not reachability, is the
  question.
- **A per-machine registry of PiCode instances** (a well-known file naming
  every data directory). New persistence, a discovery problem for instances
  that predate it, and it answers a question the sessions already answer.
  Rejected; revisit only if cross-instance *listing* is ever wanted.
- **Stamp `PICODE_TERM_URL` only** (no new marker). The URL is a hook concern
  with a lifecycle of its own; deriving ownership from it means a future
  change to the hook address silently rewrites who may be killed. Rejected;
  the stamp exists so the two concerns stay separable.
