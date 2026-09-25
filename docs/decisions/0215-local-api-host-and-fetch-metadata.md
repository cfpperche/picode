# ADR-0215: The local API refuses cross-site reads and names its hosts

- **Status**: accepted (amends ADR-0049's Host and Origin gate)
- **Date**: 2026-09-24
- **Boundary**: security model — who can reach `/api/*` and `/ws/*` on the daemon from a browser, and which `Host` names the gate admits.

## Context

ADR-0049 put a Host and Origin gate in front of `/api/` and `/ws/`. A
2026-09-24 audit of the gate, prompted by the desktop shell consuming the
same HTTP API, found two gaps. Both were read from the code, and neither
was seen exploited.

1. **Reads were not checked for their site.** Origin and `Sec-Fetch-Site`
   were checked only on mutating methods, on WebSocket upgrades and on
   `/api/events`. A `GET` from another site carries no cookie
   (`SameSite=Strict`), so on a loopback address in mode `remote` the
   gate took it for a new local browser. It either minted a session or
   rotated the secret of the newest idle one (ADR-0049 amendment). The
   response stayed opaque, since there is no CORS, but the request still
   had an effect. Any tab could trigger it with an `<img>`, and so could a
   previewed HTML file on `<label>.localhost`, which the file-preview design
   measured as `cross-site`.
2. **The Host allowlist was wider than any real setup.** It accepted any
   `*.local`, any `*.ts.net`, and the machine's hostname followed by any
   domain, so `box.attacker.example` passed. Over HTTPS a rebinding page
   also fails the certificate check, because the cert does not cover the
   attacker's name. In `PICODE_INSECURE=1` mode, though, the allowlist was
   the only defense.

The owner's real names were measured before narrowing the list. The
hostname is `DESKTOP-BGG95NA`. The Tailscale name is
`desktop-bgg95na-1.tail057039.ts.net`: Tailscale appends a number to a
machine name that is already taken, so an exact hostname rule would have
locked the owner out. The mkcert certificate's SANs already list every
name the owner set up.

## Decision

- **A guarded request with `Sec-Fetch-Site: cross-site` is refused (403)
  whatever its method**, except for the exempt routes. Of those, only
  `GET /api/health` is read cross-site on purpose: it carries CORS, and
  the certificate trust page on `:8470` polls it.
- **A loopback session is minted only for a first-party request**:
  `Sec-Fetch-Site` absent (curl, scripts), `same-origin` or `none` (a typed
  address). A `same-site` page, such as another dev server on
  `localhost:3000`, gets 401. It can still use an existing cookie, which
  the browser sends same-site, but it is never handed a new one.
- **The Host allowlist** accepts:
  - IP literals;
  - `localhost` and `*.localhost`;
  - `picode.local`;
  - the hostname, bare or as its first label (a Tailscale twin `name-N`
    counts) under one private suffix: `local`, `lan`, `home`,
    `home.arpa`, `localdomain`, `internal`, or `<one tailnet label>.ts.net`;
  - every DNS name, wildcard included, covered by the certificates the
    daemon serves (`tlsutil.CertNames`, re-read when the file changes);
  - the public URL's host.

  Any other `*.local` or `*.ts.net`, and the hostname under any other
  domain, are refused.

## Consequences

- **Easier**: the gate's rule is now "a name you issued a certificate for,
  or your machine's name on your own network". It can be read against
  `openssl x509 -ext subjectAltName`, and every row is a test
  (`TestDecisionTable`, `TestHostAllowed`).
- **Harder**: someone who reaches the daemon by a name that is neither a
  certificate SAN nor the hostname on a private suffix now gets 403. Two
  examples are a CNAME alias on the LAN, or the self-signed certificate
  (which covers only `localhost` and `picode.local`) used through a
  Tailscale name the hostname rule does not match. The fix for them is to
  set that address as the public URL (Preferences → Server), which the
  gate already admits.
- **If we're wrong** about a legitimate cross-site reader, it gets 403 with
  "cross-site request refused". The one known reader (`/api/health`) is
  exempt, and the audit found no other.
- **Not changed**: the WebSocket upgraders keep `CheckOrigin: true`,
  because both routes (`/ws/term`, `/ws/agent`) are under `/ws/`, which is
  guarded, and `Wrap` checks the Origin of every upgrade. `/pair` keeps
  the check decided on 2026-09-22 (cross-site submissions only).

## Alternatives considered

- **Exact names only** (hostname, public URL). This lost because it would
  have refused the owner's own Tailscale name (`…-1`).
- **Refusing only the loopback minting, not cross-site reads.** It closed
  the side effect but left the gate trusting reads by default. Fetch
  Metadata's resource-isolation policy (refuse `cross-site` unless the
  route opts in) costs nothing here, because no route is read cross-site
  except `/api/health`.
- **A configurable allowlist setting.** This lost for now: the certificate
  already is the owner's declaration of names, and a second list would
  drift from it.
