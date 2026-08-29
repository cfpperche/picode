import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms } from "../lib/terms.js";
import { wireTermWheel } from "../lib/termWheel.js";
import { wireTermKeys, termDataFilter } from "../lib/termKeys.js";
import { scheduleTermFit, wireTermFit } from "../lib/termFit.js";
import { wsURL } from "../lib/api.js";
import { xtermOptions, applyXtermOptions } from "../lib/termTheme.js";

import "@xterm/xterm/css/xterm.css";

export default function TerminalDock({
  open, agent, workspace,
}) {
  const hostRef = useRef(null);
  const status = useRef({ set: () => {} });

  useEffect(() => {
    if (!open || !agent || agent.mode !== "interactive") return;
    const id = agent.id;
    if (terms.has(id)) {
      const entry = terms.get(id);
      if (hostRef.current && entry.paneEl.parentElement !== hostRef.current) {
        hostRef.current.appendChild(entry.paneEl);
      }
      entry.paneEl.classList.add("active");
      scheduleTermFit(entry, true);
      return;
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    if (hostRef.current) hostRef.current.appendChild(paneEl);

    const term = new Terminal(xtermOptions());
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);

    const entry = { term, fit, paneEl, sock: null, closedByUser: false };
    const sendBytes = (bytes) => {
      if (entry.sock && entry.sock.readyState === WebSocket.OPEN) entry.sock.send(bytes);
    };
    wireTermWheel(term, sendBytes, paneEl);
    wireTermKeys(term, sendBytes);
    wireTermFit(entry);
    const sock = new WebSocket(wsURL(`/ws/term?session=picode-${id}`));
    sock.binaryType = "arraybuffer";
    entry.sock = sock;

    sock.onopen = () => {
      term.reset();
      setDot(true);
      scheduleTermFit(entry, true);
      term.onData((data) => {
        const out = termDataFilter(data);
        if (out === "") return;
        if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(out));
      });
      term.onResize(() => {
        if (sock.readyState === WebSocket.OPEN) {
          sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        }
        const dims = document.getElementById("sb-dims");
        if (dims) dims.textContent = `${term.cols}×${term.rows}`;
      });
      term.focus();
    };
    sock.onmessage = (ev) => {
      if (typeof ev.data === "string") {
        try {
          const msg = JSON.parse(ev.data);
          if (msg.type === "error") term.writeln(`\r\n\x1b[31m${msg.message}\x1b[0m`);
        } catch { /* ignore */ }
        return;
      }
      term.write(new Uint8Array(ev.data));
    };
    sock.onclose = () => {
      if (entry.unwireFit) entry.unwireFit();
      setDot(false);
      if (!entry.closedByUser) {
        term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
        if (window.__picodeKickHealth) window.__picodeKickHealth();
      }
    };

    terms.set(id, entry);
    const sess = document.getElementById("sb-session");
    if (sess && workspace) sess.textContent = `picode-${id} · ${workspace.path}`;
  }, [open, agent, workspace]);

  useEffect(() => {
    if (!open || !agent) return;
    const entry = terms.get(agent.id);
    scheduleTermFit(entry);
  }, [open, agent]);

  useEffect(() => {
    function apply() {
      const id = agent && agent.id;
      if (!id) return;
      const entry = terms.get(id);
      if (!entry || !entry.term) return;
      applyXtermOptions(entry.term);
      scheduleTermFit(entry);
    }
    window.addEventListener("picode-term-theme", apply);
    return () => window.removeEventListener("picode-term-theme", apply);
  }, [agent]);

  function setDot(connected) {
    const dot = document.getElementById("sb-dot");
    const txt = document.getElementById("sb-state-text");
    if (dot) dot.classList.toggle("connected", connected);
    if (txt) txt.textContent = connected ? "connected" : "detached";
  }

  return (
    <section
      id="dock"
      className="dock view"
      hidden={!open}
    >
      <div id="terms" className="terms" ref={hostRef} />
      <div id="statusbar" className="statusbar">
        <span className="sb-state" id="sb-state">
          <span className="sb-dot" id="sb-dot" />
          <span id="sb-state-text">detached</span>
        </span>
        <span id="sb-session" className="sb-item" />
        <span id="sb-dims" className="sb-item" />
      </div>
    </section>
  );
}

export function closeTerm(id) {
  const t = terms.get(id);
  if (!t) return;
  t.closedByUser = true;
  try { t.sock.close(); } catch { /* ignore */ }
  t.term.dispose();
  t.paneEl.remove();
  terms.delete(id);
}
