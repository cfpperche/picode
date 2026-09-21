# 2026-09-20 — feat/terminal-paste-attach: paste screenshots/files into agent terminals
Shipped: Ctrl+V / Ctrl+Shift+V with files on the clipboard opens the attach
bar seeded (`openTermAttachFiles`, `TermSurface onPasteCapture`); text
pastes keep the native xterm paste; plain shells toast. Pure helpers
`clipboardFiles` + `termHasPromptDoor` in `web/shared/domain/termPrompt.js`.
Docs: `routes.md` prompt door; fragment `terminal-paste-attach.md`.
Verified: `make ci-scoped` PASS; node tests 4/4; `make web` ok; scratch
`paste-attach` live: files-paste → `defaultPrevented` + toast on a plain
shell, text-paste lands in the terminal, `__picodeOverlayAudit` ok.
Blind spot: bar-open on a live CLI pane not run — no authenticated CLI in
scratch; the open path composes the Attach menu's `setTermAttach` shape
with the tested classifier.
visual-review: PASS (paste-shell-toast.png read; card 5/5; audit ok)
Not done: mobile `TermAttachSheet` keeps the paperclip flow (out of scope).
Merge: fast-forward ready.
