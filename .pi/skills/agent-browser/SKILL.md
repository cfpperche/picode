---
name: agent-browser
description: PiCode-only browser QA. Drive Chromium via the native agent_browser tool (npm:pi-agent-browser-native). Use when opening PiCode itself, clicking the UI, screenshots, overlay audit, or end-to-end visual checks. Generic CLI ceremony lives in the package — this skill is the product map.
---

# PiCode browser QA

Use the native `agent_browser` tool (not `bash agent-browser`). Load CLI patterns only if needed:

```
agent_browser  args: ["skills", "get", "core"]
```

## Before you say a UI task is done

1. Open the live app (port is **not** always 8445):
   ```
   cat ~/.picode/server.json
   ```
   HTTPS needs `AGENT_BROWSER_IGNORE_HTTPS_ERRORS=1` (or `PICODE_INSECURE=1` for http).
2. Console sweep: `eval` `document.querySelectorAll` is not enough — also
   `eval` `JSON.stringify(window.__picodeOverlayAudit())`. `ok: false` is FAIL.
3. Screenshot + `read` the PNG. Eval alone is not a visual verdict.
4. Clickability: a real click on every primary control you touched. CSS
   `display` beats `[hidden]` — after close, assert
   `getComputedStyle(el).display === "none"`.
5. Open/close cycles stay closed. State survives reload.

## Rebuild or you are testing the old binary

UI is `go:embed` of `internal/web/public/`. Edit + vite build + `go build`
before reload. Then `agent_browser` `eval` `location.reload()`.

## Session hygiene

Sessions survive aborted tasks. Start messy work with `close`. If `open`
lands on `about:blank`, open again once.

Pair with `/skill:visual-review` for the visual-card. This skill acts;
visual-review judges pixels.
