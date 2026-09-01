# Chrome extension

Send the tab you are looking at to a PiCode agent — URL, title, selected
text, optional screenshot. The agent is the one you already created. This
is not a second chat, and it does not replace the isolated browser the
agent uses for its own work.

## Install

1. In Chrome open `chrome://extensions`, turn on **Developer mode**,
   **Load unpacked**, and pick the `ext/` folder of this repo.
2. Register the native host so the extension can reach PiCode:

```bash
picode extension-install
```

Chrome on Windows with PiCode inside WSL — the tray binary cannot be the
host (it is a GUI program). `make desktop` builds a console sibling,
`picode-nmh.exe`. Then:

```text
picode-desktop extension-install
```

3. Pin PiCode on the toolbar. Click it to open the side panel, or
   right-click a page → **Send to PiCode**.

`picode extension-uninstall` (or `picode-desktop extension-uninstall`)
removes the host registration. The unpacked extension you remove in
`chrome://extensions`.

## Use

Pick an agent, type an optional message, Send. A stopped agent starts
first (**Start and send**). An agent that is already working queues a
follow-up. An agent running in the terminal is not switched from here —
open PiCode instead.

`chrome://` pages, the Web Store, and local files cannot be sent.

## Requirements

PiCode must be running on this machine. The extension talks to it through
the host registered above, not through a pasted URL.
