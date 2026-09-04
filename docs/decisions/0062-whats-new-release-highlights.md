# ADR-0062: What’s New release highlights

- Status: accepted
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

## Consequences

- Operators see a focused explanation after a real release without a network
  fetch or a server-wide “seen” flag.
- Every browser may acknowledge independently, which suits local and paired
  mobile clients but can show the same notes on a newly paired device.
- Notes are intentionally authored twice: a concise catalog entry for the UI
  and the complete Keep a Changelog section for the release body. The release
  check prevents a tagged release from silently publishing a generic message.
- The catalog is bounded and plain text. Rich Markdown, remote note fetching,
  analytics, and per-account synchronization remain out of scope until there
  is evidence they are needed.
