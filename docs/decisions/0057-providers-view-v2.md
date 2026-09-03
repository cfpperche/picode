# ADR-0057: Quota, identity and health on the providers roster

- **Status**: accepted
- **Date**: 2026-09-03
- **Extends**: ADR-0031 (live provider usage), ADR-0013 (multi-account vault)

## Context

`#/providers` answered one question: which logins does this machine have.
The 2026-09-03 study
([docs/benchmarks/2026-09-03-providers-view-v2.md](../benchmarks/2026-09-03-providers-view-v2.md))
found the field had moved past that. cc-switch v3.13, claude-swap, CodexBar
and the Antigravity monitors all render each account's window **on the row**;
quota behind a button is the pre-2026 design. The same study found three
independent receipts for the opposite discipline — pi's own statusline
extension omits the segment for providers it cannot measure, CodexBar refuses
a synthetic zero, and Claude-Code-Usage-Monitor labels estimates as estimates.

Three more facts settled the shape:

1. pi v0.84.4 still stores **one credential per provider**
   (`Record<string, Credential>`); native multi-account has not landed, so
   ADR-0013's vault is still the only place several accounts can live.
2. pi ships `pi auth check --provider X --json --no-refresh`, which answers
   whether the agent could use a provider right now. PiCode had no
   validation step at all, and every peer that has one (Cursor, Raycast,
   Vercel, Zapier, Auth0) validates at entry rather than mid-task.
3. Measured on this machine: with only `GROQ_API_KEY` set, pi answers
   `ready/api_key`. PiCode showed that provider as absent while every agent
   it spawns could already use it.

## Decision

**Quota moves to the roster, in three states, from a cache.**
`GET /api/providers/usage` answers from a process cache and never calls a
vendor. Each row is *live* (fetched within the refresh interval), *stale*
(the same number, labelled with its age, with a control to re-check), or a
word saying which kind of nothing it is — `not checked`, `sign in again`,
`no plan windows`, or the vendor's own error. A bar is drawn only from a
percentage a vendor returned. A balance with no ceiling (OpenRouter credits,
Anthropic extra usage) renders as text, because a number with no denominator
cannot honestly be a gauge.

**Only the active slot refreshes on a timer.** `StartUsageRefresh` walks the
active, non-paused account of each meterable provider every 5 minutes,
sequentially. These endpoints are undocumented and rate-limited; polling
every account of every provider is how a vault gets throttled. Everything
else is fetched on demand, by the Usage dialog or the row's own control.

**Identity comes from the vendor, the label stays the user's alias.** The
Anthropic profile call ADR-0031 already made for the plan now also yields the
email; both are written back to the vault row so the next page load names the
account without a network call. Plan ids are normalised to the words the
vendor uses on its pricing page (`default_claude_max_5x` becomes `Max 5x`),
and `billing_type` is dropped: `stripe_subscription` is how they charge, not
what the person has.

**The credential's source is displayable.** A provider pi would answer from
an environment variable is signed in, labelled by the variable, and cannot be
signed out from this page. The env-var table mirrors pi's own, read from the
installed bundle.

**Verify asks pi.** The row's Verify runs `pi auth check --json --no-refresh`.
PiCode neither invents a probe nor spends a token on a test completion.
`--credentials` is never passed, and a provider id that could be read as a
flag is refused.

**Pause is a verb next to Sign out.** A vault row keeps its credential and
leaves play. Pausing the active row promotes another live one; pausing the
only live row is refused and the action is not offered.

**Sign out states its blast radius.** The catalog carries how many agents and
automations name each provider, and the confirm says so.

## Consequences

- Easier: "which account should I use right now" is answerable without
  clicking, and "why did that agent fail" has an answer on the same screen.
- Harder: one more place vendor drift shows up. A broken adapter now
  degrades a visible row instead of an unopened dialog — which is the point,
  but it makes each adapter's error text user-facing copy.
- The daemon now makes provider calls with no one watching. It is bounded:
  active slots only, sequential, every 5 minutes, and it stops at the first
  context cancellation.
- ADR-0031's composer-statusbar rule is untouched: the statusbar still
  invents nothing, and the roster does not either.

## What this does not do

The single active slot (ADR-0013) stands. Per-agent credential pinning and
proactive auto-switch — the two ideas the study found in claude-swap,
OpenRouter BYOK, Cloudflare's alias and oh-my-pi — both move that line and
both are the owner's call. They are open questions in the study, not
decisions here. Nothing in this ADR changes a credential the user did not
click.

## Alternatives considered

- **Fetch every row on page load.** Refused: eight vendor calls per render,
  on endpoints with no documented rate limit.
- **Show a bar from the last known number with no age.** Refused: that is the
  invented-number failure the field's own tools warn about.
- **A named profile bundling provider, model and settings** (Roo Code).
  Refused: ADR-0028 put roles in `packages/pi-roles/` and ADR-0009 put
  per-agent config on the agent. A third store here is a fourth place to look.
- **Enforcing a spend cap.** Refused: PiCode is not in the request path, so
  any cap would be advisory. Vercel documents even a gateway's budget as a
  soft cap. Alerting on a threshold is honest; capping is not.
