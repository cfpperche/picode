# ADR-0075: Bounded, historical tool captures

- **Status**: accepted (owner approved the revised Browser preview plan)
- **Date**: 2026-09-05
- **Supersedes**: ADR-0057's unrestricted image sources and final-result fallback.

## Context

ADR-0057's renderer accepts any image string, loads external URLs on replay,
ignores capture time, and retains a transient frame when the final result
contains none. Desktop transcript reads can replace newer live tool state;
mobile can append duplicate tool rows. Frames-per-call is not continuous
observation and does not bound total session storage.

The [delivery plan](../plans/browser-preview.md) keeps Pi packages and the
existing RPC/transcript transport, but ships safety and reconciliation before
an emitter or a new panel. The owner approved that order and opt-in emission.

## Decision

The core renders **Last capture**, not a live browser. `details.preview.image`
must be a base64 PNG or JPEG data URI, at most 200 KiB decoded. External,
relative, blob and file URLs, SVG and other formats are unavailable rather
than fetched. Capture dimensions are bounded to 1600 per side and 1,600,000
pixels. Legacy optional metadata remains optional: `title`, display-only
`url`, `ts` (Unix milliseconds), and `source` (producer's plain-text identity).
Missing capture time or source never becomes an invented value. URLs displayed
as captions omit credentials, query and fragment. Caption lengths are bounded.

Older timed updates cannot replace a newer timed frame on the same tool call.
Updates after completion are ignored. The final result alone determines the
persisted capture: absent preview clears the transient one; invalid preview
shows unavailable. No-preview tools remain ordinary rows. Decode failures
have a short fallback and explicit retry. The textual result omits preview
image bytes; the existing viewer remains the only enlargement action.

History reconciliation identifies tools by `toolCallId`, preserving active
and more recent tool states only within the current agent/session context.
Stale requests and old sockets cannot update a different selection. Desktop
and mobile retain separate presentation and share headless capture helpers.

Capture is an opt-in producer responsibility. Enabling it allows pixels to
persist in Pi's session files, their backups and paired-device views. Those
pixels may include secrets: the renderer cannot redact them. No automatic
installation, screenshot polling, stream proxy or takeover is added here.

## Consequences

Opening old history no longer contacts third-party image hosts. Existing
external-image previews show unavailable; emitters must provide bounded local
raster bytes. Inline data remains portable on replay but accumulates per call.

The host validates before web fan-out and transcript presentation as well as
before rendering. This does not change Pi's own persistence or make its raw
RPC input unbounded/safe: emitter limits and slow-consumer acceptance remain
required before declaring the browser integration complete. The existing
RPC scanner and session scanner retain their limits; no second storage system
or generic asset proxy is introduced.

The panel and stream need source/session/connection lifecycle contracts beyond
this capture shape. Those remain separate increments, not implied capabilities.
