# 2026-09-24 — feat/local-api-gate: ADR-0215 narrows the local API's Host and cross-site gate

Shipped (0ba53b611, 641c5f878): ADR-0215 amends ADR-0049. Cross-site reads are refused (403) on
every guarded route except `GET /api/health`; a loopback session is minted only for a first-party
request (`Sec-Fetch-Site` absent, `same-origin` or `none`), and plain HTTP (`PICODE_INSECURE=1`)
mints one only on a loopback Host name, since a LAN peer can answer mDNS for `*.local`. The Host
allowlist narrows to: IP literals, `localhost`/`*.localhost`, `picode.local`, the hostname (bare or
first label) on a private suffix including a Tailscale twin `name-N`, every name in the daemon's
own certificates (`tlsutil.CertNames`, re-read when the file changes), and the public URL's host
with a `PICODE_PUBLIC_URL` fallback for gateway members.

Verified: go tests (`TestDecisionTable` rows, `TestHostAllowed` against the owner's real names
`DESKTOP-BGG95NA` / `desktop-bgg95na-1.tail057039.ts.net`, `TestHostAllowedWithAFullHostname`,
`TestCertNamesFollowsTheFileOnDisk`); a live scratch instance's 12-row curl matrix matched the
decision table (run before the review fixes); a real headless browser still auto-pairs on
loopback. Adversarial review found no bypass and 3 should-fix items, all folded in (gateway env
public URL, FQDN hostname, plain-HTTP mDNS-rebinding pairing). `make close` PASS.

Not done: not deployed. Browsers without fetch metadata (old Safari/Firefox) keep the old
behaviour. The plain-HTTP pairing fix and the gateway fallback are unit-tested only, not exercised
against a live gateway.

Context: item 2 of 4 hardening the desktop↔daemon API (1: waiting screen, landed; 3: version
handshake; 4: transport — next). No UI change, so no visual review.
