# ADR-0129: Custom provider definitions in the Providers GUI

- **Status**: accepted (owner approved 2026-09-13)
- **Date**: 2026-09-13
- **Boundary**: persistence — PiCode writes `~/.pi/agent/models.json`, a pi-owned file it never touched before; security model — credential placement rules for the new definitions.

## Context

Pi reads provider definitions (baseUrl, API type, compat flags, models) from
`~/.pi/agent/models.json` and credentials from `auth.json` / env. PiCode's
Providers pane only authenticates pi's built-in catalog (`LoginMethods` +
`pi --list-models`); a gateway that is not on the list — the owner's concrete
case was Cheaper Inference (`https://api.cheaperinference.com/v1`) — could
only be used by hand-editing `models.json`. That contradicts the product
premise (the GUI is the means of work) and the roster's own promise that a
signed-in provider is usable from every agent it spawns.

The field pattern is settled (2026-09-13 study): Cline/Roo put an
OpenAI-compatible slot in a form; OpenCode names providers in config
(`baseURL`, `api`, `models`) — the same shape as pi's `models.json`; Raycast
keeps custom providers in YAML but syncs the model catalog from
`GET /v1/models`. Cursor's single global base-URL override was refused: it
hijacks the built-in `openai` provider and breaks native models.

Options: (a) a PiCode-side store with a translation layer at spawn; (b)
hand-editing forever; (c) PiCode writes pi's own `models.json` by merge.
(a) makes PiCode a second source of truth for a file pi reads directly and
adds a spawn-contract change (ADR-0009) for zero gain. (b) is the bug.

## Decision

PiCode creates and edits **named custom provider definitions** by merging
into `~/.pi/agent/models.json` (0600, atomic replace, merge-by-id: unknown
providers and unknown fields inside a touched provider are preserved).
Credentials never go into `models.json`: the API key is stored exactly like
a native key — `auth.json` active slot via the existing `PUT
/api/providers/{id}`, vault for extras (ADR-0013). pi's own resolution order
(`--api-key` → `auth.json` → env → `models.json`) makes an `auth.json` entry
authoritative, so definitions and credentials stay in pi's native files and
the TUI keeps working unchanged.

A custom id must not collide with a built-in provider id (`LoginMethods` or
pi's listed set); overrides of built-ins stay out of scope for the form.
`GET /api/catalog` may expose a custom provider's baseUrl/api/models/
compat (configuration, not secret) plus a `custom` flag; never key material.
Custom rows behave like any roster row (sign out = credential only,
definition survives; Remove deletes definition + credential, with the
blast radius named). Scope is Pi only (ADR-0103); model traffic is never
proxied through PiCode (ADR-0003).

## Consequences

PiCode now writes a file owned by another tool. The merge must be
conservative (raw JSON preserved for untouched entries) or a hand-edited
`models.json` loses fields — tests pin this. Users who edit the file by
hand keep working: the GUI upserts by id and never deletes what it does not
own. If pi changes `models.json` semantics, both tools drift together
instead of PiCode carrying a private format — that coupling is the cost and
the point. Who breaks if we're wrong: a user whose hand-written
`models.json` gets a field re-ordered or dropped by our merge — guarded by
raw-message preservation tests, not by trust.

## Alternatives considered

- **PiCode-side store + spawn-time translation**: no second source of truth
  in pi's own domain; refused.
- **Environment-variable providers** (`OPENAI_BASE_URL` style): one global
  slot, same hijack problem as Cursor's override; refused.
- **JSON editor in the GUI**: makes the file the UI and keeps every footgun;
  refused — the form is the UI, the file stays the format.

## Amendment (2026-09-14) — the model listing

The form may ask an endpoint what it serves (`POST
/api/providers/custom/models`, `internal/modellist`), so ids and the limits a
gateway publishes stop being hand-copied. The credential rule above still
holds and now has a second half: the key may be *sent* to the host the user
configured for that provider — the address pi itself sends it to on every
request — and it is never returned to the browser, including when the
endpoint's own error message echoes it (redacted before the message leaves
the server). This is a metadata listing: no prompt, completion or model
traffic passes through PiCode (ADR-0003). The address is tried at
`/models` and, only when that 404s, one level down at `/v1/models`; a listing
call is bounded (12s, 2 MiB) and its failures are classified so each one
names the fix.
