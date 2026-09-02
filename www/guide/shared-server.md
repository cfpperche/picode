# Share one server

Several people, one Linux box on your tailnet. Each person gets their own
PiCode — their own agents, files, credentials and shell, as their own
Linux user. A front door on the tailnet knows who is knocking (from
Tailscale itself) and sends them to theirs.

## For the admin

On the box, once (Tailscale running, MagicDNS and HTTPS certificates
enabled for the tailnet):

```sh
sudo picode gateway install
```

That copies the binary to `/usr/local/bin/picode`, writes
`/etc/picode/gateway.json` with the box's tailnet name, and starts
`picode-gateway.service` on port 443 with a certificate from Tailscale.

For each person:

```sh
sudo picode provision --user alice --shared          # account, environment, her daemon
sudo picode users add alice@example.com alice        # her Tailscale login → her account
```

`--shared` creates the account if needed, lets her services run without
a login, writes her daemon's environment (loopback only, everyone pairs,
public URL = the box's name), and runs her own `picode provision` as
her — which installs and starts her unit. `users add` is read on every
request; nothing restarts. The login is what `tailscale status` shows
for her device (an email, or `name@github`).

Check on things with `sudo picode gateway status` (certificate, whois
self-test, members and whether each daemon answers) and `sudo picode
provision --user alice --shared --dry-run`.

To remove someone: `sudo picode users remove alice@example.com`. Her
daemon keeps her data; stop it with `sudo runuser -u alice -- picode
uninstall` (add `--purge` to delete `~alice/.picode`).

## For a member

Open `https://<box name>` — for example
`https://box.tail1234.ts.net` — from any device on the tailnet. The
first time on each device you see **Pair this iPhone** (or Mac, or
Windows): one tap. That is all; there is no password, because the
tailnet already knows who you are. Your devices show up under
**Devices** in your own PiCode, and **Forget** works as on a laptop.

## What is shared and what is not

- Shared: the machine — CPU, memory, disk, its network. `pi` and tmux
  are installed once for everyone.
- Not shared: anything PiCode knows about. Each person's daemon has its
  own database, agents, terminals, provider credentials, install token,
  paired devices and inbox. They cannot reach each other's daemon: it
  listens on loopback only and demands a pairing that only the gateway
  can start for the right person.
- One Tailscale login maps to one Linux user. A login that is not in
  the map gets "Not on this server", with the command the admin runs.

## Limits

- Everything in [On a server](./remote-server) still applies: reveal,
  the folder picker, llama.cpp and `gh` act on the box; provider sign-in
  callbacks need a port forward or a device-code flow.
- The gateway trusts the tailnet's identity. A device shared between two
  people is one login to Tailscale, and so one PiCode.
