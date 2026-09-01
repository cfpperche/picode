# ADR-0047: Web Push over VAPID, in the standard library, presence-aware

- **Status**: accepted
- **Date**: 2026-09-01

## Context

The mobile shell (ADR-0044) can answer an agent's question from a phone —
but only if the user happens to open it. Every product in the benchmark
study (`docs/benchmarks/2026-09-01-mobile-agent-supervision.md`) calls the
phone instead: Codex pushes on turn completion and "needs input", Claude
Code's Remote Control pushes "when actions required" and skips the push
while the user is at the machine, AgentWatch and Tactic Remote are built
around the alert. Phase 2 of the approved mobile plan is that alert.

Two ways to reach a phone from a self-hosted Go binary: a third-party
relay (Firebase/OneSignal SDKs, an account, a dependency, the message
leaves the machine in the clear), or the browser's own Web Push — the
phone subscribes through its vendor's push service, the server posts an
encrypted blob there, the service worker shows it. Web Push needs three
RFCs (8030 transport, 8291 encryption, 8292 VAPID) and every library that
implements them is a few hundred lines over primitives Go 1.26 ships
(`crypto/ecdh`, `crypto/hkdf`, AES-GCM, ECDSA).

## Decision

`internal/push` implements Web Push with the standard library only:

- **VAPID identity** (`vapid.go`): one P-256 key pair per install in
  `<DataDir>/vapid.json` (0600). `Authorization: vapid t=<ES256 JWT>,
  k=<public key>`; `aud` is the push service origin, `exp` 12h.
- **Encryption** (`encrypt.go`): RFC 8291 `aes128gcm`, one record. The
  test pins the RFC's Appendix A vector byte for byte, plus a round trip
  through a receiver implementation.
- **Sender** (`send.go`): one POST per subscription with `TTL`, `Urgency`
  and `Topic`; 404/410 is `ErrGone` and the subscription is dropped.
- **Notifier** (`notifier.go`): the decision table —

  | condition | action |
  |---|---|
  | a browser on the host machine pinged in the last 45 s | skip everything |
  | inbox item `result`, device pref *finished* | push, tag `result:<agent>` |
  | inbox item blocking (approval/question), pref *actions* | push, tag `inbox:<id>` |
  | inbox item non-blocking | skip — the badge is enough |
  | managed agent raised a dialog with **no socket open**, pref *actions* | push "<agent> needs you", tag `ask:<agent>` |

  The presence rule is Claude Code's: the phone is for when you are away.
  `hub.Len() == 0` on the live-dialog path means neither the desktop nor
  the phone is watching that agent right now.

**Hooks, not imports.** `store.Store.OnInboxCreated` fires from
`CreateInboxItem` and from a superseded result in `FileAgentResult`;
`rpc.Runtime.OnWaiting` fires from `noteUIRequest`. `cmd/picode/main.go`
wires both to the notifier and shares one `presence.Registry` between
the notifier and the server. Neither `store` nor `rpc` knows about push.

**Subscriptions** live in the store (migration 018): endpoint (unique),
the two browser keys, device id, user agent, per-device prefs
`{actions, finished}`, last success, failure count. Endpoints:
`GET /api/push/vapid`, `POST|PATCH|DELETE /api/push/subscriptions`,
`POST /api/push/test`. `Deps.Push == nil` answers 503.

**Client.** `sw.js` gains `push` (show, `tag` collapses repeats) and
`notificationclick` (post `{navigate, hash}` to an open window, else open
`/?mobile=1#<hash>`); `main.jsx` listens and sets the hash — a tap lands
on the agent or the inbox item. `lib/push.js` names every reason a
subscription cannot happen (no HTTPS, unsupported browser, iPhone not on
the Home Screen, permission denied) instead of failing silently.
`PushPrefs.jsx` — Enable / two switches / Send test / Disable — is mounted
in mobile More → Notifications and desktop Preferences → Notifications.
The service worker now also registers on `localhost` (a secure context),
so the flow can be exercised without a certificate.

## Consequences

- **Easier**: the phone is called when it matters and only then; nothing
  new to install, no account, the message body is encrypted end to end
  to the browser and never readable by the push service.
- **Harder**: a fourth crypto surface to keep right — mitigated by the
  RFC vector test; and a subscription is per browser install, so a
  reinstalled PWA must enable again (the 410 path cleans the old row).
- **Accepted cost**: iOS requires the PWA on the Home Screen (iOS 16.4+)
  and a permission prompt from a tap; the UI says so. The presence rule
  keys on any host-machine browser being open, not on the user actually
  looking — the same approximation Remote Control makes with its presence
  file. A desktop that is open but unattended will not push.
- **If wrong**: `Deps.Push = nil` turns the whole thing off; the hooks
  are nil-checked.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Firebase / OneSignal / ntfy relay | A vendor account and SDK for a one-binary product (ADR-0003 posture), and the message leaves the machine unencrypted. |
| `github.com/SherClockHolmes/webpush-go` | Solid, but the whole thing is ~300 lines over stdlib primitives; AGENTS.md rule 3 asks every dependency to justify itself and this one could not. |
| Native app with APNs / Live Activity (AgentWatch) | Out of scope for a PWA; Web Push covers the alert on both platforms. |
| Push on every inbox item | Non-blocking `fyi` would make the phone noisy; the badge already carries it. |
| No presence rule | The desktop user gets a phone buzz for a dialog they are already looking at. |
