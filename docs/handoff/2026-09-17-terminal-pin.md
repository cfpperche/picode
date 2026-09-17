# 2026-09-17 — feat/terminal-pin: Continue in… for muse/agy terminals
Owner reported from screenshots that muse/agy terminal sidebar menus lack
Continue in… while all other CLIs have it. Root cause: the menu needs
term.lastSession and LastSession was only ever pinned from a native runtime
entry, which wrapper-less CLIs (muse, agy) never register.
Fix: at stop/restart time, when no runtime pin fired, pin the latest
vendor-store session for that folder past the terminal's creation (same
Latest() recovery heuristic + same grok/hermes exclusion inside
pinTerminalLastSession). This also unlocks resume (both CLIs carry
ResumeArgs: muse --resume uuid, agy --conversation id). A session predating
the terminal never pins (decision table covers both).
Verified: unit tests with fixture muse/agy stores via MuseTestDB/AgyTestDB
overrides (pin lands with resume args; stale session skipped); scratch
end-to-end (real stop on fixture sessions pins full identity); desktop
screenshot read showing Continue in… in the stopped muse terminal menu.
Testing footnote, honestly told: fixture timestamps must postdate terminal
creation because creation stamps nanoseconds while readers resolve
microseconds — same-instant fixtures truncate older and trip the guard;
fixtures use +5min. No UI code changed (menu renders from the pin it
already knew). Deploy NOT done (owner's call).
uiux-review: PASS (no JSX changed; existing menu item, existing primitives)
visual-review: PASS (var/screenshots/muse-continue-in.png read; card 5/5)
