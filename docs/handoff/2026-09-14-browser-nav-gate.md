# 2026-09-14 — browser-nav-gate

Slice 4 increment 4.2: the shell-side navigation gate (the last boundary
half that only lived on the daemon).

## Done

- `picode_shell::origins::gate` — the whole decision table as one pure
  function (no grant → allow; user-initiated → allow; agent-caused →
  `allows_origin`), tested on the same rows as `browser.AllowsOrigin`.
- `btab.rs`: a grants map per webview id; `btab_cdp_call` arms it on
  act/full tier (the envelope already carried `domains`; the UI relay now
  passes it through); `btab_navigate` disarms (user's will);
  `btab_close` drops the entry; `ensure` attaches a `NavigationStarting`
  handler (webview2-com, wry's pattern) that cancels non-user loads
  outside the grant.
- `cargo xwin build` ✓ (1m32s). The crate's tests still only run on
  Windows (`cargo test --lib origins`) — same standing as the cdppolicy
  suite.

## Notes

- Live acceptance needs the shell: after the next deploy + restart, an
  agent with a domain grant on a tab should not be able to script-redirect
  it elsewhere, while the user clicking/navigating stays free.
- Slice 4 remainder is now only the grants editor (Settings ▸ Browser).
