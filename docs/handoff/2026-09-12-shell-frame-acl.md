# 2026-09-12 — feat/shell-frame-acl: the dead buttons were the ACL

Root cause of the injected frame's dead buttons: Tauri's ACL only honors
capabilities for LOCAL pages unless the capability lists the remote
origin. The main window loads https://localhost:8445 — every window.*
call from it was refused, silently (the handlers swallowed errors, which
is how the round was missed). The daemon origin (localhost + loopback
forms) is now trusted in the capability, with the reason in its
description. Drag uses the same IPC, so it was dead too; both should
come alive together — that is also the discriminating test.

Also: the fallback cluster surfaces errors (console + button tooltip)
instead of swallowing them, gains a backdrop pill so it reads as an
overlay next to the app's own top-right widgets, and devtools ship
enabled in release for this class of diagnosis.

Plan agreed with the owner (discussion recorded in-session): the served
UI grows a frame mode (controls integrated in its own top chrome, drag
zones marked, reserved top-right slot); the shell keeps the injected
frame only until that bundle is deployed, keyed off a
documentElement.dataset.frame handshake so the fallback retires itself.
Then the injection of the cluster goes away. A short ADR amending 0120
(window-frame contract) is queued for that session.

Debts: owner-visual of drag + buttons pending (exe installed, PID
confirmed); devtools-on-release is a temporary comfort — revisit when
the frame contract lands.

Merge: fast-forward ready.
