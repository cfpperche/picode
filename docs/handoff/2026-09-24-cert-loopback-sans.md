# 2026-09-24 — feat/cert-loopback-sans: certificates cover 127.0.0.1 and ::1

Shipped (1f48f8f90): `setup-cert.sh`'s SAN list adds `127.0.0.1`/`::1` to
`localhost`/`picode.local` (it used to skip loopback IPs). Provision's cert step
flags a loopback-less mkcert cert and reissues from the same CA with the union of
its current SANs and `tlsutil.LocalNames`, keeping Tailscale/LAN names.
`term_state.go` explains why it still dials `localhost` for old certs.

Verified: 3 new provision tests (reissue keeps names + adds loopback; no-mkcert and
already-covering certs left alone); a real `mkcert` issuing localhost/picode.local
/127.0.0.1/::1 into a scratch dir, checked via `openssl x509 -ext subjectAltName`;
`bash -n` on the script; `make close` full matrix PASS.

Not done: owner's production cert still lacks 127.0.0.1/::1; reissuing (`picode
provision`) is the owner's call, recorded in `docs/handoff/open/windows-wsl.md`. Not deployed.

Decisions (plan item 4): (a) OAuth's "stuck callback port forever" is already
solved by `internal/oauth`'s 15-min `loopbackTimeout`; naming a foreign holder
stays a debt in `docs/handoff/open/agent-cli-credentials.md`, reviewed 2026-09-23.
(b) declined a desktop-shell token — WebView2 can't attach a bearer to
navigations, so `/desktop/` uses same-origin loopback pairing (ADR-0215; shell
daemon calls are the exempt probe). (c) TCP stays — browser/phone/remote need it.

Context: item 4 of 4 in the desktop↔daemon hardening plan; items 1 (waiting
screen), 2 (ADR-0215 gate), 3 (ADR-0216 handshake) landed earlier today.
