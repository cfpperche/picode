# 2026-09-15 — browser-scope

Owner approved the browser v2 scope before the permissions implementation
starts; the decision is now dated in the topic file.

## Done

- `docs/handoff/open/work-browser-tabs.md` gains the scope section: v1 (site
  permissions + JavaScript), v2 (agent history access, unused-site cleanup,
  annotations, passkey opener), v3 (WebMCP, raw CDP with its ADR gate), and
  the never-on-WebView2 list with the reason.
- Same file: the measured permission API notes (PermissionRequested,
  GetDeferral, SetPermissionState, the kind and state constants) so the next
  session starts without re-research.
- Same file, traps: the three-edit rule for a new shell command, with the
  four features it cost.
