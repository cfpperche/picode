# 2026-09-13 — feat/browser-install: the dev workspace gets the browser tool

Owner asked for `pi-browser` in the PiCode workspace. Installed through the
product's own path (`POST /api/packages`, scope `project`, workspace
`picode-5fd7eb`), which runs `pi install packages/pi-browser -l` and wrote
`.pi/settings.json` — a tracked file, so this branch records the same bytes
rather than leaving the shared tree dirty. The package is now listed beside
pi-checklist, pi-diff and pi-browser-capture, which is how the other dev
packages reach this workspace.

Effect: agents in this workspace load `pi-browser` from their next start, and
a plain `pi` in this directory sees it too. First runtime acceptance of the
tool is the owner's: open the desktop app, open a tab, ask for `snapshot`.
Nothing in the daemon needed a restart (the install is a config write); the
agent must restart to pick it up.

Verified while installing: a real pi session in this workspace (SDK
`createAgentSession`, project packages via `.pi/settings.json`) reports
`browser` in `session.agent.state.tools` — the tool registers. The round trip
is still open: `POST /api/browser/tool` with `snapshot` answered **"the desktop
app is not connected"** (502), which is the honest daemon answer with no shell
on the line. To see page content: run the desktop app, open a work-browser tab,
then ask a restarted agent for `snapshot`.
