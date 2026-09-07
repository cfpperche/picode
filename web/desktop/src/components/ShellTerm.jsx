import { createSticky } from "@picode/shared/domain/termSticky.js";
import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms, parkTerm, closeTerm } from "../lib/terms.js";
import { wireTermWheel } from "@picode/shared/domain/termWheel.js";
import { wireTermKeys, termDataFilter } from "@picode/shared/domain/termKeys.js";
import { matchGlobalAction } from "../lib/appKeys.js";
import { scheduleTermFit, wireTermFit } from "@picode/shared/domain/termFit.js";
import { wireTermLinks } from "@picode/shared/domain/termLinks.js";
import { wireTermClipboard } from "@picode/shared/domain/termClipboard.js";
import { api, wsURL } from "@picode/shared/client/api.js";
import { connectTermSocket, kickTermSocket } from "@picode/shared/client/termSocket.js";
import { toast } from "../lib/toast.js";
import { xtermOptions, applyXtermOptions } from "@picode/shared/domain/termTheme.js";
import "@xterm/xterm/css/xterm.css";

function shellKey(agentId) {
  return "sh:" + agentId;
}

export function closeShellTerm(agentId) {
  closeTerm(shellKey(agentId));
}

export default function ShellTerm({ agentId, session, active, cwd, cwdKind, onOpenFile }) {
  const hostRef = useRef(null);
  const cwdRef = useRef(cwd);
  const fileRef = useRef(onOpenFile);
  fileRef.current = onOpenFile;
  useEffect(() => { cwdRef.current = cwd; }, [cwd]);

  useEffect(() => {
    if (!agentId || !session || !hostRef.current) return undefined;
    const id = shellKey(agentId);
    const onFile = (p) => { if (fileRef.current) fileRef.current(p); };
    const liveCwd = async () => {
      try {
        const base = cwdKind === "agent" ? "/api/agents/" : "/api/terminals/";
        const page = await api(base + encodeURIComponent(agentId) + "/cwd");
        if (page && page.cwd) cwdRef.current = page.cwd;
      } catch { /* keep cache */ }
      return cwdRef.current;
    };
    if (terms.has(id)) {
      const entry = terms.get(id);
      const live = entry.sock && entry.sock.readyState === WebSocket.OPEN;
      if (live) {
        if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
        entry.paneEl.classList.add("active");
        scheduleTermFit(entry, true);
        if (active && entry.term) entry.term.focus();
        if (entry.term && !entry.unwireLinks) {
          entry.unwireLinks = wireTermLinks(entry.term, () => cwdRef.current, onFile, liveCwd);
        }
        return () => parkTerm(entry.paneEl);
      }
      // The attach dropped while we were away. Reattach the SAME instance
      // instead of rebuilding it, so the scrollback survives.
      kickTermSocket(entry);
      if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
      entry.paneEl.classList.add("active");
      scheduleTermFit(entry, true);
      if (active && entry.term) entry.term.focus();
      return () => parkTerm(entry.paneEl);
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    hostRef.current.appendChild(paneEl);
    const term = new Terminal(xtermOptions());
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);
    const entry = { term, fit, paneEl, sock: null, closedByUser: false, sticky: createSticky() };
    const sendBytes = (bytes) => {
      if (entry.sock && entry.sock.readyState === WebSocket.OPEN) entry.sock.send(bytes);
    };
    wireTermWheel(term, sendBytes);
    wireTermKeys(term, sendBytes, matchGlobalAction);
    wireTermClipboard(term, { onError: () => toast.error("The browser refused the copy — select and press Ctrl+C instead.") });
    wireTermFit(entry);
    entry.unwireLinks = wireTermLinks(term, () => cwdRef.current, onFile, liveCwd);
    // Key and resize handlers live on the term once — they survive a
    // reattach and read the current socket through entry.sock.
    term.onData((data) => {
      const out = entry.sticky.apply(termDataFilter(data)); // phone Ctrl/Alt, armed from the key bar
      if (out === "") return;
      sendBytes(new TextEncoder().encode(out));
    });
    term.onResize(() => {
      if (entry.sock && entry.sock.readyState === WebSocket.OPEN && term.cols > 1 && term.rows > 1) {
        entry.sock.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
      }
    });
    connectTermSocket(entry, wsURL("/ws/term?session=" + encodeURIComponent(session)), {
      onOpen: () => {
        scheduleTermFit(entry, true); // resize the fresh tmux attach
        if (active) term.focus();
      },
      onMessage: (ev) => {
        if (typeof ev.data === "string") {
          try {
            const msg = JSON.parse(ev.data);
            if (msg.type === "error") term.writeln("\r\n\x1b[31m" + msg.message + "\x1b[0m");
          } catch { /* ignore */ }
          return;
        }
        term.write(new Uint8Array(ev.data));
      },
      onState: () => {
        term.writeln("\r\n\x1b[90m— detached —\x1b[0m");
        if (window.__picodeKickHealth) window.__picodeKickHealth();
      },
      onGiveUp: () => term.writeln("\r\n\x1b[90mSession ended. Reopen the terminal.\x1b[0m"),
    });
    terms.set(id, entry);
    return () => parkTerm(paneEl);
  }, [agentId, session]);

  useEffect(() => {
    if (!active || !agentId) return;
    const entry = terms.get(shellKey(agentId));
    if (!entry || !entry.term) return;
    entry.paneEl.classList.add("active");
    scheduleTermFit(entry, true);
    entry.term.focus();
  }, [active, agentId]);

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
