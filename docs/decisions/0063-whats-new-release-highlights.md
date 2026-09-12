# ADR-0063: What’s New release highlights

- Status: accepted, product-state gate amended 2026-09-11 (a fresh install auto-opens too)
- Date: 2026-09-04

## Context

PiCode releases need a short, useful explanation inside the product. A link
to the repository changelog is easy to miss, while copying the entire
changelog into a modal is too noisy for a terminal-averse operator. The UI
benchmarks point in the same direction: Cursor leads with benefit-oriented
release highlights, Linear puts the primary narrative first, and Zed keeps a
scannable release summary separate from fixes and breaking changes. See
`docs/benchmark-cursor.md` and
`docs/benchmarks/2026-08-24-adopt-t3code-paseo-cursor.md`.

Release notes also need to work when a PiCode server is on a private network
or temporarily offline. A release must not interrupt a user who is answering
an Inbox item, creating an agent, or recovering a connection.

## Decision

PiCode ships a small, curated release-note catalog in
`web/src/data/whats-new.json`. Each entry is keyed by a numeric semver and
contains a date plus up to nine short, benefit-led highlights. The shared
`WhatsNew` surface renders at most three eligible releases and nine total
highlights, using `ResponsiveDialog` (a centered dialog on desktop and a
bottom sheet on mobile). The footer links to the full GitHub release notes for
readers who need the complete changelog.

The release workflow validates that the tagged semver has a non-empty section
in `CHANGELOG.md` and a matching catalog entry, then publishes that section as
the GitHub release body. The
`/api/version` response includes `release: true` only when the binary was
stamped by that workflow. Source and development builds can still open the
surface manually, but never auto-open it.

The browser stores the last acknowledged semver in
`localStorage` under `picode-whats-new-seen`. A stamped build auto-opens once
per browser after the initial fleet has loaded, only when there is at least
one workspace, agent, or terminal. The UI defers while another modal is open,
the connection is recovering, an agent is waiting, an Inbox badge needs
attention, or a create/share flow is active. Closing the surface acknowledges
the running semver; a manual open can always review the available history.

| Release build | Product state | Another task needs attention | Notes newer than seen | Action |
|---|---|---|---|---|
| no | any | any | any | Do not auto-open; manual entry remains available |
| yes | no | any | any | Wait for a workspace, agent, or terminal |
| yes | yes | yes | any | Defer until the task settles |
| yes | yes | no | no | Stay closed |
| yes | yes | no | yes | Open the release highlights once |

*The product-state row above is superseded — see the amendment below.*

## Amendment 2026-09-11: a fresh install sees it too

**The rule.** Product state is no longer a condition. A stamped build
auto-opens once per browser as soon as the shell has booted and there are
notes newer than the acknowledged semver, whether or not a workspace, an
agent or a terminal exists. The row `yes / no product state / any / any →
Wait for a workspace, agent, or terminal` is struck; every other row stands
unchanged.

| Release build | Another task needs attention | Notes newer than seen | Action |
|---|---|---|---|
| no | any | any | Do not auto-open; manual entry remains available |
| yes | yes | any | Defer until the task settles |
| yes | no | no | Stay closed |
| yes | no | yes | Open the release highlights once |

**The decision.** The owner's, 2026-09-11, after the 0.2.0 verification: the
surface did not open on the freshly installed artifact and read as broken.
The runbook had already been taught to work around it — step 5 told the
verifier to seed a terminal through the instance's own API, verify, then
delete it by exact id — which is the shape of a rule that costs more than it
returns. Step 5 loses that instruction with this amendment.

**The trade it accepts.** A first-time reader is shown what changed in a
product they have not used yet: for them the list is noise, because they
lacked none of it. That is paid for by the many who arrive on an upgrade
path — `picode update`, a new package, a re-installed desktop build — where
a fresh browser profile or a fresh data directory is indistinguishable from a
first run, and where the notes are exactly what the reader came for. The
original row could not tell those two apart; it only ever knew whether the
instance was empty.

**What the old row protected, and what still protects it.** It protected a
first run from being interrupted before the reader had done anything — the
one moment when a modal in front of an empty product is at its most
graceless. What still protects that moment is every other defer condition,
and none of them was touched: another modal open, a recovering connection, a
waiting agent, an Inbox badge asking for attention, an active create or share
flow. The acknowledgement is also unchanged — closing the surface writes
`picode-whats-new-seen`, so a first run is interrupted at most once, and a
manual open remains the way back to the history.

## Consequences

- Operators see a focused explanation after a real release without a network
  fetch or a server-wide “seen” flag.
- Since the 2026-09-11 amendment a brand-new install sees it as well; the cost
  is one dismissible modal in front of a reader with nothing to lose, and the
  gain is that an upgrade on a fresh profile is no longer silently skipped.
- Every browser may acknowledge independently, which suits local and paired
  mobile clients but can show the same notes on a newly paired device.
- Notes are intentionally authored twice: a concise catalog entry for the UI
  and the complete Keep a Changelog section for the release body. The release
  check prevents a tagged release from silently publishing a generic message.
- The catalog is bounded and plain text. Rich Markdown, remote note fetching,
  analytics, and per-account synchronization remain out of scope until there
  is evidence they are needed.
