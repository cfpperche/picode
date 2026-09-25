# 2026-09-25 — wrapper-loop

A production tmux guard process spun at ~95% CPU for 30 min: its PATH held a
scratch instance's bin dir before `~/.picode/bin`, and each guard skipped only
its own dir, so the two exec'd each other forever (same pid, orphaned to
systemd --user; it had exited by the time the owner OK'd killing it).
`wrapperFindReal`, `guardFindReal` and the open-URL wrapper now skip any
`# PiCode` wrapper (`picode_wrapper`, shell builtins). The new test loops for
10 s on the old code and passes on the new.

## Next up

- Wrappers on disk update when PiCode rewrites them (restart/switch); the running production guard keeps the old loop until then.
