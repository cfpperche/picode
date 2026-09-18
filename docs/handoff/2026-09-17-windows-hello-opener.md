# 2026-09-17 — windows-hello-opener (v2d)

v2d from the browser scope: the opener row for what Windows keeps.

## What it is (and is not)

A row in Settings ▸ Browser ▸ Password manager, "Windows Hello and
passkeys", whose button opens Windows' own Sign-in options
(`ms-settings:signinoptions`). It is **not** a vault: PiCode still cannot
list, edit or import a single entry, and the existing lede says so.

The row shows **only in the shell** (`shellInvoke(window)`), because a plain
browser has no such screen to open.

## The guard grew one class — and strictly

`btab_open_external` ran `cmd /C start "" <url>` behind an "http/https only"
check. That check also let a target carrying `"` or `&` through, which is the
shape that turns a URL into a second command. The allowlist now lives in
`desktop-shell/src/external.rs` as a pure function with a decision table:

| target | handed over |
|---|---|
| `https://…`, `http://…` | yes |
| `ms-settings:signinoptions`, `ms-settings:passkeys` | yes |
| `ms-settings:` (empty), `ms-settings:Sign-In`, `ms-settings:x?y` | no |
| `https://x/" & calc`, spaces, quotes, backticks | no |
| `file:`, `javascript:`, `data:`, `cmd:`, `ms-settings-evil:` | no |

## Verified — and what is not

- `rustc --edition 2021 --test src/external.rs` → 3 tests, every row.
- `make desktop-shell` (cargo xwin, MSVC) → **0**, so `btab.rs`/`main.rs`
  compile against the real Tauri deps (2 warnings, both pre-existing).
- JS: 402 passing, including the exact payload
  `btab_open_external {url: "ms-settings:signinoptions"}` and the refusing
  shell being reported rather than swallowed.
- **Not verified**: the click on Windows. The OS screen is the owner's first
  run (debts, this topic).

## Honest note

I made these edits **in the root checkout** first — the exact failure the
contract names. Caught before any commit: the files moved to
`.worktrees/v2d` (feat/v2d), the root is clean again, and the work happened
there. The commit never happened on main: the hook would have refused it.
