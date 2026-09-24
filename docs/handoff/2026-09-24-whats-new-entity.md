# 2026-09-24 — feat/whats-new-entity: the release card renders text, not markdown

Verifying the published 0.7.0 artifact on an isolated instance (checksum OK, `/api/health` ok, `/api/version` `release:true semver:0.7.0`, and the What's New surface opening once on a fresh data directory) put the card on screen — and the fork highlight read `fork of &lt;name&gt;;`. I had copied the entity from the changelog, where markdown renders it; the card renders the string as text, so the entity showed raw. It is the only `&lt;` in the whole catalog, and it is mine.
Fixed here (plain `<name>`), which means the source and the deploy carry the correction while the published v0.7.0 artifact keeps the raw entity. `docs/release-process.md` is explicit about that case: a corrective patch, never a rewritten tag — so the recommendation is 0.7.1 when the owner wants one.
Verified: catalog parses, the entry reads `fork of <name>.`, no entities left anywhere in `whats-new.json`.
