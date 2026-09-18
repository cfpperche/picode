# ADR-0153: webapp-partitions

- **Status**: accepted
- **Date**: 2026-09-18
- **Boundary**: persistence and security model — where an installed web app's credentials live (one shared WebView2 profile → one user-data folder per app) and how far a "clear browsing data" reaches.

## Context

Installed webapps (ADR-0147) render in the shell's work-browser webviews, which all share **one** WebView2 profile (`%LOCALAPPDATA%\PiCode\WebView2`, browserlab.rs). One shared folder means: one account per service (a second login replaces the first), and "clear browsing data" logs out every installed app together with the work-browser sites. ADR-0147 deferred this ("per-webapp partitions — v2") and the plan (`docs/plans/installed-webapps-v2.md`, feature 2) picked it up with the owner's approval.

WebView2's isolation unit is the **user-data folder** given at webview creation (`data_directory`). Same origin separation between two folders is enforced by the engine; no daemon or shell code mediates it.

## Decision

An installed webapp's webview gets **its own user-data folder**, derived by the shell from the stable tab id (the ADR-0147 door): tab id `app-<webappId>` (the shell-half of `w:app-<webappId>`) → data directory `%LOCALAPPDATA%\PiCode\WebView2\webapps\<webappId>\`. Rules:

- **Derivation is single-source**: the id the UI already sends. The shell validates the id segment against `[a-z0-9][a-z0-9-]{0,79}` — anything else falls back to the shared profile (defensive; ids are minted by `store.newID` and match this shape).
- **Only installs born after this ADR partition.** The store marks them (`webapps.partitioned = 1`, migration 056); rows from before keep the shared profile until the user removes and re-adds the app. There is no cookie migration between profiles — Chromium makes copying HttpOnly/session state fragile, and the honest cost is one re-login per legacy app (documented in the UI plan and here).
- **Per-profile settings travel with the partition**: autofill and the download folder are applied per-controller at creation (`ICoreWebView2Profile` of that webview) and therefore land in each app's own folder.
- **"Clear browsing data" clears the work profile only.** App partitions are untouched — their data belongs to that app; a per-app clear is a future want, not this decision.
- **CDP and permissions are unchanged**: attach is per-webview and the origin grants of ADR-0128/0132 do not care which folder holds the cookies.

## Consequences

- Two accounts of the same service become possible (install the same site twice under different addresses/paths — uniqueness is per normalized URL).
- The blast radius of "Clear browsing data" narrows to the work profile; an installed app's login survives it.
- Disk: one WebView2 folder per app (tens of MB each), under the folder `picode-desktop disk` already reports wholesale — accepted.
- If wrong: the derivation is one function and one column; moving back is remove + re-add with the flag off.

## Alternatives considered

- **Cookie migration for legacy apps** (export from the shared profile, import into the new one) — refused: HttpOnly/session/sameSite state across profiles is fragile and silently wrong; a one-time re-login is honest.
- **Partition the work browser too** (per workspace/project, the Cursor model) — out of scope; that changes the agent-binding story (ADR-0135) and deserves its own decision.
- **Keep the shared profile and rely on engine per-origin isolation** — status quo ante; it already separates *sites*, but cannot give two *accounts* of one site, and keeps clear-data global.
