---
description: Install web apps — Excalidraw, GitHub, WhatsApp Web — and use them inside PiCode, one login per app.
---

# Web apps

The **Apps** tab can install any web app as a tile. Click the tile and the
app opens in its own tab inside PiCode, with its own login and its own
storage. WhatsApp Web, GitHub, Figma, Excalidraw — anything you use in a
browser tab can live beside your agents.

- **Where:** the **Apps** tab in the sidebar → **+**.
- **Requires:** PiCode Desktop. In a plain browser the tile explains it and
  stops.
- **Not this:** not a bookmark, and not your browser's profile — the app's
  logins live inside PiCode's own storage, separate from the browser you
  read this in.

## Install one

1. **Apps** → **+** at the top of the tab.
2. Type the address — `excalidraw.com`, the `https://` is optional — and
   **Continue**. PiCode checks the site answers and reads its name, icon and
   colors from the page. A site that does not answer is not installed.
3. Confirm the name → **Add app**. The tile lands on the grid, next to the
   built-in apps.

Click the tile and the app opens in its own tab. Web apps that declare
themselves installable open **app-style**: no address bar, the page fills
the tab — what you installed is all you see. <kbd>Ctrl</kbd>+<kbd>F</kbd>
finds in the page, <kbd>F5</kbd> reloads, right-click is the page's own.

## More than one account

The tile menu (**⋯**, top corner of the tile) → **Add another account**.
PiCode creates a second, independent copy of the same app: its own login,
its own data, side by side with the first. Rename the tile to tell them
apart.

Trying to install an address you already installed offers the same choice —
**Add as new account** — instead of refusing.

## Keep the tile truthful

Sites change their names and icons. **⋯ → Refresh** re-checks the site and
updates the tile — the name you gave it and its address stay yours.

## Storage, in plain terms

Each installed app keeps its logins and data in **its own drawer** inside
PiCode Desktop:

- Logins survive restarts of PiCode, the browser and the computer.
- **Settings → Browser → Clear browsing data** clears the *work browser*
  only. Your installed apps stay signed in.
- Two accounts of the same service work side by side.

Removing a tile (⋯ → **Remove**) closes its tab and deletes that drawer —
the app's storage goes with it, and nothing else is touched.

## What it is not

- Not an OS install: the app gets no Start-menu entry and no taskbar icon
  of its own — it lives in a PiCode tab, inside the shell.
- Not a separate set of browser settings: passwords, downloads and the
  clear-data policies are managed in **Settings → Browser**.
- Not a security sandbox against the site: it is the real site, running
  with your real login. Treat it like the tab you would have opened in
  your browser.
