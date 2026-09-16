# 2026-09-16 — domain-any: `*` for any site, and a field that explains itself

Owner asked why the grants cannot be a regex ("mais fácil, não?"). The answer
that landed: a regex means a second grammar in two languages (Go daemon +
Rust navigation gate, which must agree by construction), a grant nobody can
audit at a glance, and anchors the owner would have to remember. The real
uncovered case was exactly one — "any site".

Landed:
- **`*` = any host**, http/https only, in both matchers
  (`internal/browser/domains.go`, `desktop-shell/src/origins.rs`) with rows in
  both test tables. The scheme rule still runs first, so it is the web and
  nothing else.
- **The field explains itself** (`web/browser/src/lib/browserDomains.js`,
  +8 tests): one line per entry saying what it covers — "Any site", "x and its
  subdomains", "Every .com site", "x only" — and, for the entries that match
  nothing (`*example.com`, `*.`, `.`), a warning naming the fix. That was the
  `*` bug: the field accepted it, the matchers compared it literally, nothing
  said a word.
- Fixed the Rust test module's import so `rustc --edition 2021 --test
  src/origins.rs` runs both tables standalone (it referenced `gate` without
  importing it; the standalone harness is how these tests run here).

Verified: `go test ./internal/browser`, `node --test web/browser` (374),
`rustc --test src/origins.rs` (2 passed), `cargo xwin build`, `make close`.
Not run live: the field's hint and a real `*` navigate — owner's click.

## Next up

- Owner: set an agent to Act with `*`, ask it to open something, and check the
  field's hint reads "Any site" while you type it.

## Debts

- The JS hint in `browserDomains.js` is a third reading of the rule (the two
  matchers are the enforcement). It is described and tested as a hint; if the
  matchers ever change, this file is the one that will lie first.
