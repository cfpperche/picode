# Notices — the in-app announcement layer

> Part of [PiCode's architecture](../architecture.md) (ADR-0105: one file per subsystem). Edit here; the index only links.

Channels a viewer can mute: `finished`, `needsYou`, and since ADR-0100
`reminder` (Preferences → Notifications, "When a pin reminder is due");
feedback for the person's own action has no channel and is never muted.
`notify()` normalizes what it is handed, so a caller may pass the bare
shape. A notice may carry `onClose`: what its X means beyond hiding the
card (a reminder's X closes its Inbox item).

`web/shared/domain/notice.js` is the model every toast goes through:
`{ level, channel, actor, status, title, body, meta[], actions[], key,
target }`. It is a pure module (ADR-0072 keeps React out of `web/shared`),
so both applications share the model and its policies while each owns the
card — `web/desktop/src/components/Notice.jsx` draws a 300px card beside
the inspector rail, `web/mobile/.../Notice.jsx` a full-width one above the
tab bar. `lib/toast.js` is the single door in both apps: `notify(notice)`
for the model, `toast(text, kind)` unchanged for the one-line call sites,
`dismissNotice(key)` to withdraw one.

The policies, adapted from Superset's notification manager
([study](../benchmarks/2026-09-07-superset-notifications.md)):

| Policy | Rule |
|---|---|
| Muting | A notice's `channel` (`finished`, `needsYou`) names the preference that can silence it. Feedback for something the user just did has no channel and cannot be muted |
| Lifetime | `ok`/`info` use the user's Duration preference; a notice with an action gets at least 8s; `error`/`warn` get 3× the preference (12–30s) rather than `Infinity`, because sonner queues everything past `visibleToasts` and an unbounded class would wall the screen off; `busy` and a standing needs-you wait for their own outcome |
| Suppression | A notice whose `target` is the focused surface (`location.hash` + `document.hasFocus()`) is dropped, and a standing one is withdrawn when the user arrives there (`asksOnSurface`). Only an `error` is exempt — it reports what the user just did, not what is on screen |
| Identity | `key` becomes sonner's toast id, so a second notice from the same source replaces the first instead of stacking (`agent:<id>`, `ask:<id>:<dialog>`) |

**A settled turn** builds `agentFinishNotice` from data the browser already
holds: `turnDurationMs` for "worked for 7s", the `change` on each edit/write
tool item for the file and line counts, and the turn's last assistant
sentence. Desktop composes it in `App.jsx` at `agent_settled`; mobile's
reducer emits a `finished` effect and `useAgentSocket` composes it. Both
see only the agent whose socket is open.

**A waiting agent** goes the other way, and covers the whole fleet: the
runtime's every dialog edge rides `agent.state` (ADR-0048, published in
`cmd/picode/main.go`), `applyFleet` patches `waiting`/`dialog` onto the
agent, and `needsYou` shapes the queue both shells already render.
`needsYouPlan` diffs that queue against what has been announced —
arrivals become sticky cards, answers withdraw them — and the first pass
over a *read* fleet only records, so a reload never toasts a backlog the
badge already shows. Both shells gate that pass on having read the fleet
once (`fleetLoaded`, mobile's `loaded`): seeding from the empty first
render would make the first real answer look like an arrival. Mobile keeps quiet on the Now screen, which *is* the
queue.

The toast layer persists nothing. The Inbox (ADR-0037) owns the durable
copy and `internal/push/notifier.go` the off-device one.
