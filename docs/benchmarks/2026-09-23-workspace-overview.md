# Study: workspace overview

- **Date:** 2026-09-23
- **Surface:** a route from the sidebar workspace menu, not an editor tab.
- **Sources:** [Linear project overview](https://linear.app/docs/projects),
  [GitHub Pulse](https://docs.github.com/en/enterprise-server%403.18/repositories/viewing-activity-and-data-for-your-repository/using-pulse-to-view-a-summary-of-repository-activity),
  [Langfuse dashboards](https://langfuse.com/docs/metrics/features/custom-dashboards),
  [Recharts API](https://recharts.github.io/en-US/api/) and
  [shadcn chart recipes](https://ui.shadcn.com/docs/components/base/chart).

| Observed pattern | PiCode adaptation |
|---|---|
| Linear puts project context beside current progress and links to detailed work. | One compact page starts with needs-you, Git and agents, then missions and activity; every item opens its established action surface. |
| GitHub Pulse summarizes a selected period before its detailed repository views. | A period picker drives recorded workspace activity; Git stays a direct link to the existing graph. |
| Langfuse gives a dashboard a scope and a time range. | Workspace identity is in the route, range in the control; the home dashboard retains machine-wide meaning. |
| Recharts supplies responsive charts, tooltips and keyboard accessibility. | One lazy-loaded bar chart uses PiCode tokens; no second design system or chart framework. |

Cursor's density and the existing PiCode `PageFrame` set the chrome. The page
does not copy Linear's issue model or GitHub's hosted repository analytics.
It uses local PiCode missions, inbox and Git facts instead.
