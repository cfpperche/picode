import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms } from "../lib/terms.js";
import { wsURL } from "../lib/api.js";
import { IconMaximize, IconRestore } from "./Icons.jsx";
import "@xterm/xterm/css/xterm.css";

export default function TerminalDock({
  open, maximized, height, agent, workspace, onClose, onToggleMax, onHeight,
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
      requestAnimationFrame(() => entry.fit && entry.fit.fit());
      return;
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    if (hostRef.current) hostRef.current.appendChild(paneEl);

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 12,
      fontFamily: 'ui-monospace, "SF Mono", "Cascadia Code", Menlo, monospace',
      theme: {
        background: "#0e0e11",
        foreground: "#ececf1",
        cursor: "#7c8cf8",
        selectionBackground: "#33467c",
      },
      scrollback: 10000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);

    const entry = { term, fit, paneEl, sock: null, onWinResize: null, closedByUser: false };
    const sock = new WebSocket(wsURL(`/ws/term?session=picode-${id}`));
    sock.binaryType = "arraybuffer";
    entry.sock = sock;

    sock.onopen = () => {
      term.reset();
      setDot(true);
      const sendResize = () => {
        sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
        const dims = document.getElementById("sb-dims");
        if (dims) dims.textContent = `${term.cols}×${term.rows}`;
      };
      fit.fit();
      sendResize();
      entry.onWinResize = () => { if (paneEl.classList.contains("active")) { fit.fit(); sendResize(); } };
      window.addEventListener("resize", entry.onWinResize);
      term.onData((data) => {
        if (sock.readyState === WebSocket.OPEN) sock.send(new TextEncoder().encode(data));
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
      if (entry.onWinResize) window.removeEventListener("resize", entry.onWinResize);
      setDot(false);
      if (!entry.closedByUser) term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
    };

    terms.set(id, entry);
    const sess = document.getElementById("sb-session");
    if (sess && workspace) sess.textContent = `picode-${id} · ${workspace.path}`;
  }, [open, agent, workspace]);

  useEffect(() => {
    if (!open || !agent) return;
    const entry = terms.get(agent.id);
    if (entry && entry.fit) requestAnimationFrame(() => entry.fit.fit());
  }, [open, maximized, height, agent]);

  function setDot(connected) {
    const dot = document.getElementById("sb-dot");
    const txt = document.getElementById("sb-state-text");
    if (dot) dot.classList.toggle("connected", connected);
    if (txt) txt.textContent = connected ? "connected" : "detached";
  }

  function onSizerDown(e) {
    e.preventDefault();
    const dock = e.currentTarget.parentElement;
    if (maximized) onToggleMax();
    const startY = e.clientY;
    const startH = dock.getBoundingClientRect().height;
    dock.classList.add("resizing");
    const move = (ev) => {
      const view = document.getElementById("workspace-view");
      const tabs = document.getElementById("main-tabs");
      const maxH = (view ? view.getBoundingClientRect().height : 600) - (tabs ? tabs.offsetHeight : 0) - 80;
      const next = Math.max(120, Math.min(maxH, startH + (startY - ev.clientY)));
      onHeight(Math.round(next));
    };
    const up = () => {
      dock.classList.remove("resizing");
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  const style = maximized ? undefined : { height: height ? height + "px" : "42%" };

  return (
    <section
      id="dock"
      className={"dock" + (maximized ? " maximized" : "")}
      hidden={!open}
      style={style}
    >
      <div id="dock-sizer" className="dock-sizer" title="Drag to resize" onPointerDown={onSizerDown} />
      <div className="dock-head">
        <span id="dock-title" className="dock-title">
          {workspace ? "Terminal · " + workspace.name : "Terminal"}
        </span>
        <button id="dock-max" className="dock-icon" title={maximized ? "Restore" : "Maximize"} aria-label={maximized ? "Restore" : "Maximize"} onClick={onToggleMax}>
          <IconMaximize />
          <IconRestore />
        </button>
        <button id="dock-close" className="dock-icon dock-close" title="Hide terminal" aria-label="Hide terminal" onClick={onClose}>×</button>
      </div>
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
