# 2026-09-14 — term-stopped-state: the stopped terminal is an empty state

Shipped: a stopped CLI terminal — and a pane whose open refused — is the app's
empty-state shape, not a sentence in the corner of a black well: mark, what is
true (stopped / restarted under it / last launch failed), the pinned
conversation **Resume last session** would reopen (CLI · name · age, opening
words on hover — it lived only in a tooltip), then the actions. Nothing pinned
is a stated reason (*no session to resume*); a failed resume says why. Desktop +
phone (`TermSurface.jsx`, `.term-msg`, `--term-*`; `routes.md` has the section).

Verified: agent-browser on scratch :8481 over real stopped terminals (seeded via
API + DB pin). Read: stopped+pinned, no-pin, failed launch, resume error from a
real click on Resume, dark/dark, light/light, 640px pane, 256×224 probe, phone
390×844, overlay audit `ok: true`, QA preset passed (text×4, network, console,
errors), quiet action clicked → `#/clis`. Shots `var/screenshots/final-*.png`.

visual-review: PASS — the pane says what stopped and what Resume continues; the
first cut inked the message per *terminal* theme, the owner ruled the empty
pane is app chrome and follows the app theme (a light app had a black well), so
the pane carries `is-empty`, the ground is `--bg-base` and app tokens returned;
under 300px the mark goes and actions stack (Canvas panel minimum 256×224).

Debts: mobile `Terminal.jsx` keeps its own "That terminal is gone." /
"Attaching…" states in `m-tool-state`. A Canvas panel bound to a stopped CLI
terminal never loaded its body headless (unloaded placeholder), so that path was
measured by mounting the real subtree at 256×224. `make ci-scoped` PASS,
`make close` PASS; main merged in, fast-forward ready from the root.

## Next up

- Canvas panel bodies never loaded in a headless session; re-check the stopped-pane panel on a real machine.