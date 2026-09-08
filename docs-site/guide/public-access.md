# Open it to the internet

People on any network reach their PiCode at `https://picode.example.com`
with a Google or GitHub login. Same gateway as [Share one server](./shared-server);
this page adds the public door.

## What you need

- A shared box with `picode gateway` installed and at least one member.
- A public name pointing at the box, and a TLS proxy in front: Caddy on
  the box, or a Cloudflare Tunnel. PiCode never issues public
  certificates itself.
- An OAuth app at Google (an "OAuth 2.0 Client ID", type Web) or GitHub
  (Settings → Developer settings → OAuth Apps). The callback URL is
  `https://picode.example.com/-/auth/callback/google` (or `/github`).

## Configure the login

```sh
sudo picode gateway oidc set google <client id> <client secret> --public-url https://picode.example.com
sudo picode gateway oidc set github <client id> <client secret>
sudo systemctl restart picode-gateway
sudo picode gateway status
```

`oidc set` stores the credentials in `/etc/picode/gateway.secret.json`
(root only), turns on a plain listener on `127.0.0.1:8480` for the
proxy, and trusts `X-Forwarded-For` from `127.0.0.1` only. `status`
prints the callback URLs to paste into the provider.

Members are the same list as before. A Google user is their email; a
GitHub user is `<login>@github`:

```sh
sudo picode users add alice@example.com alice
sudo picode users add octocat@github cat
```

## Put a proxy in front

**Caddy** (`/etc/caddy/Caddyfile`): certificates are automatic.

```
picode.example.com {
    reverse_proxy 127.0.0.1:8480
}
```

**Cloudflare Tunnel**: `cloudflared tunnel create picode`, route the
hostname to the tunnel, and in the tunnel config point
`picode.example.com` at `http://127.0.0.1:8480`. Cloudflare terminates
TLS; the connection to the box stays on loopback.

Both keep the gateway off the public interface: only the proxy talks to
it, over loopback, which is why `trustedProxies` is `127.0.0.1/32`.

## What a person sees

1. `https://picode.example.com` → **Sign in** → Google or GitHub.
2. If the account is on the members list: **Pair this iPhone**, one tap.
   Otherwise "Not on this server", with the command the admin runs.
3. Their own PiCode. The sign-in lasts 30 days; `/-/auth/logout` ends
   it early. Tailnet users keep entering on `:443` with no login at all.

## A container per person

For people you do not know, a Linux user is a thin fence: `pi` is a
shell. Put each member's PiCode in a container instead:

```sh
sudo apt install systemd-container debootstrap    # once per box
sudo picode provision --user alice --shared --container
```

Alice gets a root filesystem of her own (a minimal image of the box's
release with `pi`, `tmux`, `git`), her home bound in, a private user
namespace, no capabilities to speak of, and limits (2 CPUs, 4 GB, 512
tasks). She cannot see the host's `/etc`, other homes or the host's
`pi` settings. What she still shares: the kernel and the network stack.
For anything stronger — strangers paying for it — the next step is a VM
per person, which is a different product decision.

`sudo picode provision --user alice --shared --container --remove`
deletes the container and its unit; her account and home stay until
`userdel -r alice`.

## Limits

- Everything from [On a server](./remote-server) still applies.
- Web Push and the phone shell work on the public origin like on the
  tailnet.
- No content-security policy yet on the gateway (the app's inline theme
  script needs a nonce first); the other headers are on.
