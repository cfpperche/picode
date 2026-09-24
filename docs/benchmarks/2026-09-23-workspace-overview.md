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

## V2 adaptation (2026-09-24)

| Reference | PiCode adaptation |
|---|---|
| [Linear project updates](https://linear.app/docs/initiative-and-project-updates) place changes and health beside the current project. | Show "Since your last visit" only within PiCode's seven-day event retention; never infer a health score from incomplete signals. |
| [GitHub Pulse](https://docs.github.com/en/enterprise-server%403.19/repositories/viewing-activity-and-data-for-your-repository/using-pulse-to-view-a-summary-of-repository-activity) combines recent repository actions. | Join recent Git commits with a small, workspace-scoped orchestration event summary. |
| [GitHub status checks](https://docs.github.com/en/pull-requests/reference/status-checks) distinguish failed, pending and passed validation. | Put failed checks in attention and show current check counts beside the PR link when GitHub data exists. |
| [Cursor's agent status bar](../benchmark-cursor.md) keeps agent state short and actionable. | Use existing fleet status vocabulary; a waiting agent opens its existing surface. |

Reuse the existing Recharts chart, PageFrame and native controls. The V2 adds
no presentation dependency and no new editor tab.
