# 2026-09-15 — feat/browser-tab-crash: the work browser's blank window (main)

One bug, one file, three lines: `WebTabSurface` named `showFullUrl` in the
meta effect's dependency array two lines above its own `useState` (from
`4f1a68a7`, slice 3 "Show full URL"). A `const` read in the same render before
its declaration is a `ReferenceError`, and React unmounts the root: every
work-browser tab, and the whole window with it. "New browser tab" (tab-strip
end) was the shortest way in; it was already on the board as a known bug.
Measured before/after rather than assumed: two scratches of this same machine,
`:8474` from main and `:8475` from this branch — main gave a blank page with
`ReferenceError: Cannot access 'E' before initialization` at the minified
WebTabSurface and `#root` empty; the fix gives the tab, its one-line notice in
a plain browser, and no page error, on both the click and a reload with the
tab open. Screenshots: `var/screenshots/crash-{before-main,after-fix}.png`.
The state now sits with the other `useState` calls (order among them
unchanged), with the reason written next to it so a refactor does not move it
back. Swept the rest of `web/**` for the same shape — a `useState` binding
named in a deps array above its declaration — zero other instances.
No guard exists to catch the class: this repo has no JS linter (the CI lint
step is Vale, prose) and no component test rig; extending biome to `web/` is
its own task, noted in `docs/handoff/open/work-browser-tabs.md`.
Verified: `make ci-scoped` PASS; scoped tests; before/after screenshots read.
Not verified: the shell's own WebView2 path (no Tauri host here) — the fix is
above the branch and cannot differ.
visual-review: PASS (5/5)
