# 2026-09-14 — split-hide

Owner-reported: with the split open in one tab, switching tabs left the
globo.com pane painted over the new tab (both screenshots in
/var/... — repro'd on the live desktop).

## Done

- Root cause: the split's `WebTabSurface` mounts only under
  `agentPanes[selectedId]`; a tab switch unmounts it, and the visibility
  effect (`btab_visibility`) only ran on `hidden` changes — unmount had no
  cleanup, so the native view kept its bounds. Fixed with a cleanup that
  parks the view (`visible:false`); remount shows it again, page state
  intact.
- Root checkout was edited by mistake for one minute; restored with
  `git checkout --` before any commit (the patch moved to the worktree).
- Gates: `make web` ✓, 8/8 js tests ✓, `ci-scoped` PASS.

## Notes

- The native hide cannot be exercised in a plain browser (no `__TAURI__`);
  scratch proves no DOM regression only. Live acceptance: owner switches
  tabs after the next deploy — pane must vanish with the old tab and come
  back intact.
