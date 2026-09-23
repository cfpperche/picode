# 2026-09-23 — missions-guide: explain the Missions workflow

Changed: `docs-site/guide/missions.md` now explains first use, independent lifecycle signals, every screen action, evidence and review rules, transfers, and recovery. Added `docs/changelog.d/missions-guide.md`.
Verified: `make ci-docs` built the VitePress site; `make close-summary` identified a docs-only change. This was a documentation build, not a live UI or agent workflow check.
visual-review: n/a (no UI implementation changed).
Not done: the five-mission owner pilot remains tracked in `docs/handoff/open/missions.md`. No pilot or deploy was performed in this session.
Merge: follow the branch's fast-forward landing and full CI in Git history.
