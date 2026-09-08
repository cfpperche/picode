# Study: agent notifications and the toast payload (Superset)

- **Date:** 2026-09-07
- **Sources:** [superset-sh/superset](https://github.com/superset-sh/superset)
  (open repo — file-path receipts, read through the GitHub API on
  2026-09-07), [docs.superset.sh/agent-status](https://docs.superset.sh/agent-status),
  and the owner's screenshot of [superset.sh](https://superset.sh) (2026-09-07).
  Nothing was cloned. Inferences are marked *inf.*
- **In-house counterpart:** `web/desktop/src/lib/toast.js` +
  `web/desktop/src/components/Toasts.jsx` (byte-identical twins under
  `web/mobile/src/`), `web/desktop/src/styles/app.css:3067-3122`,
  `internal/push/notifier.go`.
- **Scope:** what a notification carries, when it must not fire, and how
  one is addressed and de-duplicated. What PiCode adopts; what it refuses.

## Why now

The owner sent a Superset screenshot with one element circled by context:
a floating card, bottom-right of the center pane, reading

> ✳ **claude**  finished · worked for 7s
> Pushed and opened a draft PR.
> ⑂ 2 files **+46** **−1**                    [ Preview ]

and asked for our toasts to be refactored towards it.

**Honesty first:** that exact card is *not* a shipped Superset component.
It is a static marketing mockup —
`apps/marketing/src/app/[lang]/components/HeroSection/components/AppMockup/components/MainPanel/MainPanel.tsx`,
carrying the author's own comment: *"Elevated agent card: the story moment,
floating like Linear's agent panel"*. The values (`7s`, `2 files`, `+46`,
`−1`) are literals in JSX. What Superset actually ships is an **OS**
notification (`Notification` from Electron main) plus derived status dots;
its in-app toaster is a plain shadcn/sonner wrapper.

That does not weaken the request. It sharpens it: the card is a *design
target* nobody has shipped, and the shipped Superset code answers the
harder half — the **model** behind a notification. This study takes the
anatomy from the mockup and the semantics from the product.

## Where PiCode stands today

| Fact | Value |
|---|---|
| Entry point | `toast(message, kind)` — one string, three levels (`web/desktop/src/lib/toast.js:5`) |
| Call sites | **317** across **48** files (143 `toastError`, 89 `toast.ok`, 56 `toast.error`, 29 `toast.info`) |
| Desktop / mobile split | `lib/toast.js` and `lib/toastPrefs.js` are **byte-identical** in both apps (`diff` returns 0) |
| Actions on a toast | none — sonner's `action` option is never passed |
| Identity on a toast | none — no actor, no icon; sonner's `icons` prop is never set |
| De-duplication | none — no `id` is ever passed, so N events stack N toasts against `visibleToasts` (default 3) |
| Suppression | none in the browser. `internal/push/notifier.go` suppresses the *phone* push when a host browser is online; the browser toast has no per-surface rule |
| Agent finished | **produces no toast at all** — `case "agent_settled"` returns zero effects (`web/desktop/src/lib/agentEvents.js:69`) |
| User preferences | 7 knobs, all about the box: position, duration, visibleToasts, expand, closeButton, closePlace, richColors (`lib/toastPrefs.js:12`) |
| Turn duration | already computed — `turnDurationMs` / `fmtWorked` → `"Worked for 7s"` (`web/shared/domain/turns.js:68,87`) |
| Lines changed in a turn | already computed **client-side** — `fileChangeFromTool` returns `{path, add, del}` and rides every tool item (`web/shared/domain/diff.js:4`, `agentEvents.js:83`) |
| Agent/CLI glyph | already available — `terminalCliFaviconUrls`, `terminalCliMark` (`web/shared/domain/terminalCli.js:34`) |

The inversion is the finding: **we have seven settings for where the box
sits and zero vocabulary for what it says.** Every fact the screenshotted
card displays — actor, worked-for, file count, `+`/`−` — is already in the
browser's hands at `agent_settled` time, and is thrown away.

## What Superset does

1. **A four-word lifecycle, normalized across vendors.**
   `AgentLifecycleEvent.eventType` is exactly `Start | Stop |
   PermissionRequest | PendingQuestion`
   (`apps/desktop/src/shared/notification-types.ts`). `mapEventType`
   folds ~25 vendor hook names (`SessionStart`, `PostToolUse`,
   `agent-turn-complete`, `exec_approval_request`, `task_complete`, …)
   onto three of them and returns `null` for anything unknown
   (`apps/desktop/src/main/lib/notifications/map-event-type.ts`).
2. **A notification is addressed, not just displayed.** It carries
   `NotificationIds { paneId, tabId, workspaceId, sessionId, terminalId }`;
   the click handler resolves that into a focus target
   (`renderer/stores/tabs/utils/resolve-notification-target.ts`).
3. **Never on start; never for what you are already looking at.**
   `handleAgentLifecycle` returns immediately for `Start`, then calls
   `shouldSuppressForVisiblePane` — if the window is focused *and* the
   pane that would be announced is the visible one, nothing fires
   (`main/lib/notifications/notification-manager.ts`).
4. **One live notification per source.** The tracking key is
   `sessionId ?? paneId`; registering a new one `close()`s the previous
   entry under the same key, and a 5-minute sweep expires anything older
   than 10 minutes. A chatty agent cannot bury the screen.
5. **Sound is a separate channel.** The notification is created with
   `silent: true` and `playSound()` is called beside it, so audio is one
   preference and the banner is another (docs: *"configure sounds in
   Settings"*).
6. **Status is derived, not stored.** `renderer/stores/v2-notifications/store.ts`
   persists only facts *about the user* — `manualUnread` and a monotonic
   `terminalSeenAt` — with the comment that agent statuses are derived
   from host bindings because carrying them forward "would resurrect the
   stale-dot bug".
7. **The toaster itself is thin.** `packages/ui/src/components/ui/sonner.tsx`
   sets per-level lucide icons, `userSelect: text`, `maxHeight: 80dvh`
   and a scrollable description. No position/close-placement settings.
8. **Rich toasts are `toast.custom()`, and rare.** The one in the repo —
   `apps/desktop/src/renderer/components/StarNagToast/StarNagToast.tsx` —
   is a 356px card: title row with a close ✕, one muted description line,
   one action button, `duration: 30_000`, and an `onAutoClose` that records
   the ignore. Guarded to fire at most once per session.
9. **The mockup's anatomy** (`MainPanel.tsx`), which is the part the owner
   is asking for: a 290px card in three zones —
   *identity* (12px agent icon · `claude` at font-medium · `finished ·
   worked for 7s` in `text-muted-foreground/55` at 10px),
   *body* (11px, muted, one sentence),
   *footer* behind a `border-t`: metadata left in `font-mono
   tabular-nums` (a pull-request glyph, `2 files`, `+46` in emerald,
   `−1` in rose) and a pill-shaped ghost button `Preview` right.
   Elevation is `0 1px 1px rgba(0,0,0,.4), 0 16px 50px -12px rgba(0,0,0,.7)`
   plus a `inset 0 1px 0 rgba(255,255,255,.06)` top highlight.

## What PiCode adopts

- **A notice model in `web/shared/domain/`, not a component.** ADR-0072
  forbids React in `web/shared`, and that is the right cut here: the model
  (`{ level, actor, status, title, body, meta[], actions[], key, target }`)
  and its policies (duration by level, suppression, dedup key) are pure
  functions both apps import; desktop and mobile each own the JSX. The
  phone card is not a shrunken desktop card.
- **Superset's three-zone anatomy, in our tokens.** Identity row / body /
  bordered footer with metadata left and at most two actions right. Colours
  come from `web/shared/tokens/theme.css` (`--ok` for `+`, `--danger` for
  `−`, `--text-secondary` for the status phrase) — no second palette, and
  no emerald/rose literals.
- **Addressed notices.** Every notice may carry a route (`#/a/<id>`,
  `#/changes/<id>`); the card is clickable and the action buttons are
  explicit. Our hash router already makes this free.
- **Suppress what is on screen.** Superset's pane rule, at our granularity:
  a finished-notice for the agent whose conversation is the visible tab,
  in a focused window, does not toast. Errors are never suppressed.
- **One live notice per source.** `key: "agent:<id>"` passed as sonner's
  `id`, so a second finish replaces the first instead of stacking.
- **Level decides persistence, not just colour.** Errors and needs-you stay
  until dismissed; `ok` keeps the user's duration. A 4-second error the user
  was not looking at is a lost error.
- **The lifecycle vocabulary as our own three words.** We do not need
  Superset's 25-name normalization table — ADR-0044 already gives us one
  clean event stream — but we do adopt the shape: `finished` /
  `needs you` / `failed` as the only agent-lifecycle notice kinds, so the
  browser toast, the Inbox row (ADR-0037) and the phone push
  (`internal/push/notifier.go`) finally speak the same three words.

## What PiCode refuses

- **A second notification store.** The Inbox (ADR-0037) is the durable
  mailbox and `internal/push/notifier.go` already decides the phone. The
  toast layer stays ephemeral and derives from the same events; it does not
  persist a parallel history. (Superset reached the same conclusion from
  the other side — its store keeps only what the *user* did.)
- **Rich cards for every toast.** 317 call sites are "Saved.", "Copied.",
  "Up to 4 images." Those keep one line and gain only a level glyph. The
  three-zone card is for actor-attributed events: an agent finished, an
  automation ran, a job failed.
- **Emerald/rose, `text-muted-foreground/55`, and the rest of the mockup's
  literals.** We take the anatomy, not the Tailwind opacities.
- **More than two actions on a card.** Same budget as the workspace card
  header ([2026-09-07](2026-09-07-workspace-card-toolbar.md)).
- **A sound channel in this change.** Superset ships one; we have no audio
  anywhere in the app and adding it is a separate decision with its own
  preference, not a rider on a visual refactor.

## Open question for the owner

The seven layout preferences (`closePlace` alone owns three CSS blocks,
`app.css:3095-3122`) buy very little once the card carries its own close
affordance, and they are the reason the toast has stayed a box instead of
becoming a card. Retiring `closePlace` and `richColors` in favour of two
content switches — *announce finished* and *announce needs-you*, mirroring
`PushPrefs` — is the change that makes the new model coherent, and it
removes settings a user may have set. That is the owner's call, not the
agent's.
