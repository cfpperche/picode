import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms } from "../lib/terms.js";
import { wireTermWheel } from "../lib/termWheel.js";
import { scheduleTermFit, wireTermFit } from "../lib/termFit.js";
import { wsURL } from "../lib/api.js";
import { closeTerm } from "./TerminalDock.jsx";
import { xtermOptions, applyXtermOptions } from "../lib/termTheme.js";
import "@xterm/xterm/css/xterm.css";

function shellKey(agentId) {
  return "sh:" + agentId;
}

export function closeShellTerm(agentId) {
  closeTerm(shellKey(agentId));
}

export default function ShellTerm({ agentId, session, active }) {
  const hostRef = useRef(null);

  useEffect(() => {
    if (!agentId || !session || !hostRef.current) return undefined;
    const id = shellKey(agentId);
    if (terms.has(id)) {
      const entry = terms.get(id);
      const live = entry.sock && entry.sock.readyState === WebSocket.OPEN;
      if (live) {
        if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
        entry.paneEl.classList.add("active");
        scheduleTermFit(entry);
        if (active && entry.term) entry.term.focus();
        return undefined;
      }
      closeTerm(id);
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    hostRef.current.appendChild(paneEl);
    const term = new Terminal(xtermOptions());
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);
    const entry = { term, fit, paneEl, sock: null, closedByUser: false };
    wireTermWheel(term, (bytes) => {
      if (entry.sock && entry.sock.readyState === WebSocket.OPEN) entry.sock.send(bytes);
    }, paneEl);
    wireTermFit(entry);
    const sock = new WebSocket(wsURL("/ws/term?session=" + encodeURIComponent(session)));
    sock.binaryType = "arraybuffer";
    entry.sock = sock;
    sock.onopen = () => {
      scheduleTermFit(entry);
      term.onData((data) => {
        if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(data));
      });
      term.onResize(() => {
        if (sock.readyState === WebSocket.OPEN && term.cols > 1 && term.rows > 1) {
          sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        }
      });
      if (active) term.focus();
    };
    sock.onmessage = (ev) => {
      if (typeof ev.data === "string") {
        try {
          const msg = JSON.parse(ev.data);
          if (msg.type === "error") term.writeln("\r\n\x1b[31m" + msg.message + "\x1b[0m");
        } catch { /* ignore */ }
        return;
      }
      term.write(new Uint8Array(ev.data));
    };
    sock.onclose = () => {
      if (entry.unwireFit) entry.unwireFit();
      if (!entry.closedByUser) term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
      if (terms.get(id) === entry) terms.delete(id);
    };
    terms.set(id, entry);
    return undefined;
  }, [agentId, session, active]);

  useEffect(() => {
    function apply() {
      const entry = terms.get(shellKey(agentId));
      if (!entry || !entry.term) return;
      applyXtermOptions(entry.term);
      scheduleTermFit(entry);
    }
    window.addEventListener("picode-term-theme", apply);
    return () => window.removeEventListener("picode-term-theme", apply);
  }, [agentId]);

  return <div className="file-shell" ref={hostRef} />;
}
