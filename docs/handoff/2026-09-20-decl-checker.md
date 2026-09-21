# 2026-09-20 — decl-checker

Static gate for the blank-window bug class (main 4f1a68a7).

## Done

- scripts/declaration-order.mjs: per function body, flags reads of
  let/const/class bindings declared later in the same body when the read
  sits outside any nested function; shadowing (nested re-declaration, loop
  heads, catch params) and hoisted names (function/var/imports) are exempt.
  Conservative by design: unsure means unflagged — false positives would
  break CI and get the checker deleted.
- Parses with @babel/parser already in web/node_modules (ships with
  @vitejs/plugin-react, parses JSX natively) — no new dependency.
- scripts/declaration-order.test.mjs: bad/good fixtures under
  scripts/fixtures/declaration-order/ plus a whole-tree row over
  web/browser/src and web/shared — clean on this branch, 0 skips.
- Proven against the reference: pointed at WebTab.jsx as of 4f1a68a7 it
  flags the deps-array read (76:35); today's tree carries the fix.

## Notes

- No Windows verification: dev tooling only, no runtime behavior change.
- A sibling edits WebTab.jsx concurrently; its version is scanned at land
  time via the same whole-tree row — soundness over coverage there.
