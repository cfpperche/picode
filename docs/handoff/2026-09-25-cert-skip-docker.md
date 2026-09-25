# 2026-09-25 — feat/cert-skip-docker: certificates skip Docker bridges

Shipped (e1391575d): `tlsutil.LocalNames` skips interfaces by name (loopback flag, `docker0`, `br-*`, `veth*`), dropping 14 Docker bridge IPs and WSL's `10.255.255.254` from the cert. `setup-cert.sh` applies the same name-based rule via `ip -o -4 addr` instead of the 172.16/12 range (a range rule would drop a real 172.x LAN). Provision's reissue keeps the old cert's DNS names but recomputes IPs, so stale IPs are no longer carried forward.

Verified: `TestSkipInterface`, `TestLocalNamesKeepsLoopbackAndDropsBridges`, updated reissue test (old `203.0.113.77` not carried); Go `LocalNames` and the script's loop both yield `127.0.0.1 ::1 192.168.15.28 100.103.58.20 100.87.149.83` on this machine; `bash -n`; `make close` full matrix PASS.

Not done: the live cert keeps its extra IPs until its next reissue (valid to 2028; harmless — the Host gate reads DNS names only). Not deployed — this only matters at issuance.

Context: follow-up the owner approved after the 2026-09-25 `picode provision` reissue.
