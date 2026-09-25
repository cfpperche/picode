# ADR-0216: Out-of-process clients announce their version; the daemon answers 426 when they are too old

- **Status**: accepted
- **Date**: 2026-09-24
- **Boundary**: protocol — a request header every out-of-process API client sends, a 426 answer and two new `/api/version` fields; plus the desktop shell announcing its command protocol to the page it hosts.

## Context

The web UI is embedded in the daemon, so it always matches the API. A
stale browser tab reloads on a changed `bootId`. Three other clients can
drift from the daemon, and none of them had any handshake:

| Client | How it drifts | What breaks |
|---|---|---|
| `picode mcp …` stdio servers | They live as long as the CLI session and survive a deploy that replaces the binary on disk | An API change fails a tool call with a route error the model cannot act on |
| The browser host (`picode-browser-host`, ADR-0043) | Chrome keeps it running across a deploy | Same |
| The Windows shell (`picode-shell`) | It is updated separately from the daemon, by the owner's `make desktop-restart` or by a release install | A new `/desktop/` page calls a command the older shell lacks and gets "not allowed by ACL" |

There is no API version in the URL, and adding one does not fit: the UI
is embedded, so every route has exactly one consumer version apart from
these clients.

## Decision

- **API clients** send `X-PiCode-Client: <kind>/<build>; protocol=<n>`.
  Each kind (`picode-mcp`, `picode-browser-host`, `picode-cli`) sends its
  `version.APIProtocol`, through `version.ClientValue`.
  - When `protocol` is below `version.MinClientProtocol`, the daemon
    answers `/api/*` and `/ws/*` with **426** and an `error` that names
    what to restart. For `picode-mcp` the message is "Restart the agent to
    load the new tools". `mcptool` passes that message to the model word
    for word.
  - Requests without the header, or with a malformed one, are served as
    before. That covers browsers, the UI, curl and third-party scripts.
  - `/api/version` publishes `protocol` and `minClientProtocol`.
- **Bumping the protocol.** `APIProtocol` goes up, and `MinClientProtocol`
  with it, only when an `/api` change makes an older client *wrong*: a
  route removed, or a field whose meaning changed. Additions never bump
  it, because older clients ignore them. Both constants start at 1.
- **The shell** announces itself to the page it hosts:
  `window.__PICODE_SHELL__ = {version, protocol}`, where the protocol is
  `desktop-shell` `SHELL_PROTOCOL` and starts at 1. A shell from before
  this ADR announces nothing and reads as protocol 0.
  - The UI helper `shellSupports(n)` (`web/browser/src/lib/shellVersion.js`)
    lets a feature that needs a newer command hide itself instead of
    failing.
  - The protocol is bumped whenever a command is added or a command's
    contract changes. The helper's header lists what each protocol
    brought.

## Consequences

- **Easier**: after a deploy, a stale MCP server tells the agent (and so
  the human) what to do, instead of failing on a 404. Shell features can
  land before every shell is updated.
- **Harder**: bumping is a discipline, since nothing forces it. A
  protocol that should have been bumped leaves the drift as it was
  before this ADR, but no worse.
- **Not done, and why**: the UI does not tell the human "update PiCode
  Desktop". No update path exists for the shell yet:
  `picode-desktop install` reuses the shell executable already on disk
  (`stageShellExe`), so the notice would carry no action. It is a debt in
  `docs/handoff/open/windows-wsl.md`, waiting for that path.
- **If we're wrong** about a client sending the header, the only cost is
  a 426 on an old build, which is the point.

## Alternatives considered

- **`/api/v1` in the URL.** It lost because the one consumer that matters,
  the embedded UI, cannot drift, and every route would carry the prefix
  for three clients.
- **Comparing builds instead of protocols.** It lost because every deploy
  would refuse every running MCP server, including when nothing they call
  had changed.
- **The shell sending a header on its requests.** It lost because the
  shell's own daemon calls are health probes only. The drift is in the
  other direction, page to shell, so the announcement has to reach the
  page.
