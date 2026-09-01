# ADR-0036: Extensions host — apps on schema-driven primitives, launched from an Apps tab

- **Status**: accepted (amends ADR-0026's four-tab sidebar; complements ADR-0010 —
  pi packages stay the surface for agent capabilities)
- **Date**: 2026-08-31

## Context

PiCode has two extension stories today and both stop short of the GUI. Pi
packages (ADR-0010) extend the *agent* — tools, skills, themes — inside the
pi process, and PiCode deliberately does not own that surface. Nothing can
extend *PiCode itself*: no way to contribute a panel, a view, or server
behavior without editing the binary. The owner wants an extensions host and,
as its launch surface, an Apps sidebar tab showing a grid of app icons —
with the Inbox (ADR-0037) as the first app built on it.

Benchmarks converge hard on how such hosts succeed and fail:

- **Dogfooding is the mechanism, not a nicety.** VS Code ships ~100 core
  features as built-in extensions (git, github, emmet) and formalizes
  "first-party consumers mature the API": new APIs are *proposed*, usable
  only by blessed built-ins listed in `product.json`, and stabilize only
  after real usage. Obsidian was built plugin-API-first — ~30 core features
  (Graph view, Canvas, Daily notes) are toggleable plugins on the same API
  the community uses, which is why community plugins can feel native.
  Raycast ported GitHub/Linear/Google Workspace from internal Swift to
  public-API extensions specifically to discover what the API was missing.
  Figma ran first-party widgets internally before opening the FigJam API.
- **The consistent regrets.** (1) In-process third-party JS can never be
  walked back: VS Code and Obsidian both admit they cannot retrofit
  permissions, and live on a marketplace-malware treadmill. (2) Extensions
  on the UI thread: JetBrains has spent years digging out of EDT freezes;
  VS Code's founding rule is "no DOM access". (3) Webview-first UI: the
  canonical postmortem is VS Code notebooks — Jupyter rendered whole
  notebooks in an extension webview, performance and accessibility
  suffered, and the shell moved into core with extensions plugging in data.
  (4) Unversioned host APIs: JetBrains' verifier bureaucracy vs Zed's
  per-extension API version. (5) Eager loading: everyone converged on lazy
  activation from declarative metadata.
- **The admired middle path is constrained primitives.** Raycast extensions
  render only a fixed vocabulary (List, Grid, Detail, Form) serialized as
  JSON and rendered natively — every extension inherits the theme, the
  keyboard model, and cannot ship deceptive UI. Stripe Apps and MCP Apps
  (spec, Jan 2026) borrowed the idea. Sandboxed iframes are the industry's
  only trusted escape hatch for free-form UI (Figma's conclusion after a
  Realms-shim sandbox escape), and they demand a separate origin and CSP
  discipline PiCode does not need to take on in v1.
