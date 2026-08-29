import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms } from "../lib/terms.js";
import { wsURL } from "../lib/api.js";
import { closeTerm } from "./TerminalDock.jsx";
import "@xterm/xterm/css/xterm.css";

function shellKey(agentId) {
  return "sh:" + agentId;
}

function termTheme() {
  const s = getComputedStyle(document.documentElement);
  const v = (n, fb) => (s.getPropertyValue(n) || "").trim() || fb;
  return {
    background: v("--bg-base", "#0e0e11"),
    foreground: v("--text-primary", "#ececf1"),
    cursor: v("--accent", "#7c8cf8"),
    selectionBackground: v("--accent-soft", "#33467c"),
  };
}

export function closeShellTerm(agentId) {
  closeTerm(shellKey(agentId));
}

export default function ShellTerm({ agentId, session, active }) {
  const hostRef = useRef(null);

  useEffect(() => {
    if (!agentId || !session || !hostRef.current) return undefined;
    const id = shellKey(agentId);
    closeTerm(id);
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    hostRef.current.appendChild(paneEl);
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 12.5,
      fontFamily: 'ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace',
      theme: termTheme(),
      scrollback: 10000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);
    const entry = { term, fit, paneEl, sock: null, onWinResize: null, closedByUser: false };
    const sock = new WebSocket(wsURL("/ws/term?session=" + encodeURIComponent(session)));
    sock.binaryType = "arraybuffer";
    entry.sock = sock;
    sock.onopen = () => {
      term.reset();
      const sendResize = () => {
        if (sock.readyState === WebSocket.OPEN) {
          sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        }
      };
      fit.fit();
      sendResize();
      entry.onWinResize = () => { if (paneEl.isConnected) { fit.fit(); sendResize(); } };
      window.addEventListener("resize", entry.onWinResize);
      term.onData((data) => {
        if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(data));
      });
      term.onResize(() => {
        if (sock.readyState === WebSocket.OPEN) {
          sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        }
      });
      if (active) term.focus();
    };
    sock.onmessage = (ev) => {
      if (typeof ev.data === "string") return;
      term.write(new Uint8Array(ev.data));
    };
    sock.onclose = () => {
      if (entry.onWinResize) window.removeEventListener("resize", entry.onWinResize);
      if (!entry.closedByUser) term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
    };
    terms.set(id, entry);
    return () => { closeTerm(id); };
  }, [agentId, session]);

  useEffect(() => {
    if (!active) return;
    const entry = terms.get(shellKey(agentId));
    if (entry && entry.fit) {
      requestAnimationFrame(() => {
        entry.fit.fit();
        entry.term.focus();
      });
    }
  }, [active, agentId]);

  return <div className="file-shell" ref={hostRef} />;
}
