# ADR-0010: Pi packages surface

- **Status**: accepted
- **Date**: 2026-08-24

## Context

Pi extends through packages (`pi install`, gallery at pi.dev/packages),
not a baked-in plugin host. Users asked for web search in the ADE; that
capability lives in third-party packages (`npm:pi-web-search`, or the
`brave-search` skill). PiCode must not invent a parallel marketplace or
copy package state into SQLite (ADR-0005).

## Decision

1. **`#/packages` is a pi surface**, like Providers — never a Settings
   section. It lists, installs, and removes what pi already stores:
   user `~/.pi/agent/settings.json` and, when a workspace is selected,
   `<workspace>/.pi/settings.json`.
2. **Mutations go through `pi install` / `pi remove`.** PiCode does not
   rewrite settings.json itself.
3. **In-app gallery search is npm**, not a PiCode catalog and not a scrape
   of pi.dev (no public API; iframe is blocked). Query the registry for
   `keywords:pi-package`. The pi.dev link stays as the official page.
4. **Opt-in only.** Installing is a user action with a full-access warning.
   Nothing is installed by default.
5. **`capabilities.webSearch`** is a derived flag from configured source
   names (`pi-web-search`, `brave-search`, …). Pretty search UI in chat
   may use this later; this ADR does not add that UI.

## Consequences

- **Easier**: one source of truth (pi settings + CLI); first search package
  can land without a PiCode-owned engine.
- **Harder**: install latency and errors are whatever `pi install` returns.
- **If wrong**: scraping npm into a fake store would fork from `pi list`.

## Alternatives considered

- **Ship Brave inside PiCode**: rejected — keys, billing, and the search
  tool belong to a pi package, not our binary.
- **Full gallery browser in-app**: deferred — pi.dev already is that UI.
- **Pretty search cards first**: rejected — without an installed search
  package the cards would be invented.

## Amendment 2026-08-25

The UI now offers **This machine** (`pi install`) and **This workspace**
(`pi install -l`, cwd = selected agent folder). Session-only (`pi -e`,
This run) is still deferred.
