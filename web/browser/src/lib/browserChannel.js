// The daemon's work-browser command channel (ADR-0132). The desktop shell's
// page opens one stream; the daemon pushes one command per line; this module
// runs it against the work-browser tab the human is looking at, through the
// Tauri bridge, and posts the result back.
//
// Desktop-only by nature: without the bridge there is no page to drive, so a
// browser window never opens the stream. The policy decision (which method an
// agent's tier reaches) is made by the daemon and re-checked by the shell's
// Rust catalog — this module only routes the method to the right bridge call.

import { tabWebId } from "./routes.js";

const BRIDGE =
  typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

// EVENTS_VERB is not a CDP method: it asks the shell for the tab's recorded
// event ring (read-tier by construction) and never reaches the page.
export const EVENTS_VERB = "shell.events";

export function bridgeAvailable() {
  return !!BRIDGE;
}

// createBrowserChannel takes every dependency as an argument so its rows are
// testable without a desktop shell. It returns { close }.
export function createBrowserChannel({
  activeTabId,
  invoke,
  source,
  post,
  onError = (message, err) => console.error(message, err),
}) {
  const run = async (cmd) => {
    const id = tabWebId(activeTabId());
    if (!id) return { error: "no work-browser tab is open in the desktop app" };
    try {
      if (cmd.method === EVENTS_VERB) {
        const output = await invoke("btab_cdp_events", { id, since: cmd.params?.since });
        return { output };
      }
      const output = await invoke("btab_cdp_call", {
        id,
        method: cmd.method,
        paramsJson: JSON.stringify(cmd.params ?? {}),
        tier: cmd.tier ?? "read",
      });
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
export function openBrowserChannel(activeTabId) {
  if (!BRIDGE) return () => {};
  const channel = createBrowserChannel({
    activeTabId,
    invoke: BRIDGE,
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