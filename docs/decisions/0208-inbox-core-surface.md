# ADR-0208: Inbox is a core PiCode surface

- **Status**: accepted (owner direction, 2026-09-23; amends ADR-0036 and ADR-0037)
- **Date**: 2026-09-23
- **Boundary**: protocol — the Inbox view, action and badge move from the apps manifest contract to `/api/inbox`; existing app view/action URLs remain compatibility aliases.

## Context

ADR-0037 put the mailbox and delivery in core but made its view the first app to exercise ADR-0036's primitives API. That exercise succeeded. The Inbox now carries questions, approvals, results and reminders from across PiCode. On desktop it was still a tile beside optional apps, while the phone already gave it a top-level destination. The owner chose a dedicated button in the desktop sidebar header, with the pending badge visible from every page.

The mobile V2 benchmark ([note](../benchmarks/2026-09-07-mobile-v2.md)) adapts stable top-level destinations from Apple tab bars and treats attention notifications as normal work (Happy). Desktop follows that same access principle while keeping its own chrome.

## Decision

The Inbox is a PiCode surface. Its model and primitive-tree presentation live in `internal/inboxview`, outside the apps registry. `/api/inbox/view`, `/api/inbox/action` and `/api/inbox/badge` serve it independently of installed apps. The desktop button opens `#/inbox` directly; item links use `#/inbox/<id>`, shared with the phone and push. The desktop Apps grid and its aggregate badge no longer include Inbox. Old `#/app/inbox[/<path>]` links replace to the matching core route, and old `/api/apps/inbox/{view,action}` clients continue to work through aliases. The existing primitives renderer remains shared; storage, delivery and triage semantics do not change.

## Implementation plan

1. Extract the Inbox view and action model from the apps registry and serve it from core routes.
2. Give the desktop header an Inbox button and badge; point desktop and phone views at the core routes.
3. Translate old hashes, retain API aliases, and remove saved Inbox app tabs and the Apps tile.
4. Update the architecture, public guide, generated OpenAPI and screenshots; pass scoped and merge gates plus visual review.

| Conditions | Action | Evidence |
|---|---|---|
| Blocking items exist | Show a count on the Inbox button; open the core page | `TestInboxCoreSurfaceWithoutAppsRegistry`; scratch badge/action review |
| No blocking item, other open item exists | Show an activity dot | `TestInboxBadgeApp`; shared `CountInboxBadge` |
| Mailbox empty | Show no badge; render the Inbox empty action | `TestInboxRootView`; scratch empty review |
| Apps registry absent or failing | Inbox routes and badge still answer | `TestInboxCoreSurfaceWithoutAppsRegistry` |
| Old app item hash or API URL | Resolve the same item through the core page or alias | route test, server test, scratch legacy-link review |
| Saved `x:inbox` tab | Drop it during tab restoration | tab restoration predicate in `App.jsx` |

## Consequences

- A pending question has a visible badge on the Inbox button without an app manifest read. The apps registry can be absent and the mailbox still opens.
- The desktop header gains one control. At narrow widths the brand collapses to its mark and the seven controls tighten; visual review must cover the minimum width.
- Apps no longer dogfoods the mailbox view, but Docker, tmux and the demo still exercise primitives. No database migration or new dependency is needed.
- The core page is route based, like Automations and Snippets, so it does not persist as an app tab. Saved `x:inbox` tabs are dropped on restoration; old deep links still reach the mailbox.

## Alternatives considered

| Option | Why it lost |
|---|---|
| Keep the app and add a sidebar shortcut | The user would still see Inbox in Apps, and core availability and badge would still depend on the apps manifest. |
| Put Inbox in the user menu footer | Questions awaiting a human would be less visible, and one footer control would have two destinations. |
| Build a separate bespoke view renderer | The shared primitives already express every Inbox state; duplicating it would add maintenance without improving the core boundary. |

## Amendment 2026-09-25 — old addresses retired (owner)

The owner retired the compatibility addresses this ADR kept. The old `#/app/inbox[/item/<id>]` hash no longer redirects on desktop or mobile, and (same day, later) the `/api/apps/inbox/{view,action}` API aliases were retired too: the UI reads `/api/inbox/*`, and only tests still used the aliases. An old bookmark now lands where any unknown address does.
