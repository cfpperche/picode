### Changed

- **The Canvas background is now chosen on the canvas.** `⋯` → **Background**
  opens a submenu with Plain, Dots, Grid and Cross, each showing a real
  sample of its texture and a tick on the one you are using. The rows stay
  open while you pick, so the plane behind the menu is the preview. Your
  choice is the same one you had — it is still remembered per browser, and
  nothing on a canvas moves when the texture changes.
- **Preferences → Appearance is PiCode's own chrome again**: the theme, and
  nothing else. The **Canvas background** group has left it, and so has the
  `⋯` menu's old `Background…` item, which existed only to send you there.
  An app's settings now live in the app — a project rule from today, not
  just a tidy-up (ADR-0109).
- **Messages' list of hand-drawn links reads as Messages'**, because that is
  whose it is: it is now **Granted contacts** — the pairs you allowed to
  message each other by drawing a link between two panels on a canvas. Same
  rows, same Remove, same guarantee that a link never lets one session read
  another's history.
