# Security and pairing

PiCode gives whoever is signed in a shell on the machine that runs it,
so access is per **device**, not per network.

## Who can reach it

| Setting (Preferences → Server → Who must pair) | Browsers on the PiCode machine | Every other device |
|---|---|---|
| **This machine is trusted** (default) | work without doing anything | need a pairing link |
| **Every device pairs** | need a pairing link | need a pairing link |
| **Off** | no pairing | no pairing — only behind a proxy you trust |

Whatever the setting, a page from another site cannot drive your PiCode:
requests from foreign origins and unknown host names are refused.

## Pair a phone or another computer

1. On a device that already works, open **Devices** (user menu) → **Pair
   a device**, or click the phone icon in the sidebar (the QR carries the
   pairing link).
2. Scan the code or open the link on the other device. It works once and
   expires in ten minutes.
3. The device appears under **Devices** with an online dot while it is connected. **Forget** revokes it.

If no device is paired yet, run `picode pair` on the machine that runs
PiCode: it prints a link.

## Scripts, the Chrome extension and pi tools

They read the install token from `~/.picode/token` and send it as
`Authorization: Bearer …`. Rotate it with `picode token rotate` or from
Preferences → Server, where the pairing rule also lives. Automation webhooks keep their own secret and never
need the token.

## What a pairing knows

The device's browser label, its IP and when it was last seen. Sessions
expire after ninety days without use.
