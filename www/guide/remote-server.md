# Run PiCode on a server

One PiCode on a Linux box you own, reached from your PC and your phone
over your tailnet. Same binary as the laptop; the difference is that you
tell it where it lives.

## 1. Install from a release

No repo checkout is needed. On the server:

```sh
V=$(curl -fsSL https://api.github.com/repos/cfpperche/picode/releases/latest | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p')
curl -fLO https://github.com/cfpperche/picode/releases/download/v$V/picode-linux-amd64
curl -fLO https://github.com/cfpperche/picode/releases/download/v$V/SHA256SUMS
sha256sum -c --ignore-missing SHA256SUMS      # picode-linux-amd64: OK
chmod +x picode-linux-amd64
./picode-linux-amd64 install --env PICODE_HOST=0.0.0.0
./picode-linux-amd64 provision --dry-run     # the health table
```

`install` copies the binary to `~/.local/bin/picode`, writes the systemd
user unit and starts it. `--env KEY=VALUE` (repeatable) goes into
`~/.config/systemd/user/picode.service.d/env.conf`, which later deploys
and updates leave alone. `provision --dry-run` is the doctor: linger,
systemd, certificate, service, health, `pi` on PATH, Tailscale, and
whether other machines can reach this daemon. Run `provision` without
`--dry-run` to fix what it can (linger and the certificate need root
once; it says so).

`picode update` checks GitHub, verifies the download against the
release's `SHA256SUMS`, and only then restarts on the new binary.

Also needed on the box: `pi` (`npm install -g @earendil-works/pi-coding-agent`),
tmux 3.5+, and `mkcert` if you want a certificate browsers trust without
a warning (`provision` issues it).

## 2. Tell it where it lives

Open the app once from the server itself (or from your PC on the LAN)
and go to **Preferences → Server → Reach this server**:

- **Bind** — *All interfaces* is the default. *Tailnet and this machine*
  binds the Tailscale address plus loopback, so the LAN cannot reach it
  but the box itself (its browser, `picode pair`, scripts) always can.
- **Public URL** — the address every other machine uses:
  `https://box.tailxxxx.ts.net:8445` (the tailnet name PiCode suggests)
  or the tailnet IP. It goes into pairing links, `server.json` and the
  phone drawer. It does not move the listener.

Both are settings in the database, so they survive updates. **Clear**
next to the public URL removes it — links go back to an address of this
machine. That is not a lock: a device that already paired keeps its
session until you **Forget** it on Devices, and the server keeps
listening where Bind says. For the very
first start, before there is a UI, `--env PICODE_HOST=…` does the same
as Bind.

## 3. Pair your devices

On the server's own browser, or any paired device: **Devices → Pair a
device**, or the phone icon. With a public URL set, the QR carries it.
See [Security and pairing](./security).

**With Tailscale, the phone installs nothing.** PiCode asks Tailscale
for a certificate for the box's tailnet name (`box.tailxxxx.ts.net`),
signed by a public authority every phone already trusts, and renews it
by itself. The drawer lists that name first, marked as needing nothing.
Two things have to be true once:

- HTTPS certificates are enabled for your tailnet (Tailscale admin
  console → DNS → HTTPS Certificates).
- `tailscale cert` may run as your user: `sudo tailscale set
  --operator=$USER` once. Until then `picode provision --dry-run` shows
  the `tailnet-cert` row with that hint, and the IP addresses keep
  working with the local certificate (installed once on the phone
  through the QR's trust page).

Without Tailscale, the local certificate serves everything, as on a
laptop.

## 4. The Chrome extension on another PC

On the PC (Linux or WSL), with the extension's native host binary:

```sh
picode extension-install --server https://box.tailxxxx.ts.net:8445 --token <install token> --ca ~/rootCA.pem
```

The token is `picode token` on the server (the file it prints). `--ca` is
the server's mkcert root (`mkcert -CAROOT` on the server, copy
`rootCA.pem`); without it the PC must already trust that CA. This writes
`~/.picode/remote.json`, which the native host reads instead of the local
`server.json`. To go back to a local PiCode, delete that file.

## 5. `pi` on a laptop, inbox on the server

`pi-inbox` posts to `PICODE_URL` when set, with `PICODE_TOKEN` as the
bearer:

```sh
export PICODE_URL=https://box.tailxxxx.ts.net:8445
export PICODE_TOKEN=<install token>
export NODE_EXTRA_CA_CERTS=/path/to/rootCA.pem   # Node must trust the server's CA
```

Agents the server itself runs need nothing: they inherit `PICODE_DATA`.

Several people on one box? See [Share one server](./shared-server).

## What stays on the server

- **Reveal**, the **folder picker**, **llama.cpp** and **`gh`** act on the
  server's filesystem and processes, not your PC's.
- **Provider sign-in** (Anthropic, Codex, MCP auto-auth) opens a callback
  on a loopback port of the machine running `pi` — the server. From a
  remote browser, forward it first (`ssh -L 53692:localhost:53692 box`,
  same for 1455), or use a provider's device-code flow. Once signed in,
  the credential lives on the server.
- The **trust page** (port 8470) is plain HTTP on the LAN/tailnet only.
  A public deployment (Track D) turns it off.
