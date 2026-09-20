import { wireTerminalRuntime } from "@picode/shared/client/terminalRuntime.js";
import { useEffect, useRef } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { terms, parkTerm, closeTerm } from "../lib/terms.js";
import { paneLeaveKey } from "@picode/shared/domain/termKeys.js";
import { matchGlobalAction } from "../lib/appKeys.js";
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

// active: this host is visible — claim the pane and fit it. autoFocus
// (default true): being active also takes the keyboard. A Canvas panel
// passes false unless it is the one focused panel (plan §4.6): every
// visible panel re-claims its pane on reveal, exactly one calls
// term.focus().
export default function ShellTerm({ agentId, session, active, autoFocus = true, cwd, cwdKind, tabId = "", onOpenFile, onOpenLink }) {
  const hostRef = useRef(null);
  const bindLinksRef = useRef(null);
  const cwdRef = useRef(cwd);
  const fileRef = useRef(onOpenFile);
  fileRef.current = onOpenFile;
  // Where a printed http(s) link opens is the app's decision (its own
  // preference), so the pane hands the URL up instead of calling window.open.
  const linkRef = useRef(onOpenLink);
  linkRef.current = onOpenLink;
  const focusRef = useRef(autoFocus);
  focusRef.current = autoFocus;
  // The pane carries its own live cwd: the right-click menu resolves the
  // token under the cursor against it, the same way the Ctrl+click underline
  // does (lib/termActions.js).
  useEffect(() => {
    cwdRef.current = cwd;
    const entry = terms.get(shellKey(agentId));
    if (entry && entry.paneEl) entry.paneEl.dataset.termCwd = cwd || "";
  }, [cwd, agentId]);

  useEffect(() => {
    if (!agentId || !session || !hostRef.current) return undefined;
    const id = shellKey(agentId);
    const host = hostRef.current;
    // Park only what this host still holds (ADR-0109): the same terminal
    // can be mounted twice — its tab and a native app's panel — and the
    // pane lives in whichever is visible. Unmounting the other host must
    // not pull the pane out from under the one showing it.
    const park = (el) => { if (el.parentElement === host) parkTerm(el); };
    const onFile = (p) => { if (fileRef.current) fileRef.current(p); };
    const liveCwd = async () => {
      try {
        const base = cwdKind === "agent" ? "/api/agents/" : "/api/terminals/";
        const page = await api(base + encodeURIComponent(agentId) + "/cwd");
        if (page && page.cwd) {
          cwdRef.current = page.cwd;
          const live = terms.get(id);
          if (live && live.paneEl) live.paneEl.dataset.termCwd = page.cwd;
        }
      } catch { /* keep cache */ }
      return cwdRef.current;
    };
    bindLinksRef.current = (entry) => {
      entry.unwireLinks?.();
      entry.unwireLinks = wireTermLinks(entry.term, () => cwdRef.current, onFile, liveCwd, (href) => {
        if (linkRef.current) linkRef.current(href);
        else window.open(href, "_blank", "noopener,noreferrer");
      });
    };
    const cached = terms.get(id);
    if (cached && cached.session !== session) closeTerm(id);
    if (terms.has(id)) {
      const entry = terms.get(id);
      // A hidden duplicate host must not steal the pane from its visible tab.
      if (!active) return () => park(entry.paneEl);
      entry.paneEl.dataset.termTabId = tabId;
      bindLinksRef.current(entry);
      const live = entry.sock && entry.sock.readyState === WebSocket.OPEN;
      if (live) {
        if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
        entry.paneEl.classList.add("active");
        scheduleTermFit(entry, true);
        if (active && focusRef.current && entry.term) entry.term.focus();
        return () => park(entry.paneEl);
      }
      // The attach dropped while we were away. Reattach the SAME instance
      // instead of rebuilding it, so the scrollback survives.
      kickTermSocket(entry);
      if (entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
      entry.paneEl.classList.add("active");
      scheduleTermFit(entry, true);
      if (active && focusRef.current && entry.term) entry.term.focus();
      return () => park(entry.paneEl);
    }
    const paneEl = document.createElement("div");
    paneEl.className = "term-pane active";
    // The right-click menu finds its host from the pane it was opened on
    // (lib/termActions.js). termKind is the dispatch address (agent vs
    // terminal APIs), not a reason to hide lifecycle rows.
    paneEl.dataset.termId = agentId;
    paneEl.dataset.termKind = cwdKind === "agent" ? "agent" : "term";
    paneEl.dataset.termCwd = cwdRef.current || "";
    paneEl.dataset.termTabId = tabId;
    hostRef.current.appendChild(paneEl);
    const term = new Terminal(xtermOptions());
    const fit = new FitAddon();
    term.loadAddon(fit);
    // Find lives on the instance, not on the bar: the query and its
    // decorations survive closing and reopening the find field.
    const search = new SearchAddon();
    term.loadAddon(search);
    term.open(paneEl);
    const entry = { term, fit, search, paneEl, session, sock: null, closedByUser: false };
    bindLinksRef.current(entry);
    wireTerminalRuntime(entry, {
      url: wsURL("/ws/term?session=" + encodeURIComponent(session)),
      onClipboardError: () => toast.error("The browser refused the copy — select and press Ctrl+C instead."),
      onDetached: () => window.__picodeKickHealth?.(),
      reservedKey: ev => matchGlobalAction(ev) || paneLeaveKey(ev),
      onOpen: () => { if (active && focusRef.current) term.focus(); },
    });
    terms.set(id, entry);
    return () => park(paneEl);
  }, [agentId, session]);

  useEffect(() => {
    if (!active || !agentId) return;
    const entry = terms.get(shellKey(agentId));
    if (!entry || !entry.term) return;
    bindLinksRef.current?.(entry);
    entry.paneEl.dataset.termTabId = tabId;
    // The pane follows the visible host (ADR-0109): shown in a native
    // app's panel and then revealed in its own tab, it comes back here
    // instead of staying in the hidden host — the tab would be empty.
    if (hostRef.current && entry.paneEl.parentElement !== hostRef.current) hostRef.current.appendChild(entry.paneEl);
    entry.paneEl.classList.add("active");
    scheduleTermFit(entry, true);
    if (autoFocus) entry.term.focus();
  }, [active, autoFocus, agentId, tabId]);

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
