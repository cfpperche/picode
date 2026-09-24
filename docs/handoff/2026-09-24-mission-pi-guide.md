# 2026-09-24 — mission-pi-guide: explain native Pi Mission reporting

Shipped: Updated `docs-site/guide/missions.md` to name Pi's native `mission` tool in managed and interactive sessions, with no MCP adapter or workspace install. The guide now directs other CLIs to the optional MCP tools or `picode mission` command, and identifies the command as a Pi fallback. Added `docs/changelog.d/mission-pi-guide.md`.

Verified: `make ci-scoped` passed docs and Vale checks. This documentation change was not checked in a live agent session.

visual-review: n/a

Not done / debts: None identified for this guide correction. No deployment was performed.

Merge: landed by fast-forward at `ed8ee0c45`; full CI passed. Publication of the public site remains a separate owner action.
