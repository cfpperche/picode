# 2026-09-25 — wrapper-loop

A production tmux guard process spun at ~95% CPU for 30 min: its PATH held a
scratch instance's bin dir before `~/.picode/bin`, and each guard skipped only
its own dir, so the two exec'd each other forever (same pid, orphaned to
systemd --user; it had exited by the time the owner OK'd killing it).
`wrapperFindReal`, `guardFindReal` and the open-URL wrapper now skip any
`# PiCode` wrapper (`picode_wrapper`, shell builtins). The new test loops for
10 s on the old code and passes on the new.

- The guard was only written when missing, so a deploy never replaced an old
  one: `ensureTmuxGuard` now rewrites a stale body and runs at boot, like the
  CLI and open-URL wrappers already did. The fix reaches production on the
  next deploy.
