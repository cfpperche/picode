# Plan: installed webapps v2 — metadata refresh + per-app partitions

> Owner-approved 2026-09-18 (refresh + partitions). Builds on ADR-0147 and
> its amendments; the standalone window stays refused (topic record:
> `docs/handoff/open/installed-webapps.md`). Two branches, landed in order.

## Feature 1 — Refresh metadata (`feat/webapp-refresh`)

A tile's identity (icon, `start_url`, `scope`, `display`, `theme_color`) is
frozen at install; sites rebrand and tiles start lying. The tile menu gains
**Refresh**: the daemon re-resolves the stored URL with the same bounded
fetch and rewrites the manifest-derived fields.

| Slice | Work |
|---|---|
| 1 | Store: `UpdateWebappMetadata(id, WebappInput)` — rewrites icon/icon_mime/start_url/scope/display/theme_color in one transaction, announces `webapp.updated` (invariant row); **name and url are never touched** (the name is the user's editable label, the url is the identity the user chose) |
| 2 | Server: `POST /api/webapps/{id}/refresh` — fetch (same rules: reachable or 502, nothing changes), metadata via `webappLookupMetadata`, icon re-fetched, then the store update; 404 for unknown ids |
| 3 | UI: menu item Refresh with busy state on the tile menu; feed `webapp.*` already reconciles the list — nothing new |
| 4 | Tests: store round-trip (fields change, name/url pinned), server refresh (200 fields updated · 502 site down → row untouched · 404), invariant row; scratch QA: install fixture, change its manifest, Refresh, tile updates |

Decision table (tests cover each row):

| Conditions | Action |
|---|---|
| Site reachable, manifest present | icon + manifest fields replaced; name/url kept |
| Site reachable, no manifest | icon falls back (link/favicon); manifest fields cleared |
| Site unreachable / HTTP ≥ 400 | 502, row untouched, tile menu shows the error |
| Unknown id | 404 |
| Concurrent duplicate concerns | none — refresh never inserts |

## Feature 2 — Per-webapp WebView2 partitions (`feat/webapp-partitions`)

Today every webview shares the work profile
(`%LOCALAPPDATA%\picode-shell\…`, `browserlab.rs`), so one app's "clear
browsing data" wipes every webapp's login, and two accounts of one service
are impossible. Each installed webapp gets **its own user-data folder**.

- **Partition derivation (single source of truth: the tab id).** The shell
  already receives the stable id `w:app-<webappId>` (ADR-0147 door).
  `btab.rs::ensure` derives the data directory from that prefix:
  `%LOCALAPPDATA%\picode-shell\webapps\<webappId>\`. No new IPC arg, no UI
  change; regular work tabs (`w:<n>`) keep the shared profile.
- **Store:** migration adds `partitioned INTEGER NOT NULL DEFAULT 0`;
  installs set it to 1. Only `partitioned` rows launch partitioned —
  existing installs keep their shared-profile login until the user
  removes + re-adds the app (documented; no cookie copying between
  profiles — Chromium makes that fragile, and the honest cost is one
  re-login per legacy app).
- **Per-profile settings travel with the partition:** autofill prefs and
  the download folder are already applied per-controller at creation
  (`ICoreWebView2Profile`); verify they land on partition profiles too.
- **Clear browsing data** keeps clearing the *work* profile only; app
  partitions are untouched (their data belongs to the app; a per-app
  clear is a later want, not this plan).
- **CDP / permissions:** unchanged — attach is per-webview, origin grants
  (ADR-0128/0132) unaffected.
- **Boundary → ADR-0153** (`make adr NAME=webapp-partitions`): persistence
  and security model (where webapp credentials live; blast radius of
  clear-data narrows).
- **Tests/Rust:** partition-path derivation unit test; per-profile prefs
  applied on partition profiles. **Windows-only verification:** the owner
  logs into one app, duplicates a partition via a second install of the
  same site under a different path, and confirms isolation; login
  persistence across shell restart per partition.

| Slice | Work |
|---|---|
| 1 | ADR + migration + store (`partitioned` on create, invariant row) |
| 2 | Rust: derive partition from the tab id, profile settings per partition, unit test |
| 3 | Docs (security-model, data-persistence, routes) + swap via `make desktop-restart` + owner verification on Windows |

## Risks

- Partitions multiply user-data folders on disk (one per webapp; WebView2
  folders are tens of MB) — accepted; `picode-desktop disk` already reads
  `%LOCALAPPDATA%` wholesale.
- A webapp whose login lives in the old shared profile goes silent-logged-out
  after a remove+re-add — the refresh feature does not touch cookies;
  documented in the ADR.
- Refresh drops manifest fields when a site *removes* its manifest —
  intended (truthful), surfaced as the tile falling back to a favicon.
