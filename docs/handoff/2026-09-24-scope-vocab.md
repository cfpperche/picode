# 2026-09-24 — scope-vocab: one scope vocabulary for every setup tab's address

Owner asked for one strategy after the pane-scope fix (a translator between tabs).
`web/shared/domain/scopes.js` holds the shared words (global/workspace/agent), each
tab's own words (PACKAGE, SKILL, LAYER, MODEL, MEMORY) and `readScope`/`writeScope`.
Every route module writes `?scope=<shared word>` and reads it into its own word, so
components and APIs are unchanged; older words and `?layer=` are aliases that set a
redirect to the shared form. Routes expose `scopeKind` only when the address named a
scope; `cliPaneSetupContext` carries that (never a tab's default) and `cliSetupHref`
maps it into the next tab's words. `paneScope` is gone; `scopeIcon.js` re-exports
`scopeKind`. `CliSettings.writeRoute` reads the layer through `settingsLayerOf`.
Changed on purpose: `layer=user` now reads as Global (user is Global's alias) instead
of being dropped.
Verified: route tests updated plus a table test for every tab; scratch: Skills QA →
Packages → Connectors, Packages agent → Skills, Settings QA → Packages keep the chip;
old `layer=project`, `scope=project`, `scope=machine` links rewrite; no invalid page.
Docs: routes.md (new paragraph), cli-settings.md, integrations.md, guide/settings.md.
