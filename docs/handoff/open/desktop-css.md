# Desktop bundle stylesheet

The /desktop/ bundle initially shipped without the app stylesheet: its
css chain lived in a file outside the desktop package's vite root, and
the tailwind v4 plugin skips compiling css outside the build root. The
desktop entry now owns index.css inside its root (tailwind scan pointed
at the browser app sources, tokens/app.css/providers imported from the
shared and browser packages).
