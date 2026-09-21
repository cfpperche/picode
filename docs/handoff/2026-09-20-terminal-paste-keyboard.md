# 2026-09-20 — feat/terminal-paste-keyboard: keyboard paste re-reads clipboard, menu stages files
Shipped: keydown paste re-reads the clipboard via `readPasteClipboard`
(files+text) and re-dispatches an equivalent paste — the preventDefault
that stopped xterm's ^V also stopped the files-carrying event, so images
pasted nothing. PiCode menu Paste stages files via `handlers.pasteFiles`.
Removed the now-dead `termPasteClaim.js` + export in the same cutover.
Verified: `make ci-scoped` PASS; `make web` ok; node incl. new
`readPasteClipboard` cases; scratch `paste-kb` live with a stubbed
clipboard: synthetic Ctrl+V keydown with image+text → routed to the door
(toast on a plain shell, zero stray text); read-denied → text lands via
`term.paste`; overlayAudit ok.
Blind spot: bar-open on a live CLI pane still needs an authenticated CLI;
Firefox without the ClipboardEvent constructor falls back to text (files
then only stage through the menu row).
visual-review: PASS (paste-kb-e2e.png read; card 5/5; audit ok)
Not done: none. Mobile flows untouched.
Merge: fast-forward ready.
