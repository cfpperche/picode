# 2026-09-13 — feat/browser-grants: a grant buys something (slice 4, increment 1)

Shipped: the act vocabulary, the destination rule, and params on the wire.
`evaluate` (`Runtime.evaluate`) and `navigate` (`Page.navigate`) are the first
act verbs, so the `act` tier is no longer a promise nobody keeps. The tool
still names a verb, never a CDP method. `browser.AllowsOrigin` writes the
domain rule down (http/https only; exact host; `*.example.com` or
`.example.com` for subdomains; port ignored) and the route checks the one verb
with a destination against the grant before the command leaves — the shell's
navigation gate will be the other half of the same rule. Params travel
tool → daemon → shell now (`Command.Params` was already in the channel; the
tool request had no field), which also fixed `events since`: it was sent at the
top level, where the channel never read it, so every poll replayed the ring.

Verified: `go test ./internal/browser/` (verb tiers, closed vocabulary, the
12-row origin table, `AllowsVerb`'s tier+origin rows) and
`./internal/server -run TestBrowser` (act refused without a grant; navigate
outside the grant refused naming the origin; non-object params 400; a granted
navigate reaching the shell with its params and domains; a read verb still
carrying no params). 8 rows of `packages/pi-browser/test`, incl. the five-verb
union check.
visual-review: n/a — no UI surface changed. The editor is the next increment.
Not done / debts: the grants editor (Settings ▸ Browser: tier × domains; the
write path is `browser.Save`, the read path does not exist yet — no list
endpoint); the shell-side navigation gate; the shell must mirror
`AllowsOrigin` or daemon and shell can disagree; no Windows runtime
acceptance of any act verb.

## Debts

- Two writers of one rule: `AllowsOrigin` (Go) and the gate to come (Rust); keep them in step.
- Act verbs have never run on Windows: only compilation and Go/route tests exist.