- **Hosting surface.** Slack App Home, Teams personal apps and Discord
  Activities all open apps as a main-area surface from a sidebar/rail icon;
  nobody hosts an app's body inside the narrow sidebar. Teams' pinned app
  rail is the closest analog to the owner's mock. Grids beat lists when the
  icon is the distinguishing feature (Raycast's own Grid guidance); an
  empty grid is a launch mistake — seed with first-party apps, exactly how
  Obsidian ships core plugins enabled.

## Decision

PiCode gains an **apps host**: an app is a **manifest plus schema-driven
UI**, never code loaded into the PiCode process or page.

1. **Manifest.** Each app declares `{id, name, icon, apiVersion, badge?,
   surfaces}`. `apiVersion` is mandatory from day one (Zed's lesson). The
   sidebar and grid render entirely from manifests — no app code runs until
   the user opens the app (VS Code's lazy-activation split).
2. **UI = primitives, rendered by PiCode.** An open app is a main-area tab
   (peer of agents/terminals/editors) whose content is a JSON tree of
   PiCode-rendered primitives — list, detail (markdown), form, actions —
   the Raycast model. Apps cannot touch the DOM, own the tab strip, or
   draw their own chrome; badges, focus and close/restore stay in the host
   (Gamma's rule: core owns what an app must never break).
3. **v1 apps are first-party and in-binary.** They are Go handlers behind
   `/api/apps/<id>/...` returning primitive trees, but they pass through
   the same manifest → grid → primitives pipeline a third-party app would.
   This is the proposed-API tier: the contract is exercised and versioned
   before anything external is allowed in.
4. **Apps sidebar tab.** A fifth tab (amending ADR-0026's four): a grid of
   app icons with name below, phone-style per the owner's mock. Badges:
   numeric count only for actionable items (e.g. Inbox blocking questions),
   a plain dot for other activity. The grid is keyboard-navigable and is
   never empty — first-party apps (Inbox first) ship pre-seeded.
5. **Division of labor with pi packages stands.** Agent capabilities keep
   going through `pi install` (ADR-0010). The apps host is only for GUI and
   PiCode-server surfaces. Where a feature needs both, it pairs a pi
   package with a PiCode app (the `packages/pi-roles` precedent, ADR-0028,
   extended to the GUI side).

## Refuse

| Asked for | Refused because |
| --- | --- |
| Third-party JS in the PiCode process or SPA | Every precedent (VS Code, Obsidian, Open WebUI) says it cannot be walked back; in a server product the blast radius is the server and all its data. |
| Free-form HTML / iframe apps in v1 | The escape hatch exists (sandboxed iframe on a separate origin — Figma/VS Code non-negotiable) but is v2: primitives cover pick-an-item, show-a-report, fill-a-form, approve-an-action. |
| WASM plugin runtime | Right when untrusted logic must run in-process (Figma, Zed); a Go server gets equal isolation from plain subprocesses. Zed's ecosystem ceiling is the warning. Revisit only for per-tenant untrusted logic. |
| App marketplace / gallery | ADR-0010's stance holds: no parallel marketplace. v1 has no third-party distribution at all; when it comes, git-repo-as-registry first (Zed/Claude Code pattern), knowing it gets replaced. |
| Apps rendered inside the sidebar | No benchmark hosts app bodies in the rail; sidebar shows icon + badge, content opens as a main-area tab. |
| Eager app activation | Manifest renders the surface; code (server handler) runs on open only. |

## Consequences

- The primitive vocabulary is now an API: adding a primitive is cheap,
  changing one is a versioned contract (`apiVersion` gates rendering).
  It will be too small at first — by design; the Inbox (ADR-0037) is the
  forcing function that grows it against a real feature, not speculation.
- First-party-only v1 means "extensions host" ships without extensibility
  for outsiders. Accepted: VS Code's proposed-API tier worked in exactly
  this order, and the pipeline (manifest → grid → primitives → versioning)
  is the part that must be right before anyone external hits it.
- A fifth sidebar tab spends header space ADR-0026 already called tight;
  the brand version yields below the same breakpoint.
- If wrong: the grid is presentational; manifests are data; the in-binary
  apps degrade to plain routes + views, which is what they would have been
  without this ADR.

## Alternatives considered

- **Extend pi's own extension system to draw PiCode UI** — pi extensions
  run inside the pi process per agent; PiCode surfaces (sidebar, tabs,
  server routes) are outside their world. The agent platforms that solved
  GUI extensibility (MCP Apps, MCP-UI) did it host-side, exactly this shape.
- **Webview/iframe host first** — the VS Code notebooks postmortem and
  Trail of Bits' webview escapes argue the opposite order: primitives
  default, iframe escape hatch later, if ever.
- **In-process plugin JS (Obsidian model)** — richest ecosystem per user in
  the survey, but its security story is purely social and increasingly
  under strain; unacceptable for a server product. See Refuse.
- **Skip the host, build Inbox as a hardcoded feature** — loses the only
  reliable way to know the app API is sufficient (the dogfooding evidence
  above), and the owner's stated goal is the platform, not one inbox.

## Amendment 2026-08-31 — iframe is the marketplace's first-class body; primitives freeze

Decided with the owner the same day the host shipped, settling the
third-party question ahead of time:

1. **When the marketplace era arrives, the sandboxed iframe is the
   first-class surface for an app's body** — not a reluctant escape
   hatch. Built once, properly: separate origin (the Figma/VS Code
   non-negotiable), strict CSP, postMessage bridge with an audited API,
   and an official published tokens + component package so third-party
   apps can *choose* to look native. A manifest will declare its surface
   (primitives view vs. frame). The "iframe apps in v1" refusal above
   stands unchanged; this fixes the v2+ direction.
2. **Primitives stay, with a narrower and permanent role**: the cheap
   default for simple apps (the Raycast evidence: most tool apps are
   CRUD and their authors prefer writing no frontend), the connective
   tissue in host chrome where an iframe cannot go (inbox items, sidebar
   rows, palette entries, notifications), and the **only** valid surface
   for sensitive actions — approving an agent's action, destructive
   confirms. Tokens don't stop phishing; host-rendered controls do: a
   third-party frame never draws the button that authorizes anything.
3. **The primitive vocabulary is frozen** at list / detail / form /
   actions plus what a first-party app concretely forces. It has no
   ambition to become a UI framework — expressiveness beyond it is what
   the frame surface is for. This supersedes the "forcing function that
   grows it" line under Consequences: the Inbox may still add a block it
   genuinely needs, but growth-by-default is over.

The layered end-state is VS Code's: declarative contributions and
webviews coexisting for a decade, each in its role, neither legacy.
Everything shipped in v1 — manifest, registry, tab, grid, badges, tab
family, routes — is surface-agnostic and carries over unchanged.

## Amendment 2026-09-01 — the first app grew fields, not block types

Dogfooding did what it was supposed to: the Inbox (ADR-0037) hit the
walls of a v1 vocabulary that could only say title/subtitle/badge, and
the surface read like a terminal dump. The growth is deliberately the
kind the freeze allows — **the four block types stand unchanged**; what
was added is optional, additive metadata plus one layout hint:

- `View.Layout` (`"" | "split"`) and `Block.Pane` (`"" | "list" |
  "detail"`) — a *hint*, not a container: blocks stay a flat list and the
  host decides the arrangement (list left, detail right; stacked under
  880px). `View.Empty` carries the app's own blankslate line so the host
  can own how emptiness looks without inventing the copy.
- `Block.Title` / `Meta` / `At` and `ListItem.Meta` / `At` / `Tone` /
  `Unread`, plus `Action.Icon`. Timestamps travel as RFC3339 and are
  formatted by the host (relative in the row, absolute on hover) — an
  app never pre-formats time.
- `APIVersion` stays 1: the fields are optional and the embedded UI ships
  in the same binary as the server, so both sides move together. A
  client that ignores them renders exactly what it rendered before —
  proven by the demo app, which was not touched.

The tone vocabulary is deliberately small and semantic (`info|ok|warn|
danger`), with red reserved for destruction: an approval that *asks*
about a destructive action reads `warn`, and only the Ignore button that
denies an agent its reply is `danger`.

## Amendment 2026-09-01 — visibility needed one more field: `View.Tabs`

Dogfooding found a second real gap the same day: `done` inbox items are
`UPDATE`-only from the start — never deleted — but had no way to reach
them. The default view only ever queried `unread`/`read`, so a resolved
question just disappeared, recoverable only through a raw API call. The
fix is the same shape as the amendment above, once more: an optional
field, not a fifth block type.

- `View.Tabs []Tab` — `{id, label, path, badge?}`, rendered as a
  segmented control above the view (Active/Done/All for the Inbox).
  `Tab.Path` reuses the exact navigation mechanism `ListItem.Path`
  already had — a click is the same `setPath` the client already runs,
  nothing new on the wire or in the client's state model.
- Deliberately its own type, not `[]Action`: an `Action` carries
  `Confirm`/`Danger`/`Primary`, all meaningless for navigation, and
  reusing it would let a tab inherit destructive styling by accident.
- Manual cleanup needed no new primitive at all: a per-row `delete`
  action and one bulk `clear-done` action reuse the existing
  `Action`/`Confirm` vocabulary as-is. Retention/auto-sweep policy is
  explicitly future work, out of scope here by the owner's choice.

Block types are still the frozen four; `View` now carries `Layout`,
`Empty` and `Tabs` as its optional, app-agnostic hints.

## Sources

- VS Code built-in extensions: <https://github.com/microsoft/vscode/tree/main/extensions>;
  proposed-API pipeline: <https://code.visualstudio.com/api/advanced-topics/using-proposed-api>,
  <https://github.com/microsoft/vscode/wiki/Extension-API-process>
- Notebooks-out-of-webviews postmortem: <https://code.visualstudio.com/blogs/2021/11/08/custom-notebooks>;
  workbench limits: <https://code.visualstudio.com/api/extension-capabilities/extending-workbench>;
  Erich Gamma on extensions hurting you: <https://www.theregister.com/2021/01/28/erich_gamma_on_vs_code/>
- Extension host process model: <https://code.visualstudio.com/api/advanced-topics/extension-host>;
  runtime security limits: <https://code.visualstudio.com/docs/configure/extensions/extension-runtime-security>;
  webview escapes: <https://blog.trailofbits.com/2023/02/21/vscode-extension-escape-vulnerability/>
- Raycast primitives + porting first-party features: <https://www.raycast.com/blog/how-raycast-api-extensions-work>;
  Grid-vs-List guidance: <https://developers.raycast.com/api-reference/user-interface/grid>
- Obsidian core plugins: <https://help.obsidian.md/plugins>; plugin-API-first
  (Licat): <https://robhaisfield.com/notes/building-community-in-obsidian-with-licat>;
  trust limits: <https://forum.obsidian.md/t/security-of-the-plugins/7544/19>
- Zed WASM/WIT + versioned API: <https://zed.dev/blog/zed-decoded-extensions>;
  capability ceiling: <https://zed.dev/docs/extensions/capabilities>
- JetBrains in-process pain: <https://blog.jetbrains.com/platform/2025/09/investigating-intellij-platform-ui-freezes/>,
  <https://plugins.jetbrains.com/docs/intellij/verifying-plugin-compatibility.html>
- Figma sandbox history: <https://www.figma.com/blog/how-we-built-the-figma-plugin-system/>,
  <https://www.figma.com/blog/an-update-on-plugin-security/>; FigJam
  internal-first widgets: <https://www.figma.com/blog/bringing-the-power-of-our-open-platform-to-figjam/>
- Dify monolith → out-of-process plugins: <https://dify.ai/blog/dify-plugin-system-design-and-implementation>
- MCP Apps (GUI-for-agent-tools consensus): <https://blog.modelcontextprotocol.io/posts/2026-01-26-mcp-apps/>,
  <https://github.com/MCP-UI-Org/mcp-ui>
- Hosting/launcher UX: Slack App Home <https://docs.slack.dev/surfaces/app-home/>;
  Teams personal apps/rail <https://learn.microsoft.com/en-us/microsoftteams/platform/tabs/what-are-tabs>;
  Discord launcher <https://support-apps.discord.com/hc/en-us/articles/26592957841303-How-to-Discover-and-Add-Apps>
- Blender add-ons onto the public platform: <https://code.blender.org/2024/05/extensions-platform-beta-release/>;
  API dogfooding argument: <https://zapier.com/engineering/api-dogfooding/>
