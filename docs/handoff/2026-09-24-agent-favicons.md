# 2026-09-24 — feat/agent-favicons: sidebar CLI favicons wear the tab-strip treatment
Shipped: CLI favicon images render full bleed everywhere — `img.ws-face` is
transparent/borderless; the white plate survives only on letter/glyph
fallback spans (web/browser/src/styles/app.css, web/mobile/src/styles/app.css,
web/shared/styles/providers.css). CliAgentFace, TermFace and the notice face
carry `term-cli-face`, so monochrome vendor marks invert to white on dark
(browser + mobile ProviderFaces.jsx / Notice.jsx). The collapsed workspace
strip swaps its -6px chip overlap for a 2px gap — full-bleed marks no longer
smudge together. The official URL lists in web/shared/domain/terminalCli.js
are untouched: the unification the owner authorized was class/CSS-level.
Verified: `make ci-scoped` PASS (fmt, vet, hooks, test-js 54/54, build);
scratch instance (qa-scratch favicons) screenshots read in a subagent —
dark/light × expanded rows/collapsed strip on desktop, dark/light on mobile;
faces confirmed full-bleed, separated, readable (var/screenshots/final-*.png,
worktree-local). Blind spot: the Windows desktop shell was not exercised,
only the served web UI at 1706x960 and a 390px mobile viewport.
visual-review: PASS (desktop card 5/5, no regression vs the owner's report)
Merge: fast-forward ready.
