# 2026-09-14 — term-window: the stopped state sits in a terminal window

Shipped: the stopped CLI terminal (and a refused open) is drawn inside
**TermWindow** — a picture of a terminal window per the owner's ohmyz.sh mock:
titlebar with the macOS traffic lights, the terminal's own name and folder as
the centered title (`codex — <folder>`, full cwd on hover), the state content
hugged inside, the whole window centered on the pane (max 520px, 360 on the
phone) instead of filling it. The page keeps the app theme
(`.term-surface.is-empty` → `--bg-base`); the window is dark in both app
themes — it is a picture of a terminal — with its palette pinned to the dark
theme's own values as `--tw-*`, and the quiet action wears the window's
chrome while the primary keeps the accent. Under 300px (Canvas panel minimum)
the title hides and the actions stack. A live terminal is untouched.
Files: `TermSurface.jsx` + `app.css` / `mobile-tools.css` (both apps);
`routes.md` has the rule.

Verified: agent-browser on scratch :8490 over real stopped terminals. Read:
light app (pinned + bare), dark app, 256×224 probe (title hides, actions
stack, nothing clips), phone 390×844, live shell unchanged (no is-empty,
terminal ground). QA preset passed (text×3, selector, network, console,
errors); overlay audit `ok: true`. One real defect caught and fixed on the
way: the phone's `TermWindow` was referenced before its definition — the
error boundary caught it, fixed and re-verified. Shots:
`var/screenshots/window-*.png` (not committed).

visual-review: PASS — the window reads as a terminal on a themed page in both
themes; content, actions and title all inside the frame; no clipping at any
probed size.

Gates: `make ci-scoped` PASS, `make close` PASS. Merge: main merged in,
fast-forward ready from the root.

## Debts

- The traffic-light dots are pinned hex values (#ff5f57/#febc2e/#28c840, the
  macOS mock's own); if the mock ever leaves, they leave with it.