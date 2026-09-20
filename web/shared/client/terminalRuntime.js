import { createSticky } from "../domain/termSticky.js";
import { wireTermWheel, wireTermTouch } from "../domain/termWheel.js";
import { wireTermKeys, termDataFilter } from "../domain/termKeys.js";
import { wireTermClipboard } from "../domain/termClipboard.js";
import { wireTermFit, scheduleTermFit } from "../domain/termFit.js";
import { connectTermSocket } from "./termSocket.js";

// The terminal engine is headless: each app supplies its xterm and chrome.
// Agent, Agent CLI and Canvas hosts never install their own transport/input.
export function wireTerminalRuntime(entry, { url, reservedKey, onClipboardError, onOpen, onDetached }) {
  const { term, paneEl } = entry;
  entry.sticky = createSticky();
  const send = bytes => {
    if (entry.sock?.readyState === WebSocket.OPEN) entry.sock.send(bytes);
  };
  wireTermWheel(term, send);
  entry.unwireTouch = wireTermTouch(paneEl);
  wireTermKeys(term, send, reservedKey);
  wireTermClipboard(term, { onError: onClipboardError });
  wireTermFit(entry);
  term.onData(data => {
    const out = entry.sticky.apply(termDataFilter(data));
    if (out !== "") send(new TextEncoder().encode(out));
  });
  term.onResize(() => {
    if (term.cols > 1 && term.rows > 1) send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
  });
  connectTermSocket(entry, url, {
    onOpen: () => { scheduleTermFit(entry, true); onOpen?.(); },
    onMessage: ev => {
      if (typeof ev.data !== "string") { term.write(new Uint8Array(ev.data)); return; }
      try {
        const msg = JSON.parse(ev.data);
        if (msg.type === "error") term.writeln("\r\n\x1b[31m" + msg.message + "\x1b[0m");
      } catch { /* ignore non-control text */ }
    },
    onState: () => { term.writeln("\r\n\x1b[90m— detached —\x1b[0m"); onDetached?.(); },
    onGiveUp: () => term.writeln("\r\n\x1b[90mSession ended. Reopen the terminal.\x1b[0m"),
  });
}
