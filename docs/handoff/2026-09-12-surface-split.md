# 2026-09-12 — feat/surface-split: /browser/ + /desktop/ (owner decision)

The owner renamed the surfaces instead of accepting /appshell: the
responsive web app now serves at /browser/ (package web/desktop became
web/browser, @picode/browser), and a new thin web/desktop package serves
the shell's own composition at /desktop/ — an entry rendering the
browser app's exported App with a shellChrome prop, plus the Management
page, whose tokens now import from @picode/shared (hand-copied block
deleted). The shell loads /desktop/ directly (launcher skipped) and the
__PICODE_SHELL__ global is extinct — the desktop bundle is shell-aware
by construction, the browser bundle never hears about windows.

Boundary exception: desktop may import exactly the browser package's
named exports (COMPOSES in boundaries.mjs; presentation-dependency ban
kept per root: shared bans react, composed browser is the app itself).
Touched: server cache prefixes + ui_cache_test, launcher links/proxies,
shell picker + tests (surface "desktop" now lives at /browser/), docs
screenshot profiles, qa scripts, term-scrollbar test.

Verified: npm tests green (web 294 + tools + shared), make web builds
all four, cargo xwin build, ci-scoped PASS; shell exe installed +
relaunched, PID confirmed. Owner-visual: shell shows the /desktop/
bundle (single wordmark, dashboard via the bar); browser at /browser/.

Debts: docs-site guides may still mention old paths in prose; the
qa-scratch script's captured URLs.

Merge: fast-forward ready.
