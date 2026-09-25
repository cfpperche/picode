// The daemon's work-browser command channel (ADR-0132). The desktop shell's
// page opens one stream; the daemon pushes one command per line; this module
// runs it and posts the result back.
//
// A session drive (ADR-0172) names its own split: ensureSession opens the
// pane beside that principal and returns the webview id. Anything else still
// reads the tab on screen. The policy decision is made by the daemon and
// re-checked by the shell's Rust catalog — this module only routes.

import { tabWebId } from "./routes.js";
import { createComputerRunner } from "./computerChannel.js";

const BRIDGE =
  typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

// EVENTS_VERB is not a CDP method: it asks the shell for the tab's recorded
// event ring (read-tier by construction) and never reaches the page.
export const EVENTS_VERB = "shell.events";

const SCREENSHOT = "Page.captureScreenshot";

// withTimeout settles with { value } or { timedOut: true }; a rejection
// before the deadline still rejects.
function withTimeout(promise, ms) {
  let timer;
  return Promise.race([
    promise.then((value) => ({ value })),
    new Promise((resolve) => { timer = setTimeout(() => resolve({ timedOut: true }), ms); }),
  ]).finally(() => clearTimeout(timer));
}

export function bridgeAvailable() {
  return !!BRIDGE;
}

// createBrowserChannel takes every dependency as an argument so its rows are
// testable without a desktop shell. It returns { close }.
export function createBrowserChannel({
  activeTabId,
  ensureSession = null,
  invoke,
  source,
  post,
  runComputer = null,
  onError = (message, err) => console.error(message, err),
  hiddenPixelWait = 4000,
  revealSettle = 400,
}) {
  const run = async (cmd) => {
    // ADR-0148: a computer frame is a desktop action, not a CDP method. It
    // never touches a tab; a page without a runner answers instead of
    // pushing it through the browser bridge.
    if (cmd.kind === "computer") {
      if (!runComputer) return { error: "not_connected: this page cannot drive the computer" };
      return runComputer(cmd);
    }
    let id = "";
    let session = null;
    if (cmd.session) {
      if (!ensureSession) return { error: "no work-browser tab is open in the desktop app" };
      const opened = await ensureSession(cmd);
      if (!opened || !opened.id) return { error: opened?.error || "no work-browser tab is open in the desktop app" };
      id = opened.id;
      session = opened;
      if (cmd.method === "shell.open") {
        const url = typeof cmd.params?.url === "string" ? cmd.params.url : "";
        try {
          if (url) await invoke("btab_navigate", { id, url });
        } catch (e) {
          return { error: String(e?.message || e) };
        }
        return { output: { opened: true, url } };
      }
    } else {
      id = tabWebId(activeTabId());
      if (!id) return { error: "no work-browser tab is open in the desktop app" };
    }
    try {
      if (cmd.method === EVENTS_VERB) {
        const output = await invoke("btab_cdp_events", { id, since: cmd.params?.since });
        return { output };
      }
      const call = () => invoke("btab_cdp_call", {
        id,
        method: cmd.method,
        paramsJson: JSON.stringify(cmd.params ?? {}),
        tier: cmd.tier ?? "read",
        domains: Array.isArray(cmd.domains) ? cmd.domains : [],
      });
      // A session split the human is not looking at stays where it is
      // (ADR-0172). A screenshot needs painted pixels, and a hidden
      // WebView2 may not paint: give it a short wait, then bring the split
      // forward and ask once more rather than time the agent out.
      if (cmd.method === SCREENSHOT && session && !session.onScreen && session.reveal) {
        const first = await withTimeout(call(), hiddenPixelWait);
        if (!first.timedOut) return { output: first.value };
        onError("browser screenshot:", new Error("the hidden split did not paint; showing it"));
        session.reveal();
        await new Promise((resolve) => setTimeout(resolve, revealSettle));
      }
      const output = await call();
      return { output };
    } catch (e) {
      return { error: String(e?.message || e) };
    }
  };

  const handle = async (cmd) => {
    if (!cmd || !cmd.id) return;
    const result = await run(cmd);
    try {
      await post({ id: cmd.id, ...result });
    } catch (e) {
      onError("browser result:", e);
    }
  };

  const listener = (ev) => {
    let cmd = null;
    try {
      cmd = JSON.parse(ev.data);
    } catch {
      return; // a frame we cannot read is not a command
    }
    handle(cmd);
  };

  source.addEventListener("command", listener);
  return {
    close() {
      source.removeEventListener("command", listener);
      source.close();
    },
  };
}

// openBrowserChannel returns a close function. The stream reconnects on its own
// (EventSource), so a restarted daemon or a lost line rejoins without a poll.
export function openBrowserChannel(activeTabId, ensureSession) {
  if (!BRIDGE) return () => {};
  const channel = createBrowserChannel({
    activeTabId,
    ensureSession,
    invoke: BRIDGE,
    runComputer: createComputerRunner({ invoke: BRIDGE }),
    source: new EventSource("/api/browser/stream"),
    post: async (result) => {
      const res = await fetch("/api/browser/result", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(result),
      });
      // 404 means the tool call already gave up (a timeout): the shell did its
      // part, and there is nothing to do about it.
      if (!res.ok && res.status !== 404) throw new Error(`result ${res.status}`);
    },
  });
  return () => channel.close();
}