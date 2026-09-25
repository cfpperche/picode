### Security

- **The local API no longer answers other websites' requests.** A page open
  in another tab, or an HTML file shown in the preview pane, could make
  PiCode on this machine create or refresh a sign-in session just by
  loading an image from it. Requests from other sites are now refused, and
  only PiCode's own pages (or a script with no browser at all) are signed in
  automatically on this machine.
- **PiCode answers only to names you actually use.** It used to accept any
  `.local` or Tailscale name, and this computer's name followed by any
  domain. Now it accepts the addresses it always accepted (`localhost`, IP
  addresses, `picode.local`), this computer's name on your home network or
  tailnet (including Tailscale's `name-1` style), and every name on your
  PiCode certificate. If you reach PiCode by another name, set that address
  as the public URL in Preferences → Server.
