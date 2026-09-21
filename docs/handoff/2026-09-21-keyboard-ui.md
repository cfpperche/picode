# 2026-09-21 — feat/keyboard-ui: the Keyboard pane, rebuilt (P0 of docs/plans/keyboard-pane.md)

Shipped: the Pi keyboard map is a screen of its own — 32px rows, the chord as a
keycap at `--kbd-h` (was `--ctl-h`), the row's own `Reset` / `Add key` in a
fixed right column (Reset left of Add, so Add keeps the slot the reader
learned), and a toolbar that sticks to the top of the CLI page's own scroller:
filter, **Find by key**, counting facets (`All`, `Changed`, `Shared`, `Off`),
the row count, **Reset all** behind a confirm. A changed row wears the guest
settings' accent bar; chords a browser keeps are labelled on their row. API: the
report gained `file`/`exists`/`platform`, `PUT` gained `{resetAll: true}`
(`pikeys.ResetKnown` — one write, unknown keys kept); the catalog gained
`app.thinking.save` and nine `Alt` rows read out of pi's own docs. The pane's
CSS is scoped under `.key-pane`, because AppKeys (Preferences → Keyboard) shares
those class names — verified untouched: 36px chips, flex rows. Two shared-chrome
fixes rode along: `.cli-pane-tabs` scrolls inside its card instead of running
past it at 1024px, and desktop `.btn-danger` has a rest state (it was hover-only,
so every desktop confirm drew Remove and Cancel alike).
Verified: `make ci-scoped` PASS; `scripts/qa-cli-settings.mjs` on the docs
fixture — **32 rows pass on both apps**, every keyboard row included, plus the
stale settings rows repaired here; `__picodeOverlayAudit()` ok on every audited
capture. visual-review: **PASS** (card 5/5, second pass;
`var/screenshots/keyboard-pane/`). The first pass found four issues, all fixed —
the tab-strip clip, the Add/Reset column shift, the neutral destructive button,
duplicate "scrolled" shots — and two findings were dismissed with measurement: a
settled mobile toast sits at top 714–772 in an 844 viewport, and the "clipped
toast" was a frozen exit animation below the fold.
Not done: the harness's post-loop untrusted/trust/free-agent matrix still stops on
a stale `ready()` from the native-settings landing (`open/agent-clis-native.md`).
Merge: fast-forward ready.

## Next up

- P1 of `docs/plans/keyboard-pane.md`: one report envelope for every CLI (`cliKeys.js` + the `/api/pi-keys` wrap), then P2's `internal/clikeys` with Omp first.
