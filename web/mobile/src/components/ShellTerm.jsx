import { wireTerminalRuntime } from "@picode/shared/client/terminalRuntime.js";
import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { terms, parkTerm, closeTerm } from "../lib/terms.js";
import { scheduleTermFit } from "@picode/shared/domain/termFit.js";
import { wireTermLinks } from "@picode/shared/domain/termLinks.js";
import { api, wsURL } from "@picode/shared/client/api.js";
import { kickTermSocket } from "@picode/shared/client/termSocket.js";
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
    const cached = terms.get(id);
    if (cached && cached.session !== session) closeTerm(id);
    if (terms.has(id)) {
      const entry = terms.get(id);
      if (entry.unwireLinks) entry.unwireLinks();
      entry.unwireLinks = wireTermLinks(entry.term, () => cwdRef.current, onFile, liveCwd);
      const live = entry.sock && entry.sock.readyState === WebSocket.OPEN;
      if (live) {
        if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
        entry.paneEl.classList.add("active");
        scheduleTermFit(entry, true);
        if (entry.term && !entry.unwireLinks) {
          entry.unwireLinks = wireTermLinks(entry.term, () => cwdRef.current, onFile, liveCwd);
        }
        return () => parkTerm(entry.paneEl);
      }
      // The attach dropped while we were away (phone lock, network).
      // Reattach the SAME instance instead of rebuilding it, so the
      // scrollback the reader was following survives.
      kickTermSocket(entry);
      if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
      entry.paneEl.classList.add("active");
      scheduleTermFit(entry, true);
      return () => parkTerm(entry.paneEl);
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    hostRef.current.appendChild(paneEl);
    const term = new Terminal(xtermOptions());
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(paneEl);
    const entry = { term, fit, paneEl, session, sock: null, closedByUser: false };
    entry.unwireLinks = wireTermLinks(term, () => cwdRef.current, onFile, liveCwd);
    wireTerminalRuntime(entry, {
      url: wsURL("/ws/term?session=" + encodeURIComponent(session)),
      onClipboardError: () => toast.error("The browser refused the copy — select and press Ctrl+C instead."),
      onDetached: () => window.__picodeKickHealth?.(),
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
