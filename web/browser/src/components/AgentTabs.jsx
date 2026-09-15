import { useEffect, useLayoutEffect, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { locate, displayAgentName } from "@picode/shared/domain/tree.js";
import { isTermTab, tabTermId, isFileTab, parseFileTab, isGitTab, gitTabKey, isTreeTab, treeTabRoot, isAppTab, tabAppId, isWebTab, tabWebId } from "../lib/routes.js";
import { repoNameFromKey } from "../lib/gitgraph.js";
import { revealLeft } from "../lib/tabStrip.js";
import { useTabStrip } from "../lib/useTabStrip.js";
import { matchAction } from "../lib/appKeys.js";
import { IconFile, IconGit, IconFolders, IconChevronLeft, IconChevronRight, IconList, IconCheck } from "./Icons.jsx";
import AppIcon from "./AppIcon.jsx";
import { IconGlobe } from "./Icons.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import { terminalCli, terminalCliLabel, terminalDisplayCli, terminalStatus } from "@picode/shared/domain/terminalCli.js";

// One description per tab id, shared by the strip and the "All tabs"
// list so both show the same face, name and status. Null means the tab
// has nothing to render yet (a terminal the client has not received).
function describeTab(id, { terms, appList, workspaces, freeAgents, webTabs }) {
  if (isWebTab(id)) {
    const w = webTabs?.[tabWebId(id)];
    const label = (w?.title || w?.url || "").replace(/^https?:\/\//, "").split("/")[0] || "New tab";
    return { icon: <IconGlobe />, label, title: w?.url || "", status: null, closeTitle: "Close tab" };
  }
  if (isTermTab(id)) {
    const term = terms.find((t) => t.id === tabTermId(id));
    if (!term) return null;
    // Terminal CLI state on the tab (ADR-0056/0060): needs-you is the
    // user's move, so it gets the accent dot; working gets the same accent
    // motion as a running agent.
    const st = terminalStatus(term);
    return {
      icon: <TerminalCliBadge term={term} />,
      label: term.name,
      title: terminalDisplayCli(term) ? terminalCliLabel(terminalDisplayCli(term)) : "Terminal",
      status: st === "needs-you" ? "attn" : st === "working" ? "running" : null,
      closeTitle: "Close tab (terminal keeps running)",
    };
  }
  if (isFileTab(id)) {
    const f = parseFileTab(id);
    if (!f) return null;
    const name = (f.path || "").split("/").pop() || f.path || "File";
    return { icon: <IconFile size={13} />, label: name, title: f.path, status: null, closeTitle: "Close tab" };
  }
  if (isGitTab(id)) {
    // The tab is the repository (ADR-0022), so its name comes from the
    // key rather than from whichever owner happened to open it.
    return { icon: <IconGit size={13} />, label: repoNameFromKey(gitTabKey(id)) || "Git", title: gitTabKey(id), status: null, closeTitle: "Close tab" };
  }
  if (isTreeTab(id)) {
    // The tab is the folder (ADR-0030); its name is the folder's own.
    const root = treeTabRoot(id);
    const name = root.startsWith("@") ? "Files" : root.split("/").filter(Boolean).pop() || root;
    return { icon: <IconFolders size={13} />, label: name, title: root, status: null, closeTitle: "Close tab" };
  }
  if (isAppTab(id)) {
    // The manifest may not have arrived yet — render the raw id rather
    // than null (a null tab silently vanishes, ADR-0036).
    const aid = tabAppId(id);
    const m = appList.find((a) => a.id === aid);
    return { icon: <AppIcon name={m ? m.icon : ""} label={m ? m.name : aid} size={13} />, label: m ? m.name : aid, title: "", status: null, closeTitle: "Close tab" };
  }
  const loc = locate(workspaces, freeAgents, id);
  if (!loc || !loc.agent) return null;
  const ag = loc.agent;
  // Agent tabs mirror terminal tabs (ADR-0056/0060): identity leads as the
  // favicon — the agent's provider face, the same mark the sidebar wears —
  // and the trailing dot carries activity: needs-you is the user's move
  // (accent), running gets the green working dot.
  const mode = ag.mode || "stopped";
  return {
    icon: <ProviderFace agent={ag} />,
    label: displayAgentName(ag, loc.workspace),
    title: "",
    status: ag.waiting ? "attn" : mode !== "stopped" ? "running" : null,
    closeTitle: "Close tab (agent keeps running)",
  };
}

function StatusDot({ status }) {
  if (status === "attn") return <span className="mtab-dot attn" title="Needs you" />;
  if (status === "running") return <span className="mtab-dot running" title="Working" />;
  return null;
}

export default function AgentTabs({ tabs, workspaces, freeAgents, terminals, apps, webTabs, selectedId, onSelect, onClose, onReorder, sessionSlot, endSlot, keepVisible }) {
  const ctx = { terms: terminals || [], appList: apps || [], workspaces, freeAgents, webTabs };
  const entries = tabs.map((id) => ({ id, d: describeTab(id, ctx) })).filter((e) => e.d);
  const ids = entries.map((e) => e.id);
  const stripRef = useRef(null);
  const revealed = useRef(false);
  const strip = useTabStrip(stripRef, [tabs]);
  // The latest props for listeners that outlive a render.
  const live = useRef({ ids, selectedId, onSelect, onClose });
  live.current = { ids, selectedId, onSelect, onClose };

  // The strip has no scrollbar, so selecting or opening a tab must bring it
  // into view (every benchmark does; see lib/tabStrip.js). The first reveal
  // after mount jumps — a smooth glide from 0 on page load reads as motion
  // nobody asked for; later reveals follow the strip's scroll-behavior.
  useLayoutEffect(() => {
    const el = stripRef.current;
    if (!el) return;
    const tab = el.querySelector(".mtab.active");
    if (!tab) return;
    const left = revealLeft(el, { left: tab.offsetLeft, width: tab.offsetWidth });
    if (left != null) el.scrollTo(revealed.current ? { left } : { left, behavior: "instant" });
    revealed.current = true;
    // strip.overflow: the arrows and the list button appear the moment tabs
    // overflow and take width from the strip, which can clip the tab this
    // effect had just revealed.
  }, [selectedId, tabs, strip.overflow]);

  // Alt+[ / Alt+] cycle tabs from anywhere, wrapping like a browser, and
  // Alt+W closes the current one; terminals hand these chords back
  // (wireTermKeys passthrough).
  useEffect(() => {
    const onKey = (e) => {
      if (matchAction("app.tab.close", e)) {
        const { selectedId: sel, ids: all, onClose: close } = live.current;
        if (!sel || !all.includes(sel)) return;
        e.preventDefault();
        close(sel);
        return;
      }
      const prev = matchAction("app.tab.prev", e);
      const next = !prev && matchAction("app.tab.next", e);
      if (!prev && !next) return;
      const { ids: all, selectedId: sel, onSelect: select } = live.current;
      if (all.length < 2) return;
      e.preventDefault();
      const i = all.indexOf(sel);
      const j = i < 0 ? 0 : (i + (next ? 1 : -1) + all.length) % all.length;
      select(all[j]);
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  // WAI-ARIA tablist with manual activation (MUI default, Radix `manual`):
  // arrows / Home / End move focus between tabs, Enter or Space selects,
  // Delete or Backspace closes the focused tab (the ARIA tabs-with-removal
  // pattern) and keeps focus on its neighbour.
  // Automatic activation would not work here — selecting a terminal tab
  // hands focus to its xterm textarea, so the next arrow would go to the
  // shell instead of the strip.
  const onStripKey = (e) => {
    if (e.target.closest(".mtab-close")) return;
    const el = stripRef.current;
    const tabsEl = el ? [...el.querySelectorAll(".mtab")] : [];
    const cur = e.target.closest(".mtab");
    const i = Math.max(0, tabsEl.indexOf(cur));
    if (e.key === "Enter" || e.key === " ") {
      if (!cur) return;
      e.preventDefault();
      if (cur.dataset.tab !== selectedId) onSelect(cur.dataset.tab);
      return;
    }
    if (e.key === "Delete" || e.key === "Backspace") {
      if (!cur) return;
      e.preventDefault();
      const neighbour = tabsEl[i + 1] || tabsEl[i - 1];
      onClose(cur.dataset.tab);
      if (neighbour) neighbour.focus();
      return;
    }
    let next = null;
    if (e.key === "ArrowRight") next = tabsEl[Math.min(tabsEl.length - 1, i + 1)];
    else if (e.key === "ArrowLeft") next = tabsEl[Math.max(0, i - 1)];
    else if (e.key === "Home") next = tabsEl[0];
    else if (e.key === "End") next = tabsEl[tabsEl.length - 1];
    else return;
    e.preventDefault();
    if (next && next !== cur) next.focus();
  };

  const statusOf = (id) => { const e = entries.find((x) => x.id === id); return e ? e.d.status : null; };
  const attnLeft = strip.hidden.left.some((id) => statusOf(id) === "attn");
  const attnRight = strip.hidden.right.some((id) => statusOf(id) === "attn");
  const hiddenSet = new Set([...strip.hidden.left, ...strip.hidden.right]);
  const outOfView = entries.filter((e) => hiddenSet.has(e.id));
  const inView = entries.filter((e) => !hiddenSet.has(e.id));

  const listItem = ({ id, d }) => (
    <DropdownMenu.Item key={id} className="um-item tab-list-item" onSelect={() => onSelect(id)} aria-current={id === selectedId ? "true" : undefined}>
      <span className="um-item-name">
        <span className="mtab-term">{d.icon}</span>
        <span className="um-name">{d.label}</span>
      </span>
      <StatusDot status={d.status} />
      {id === selectedId ? <IconCheck size={13} /> : null}
    </DropdownMenu.Item>
  );

  return (
    <>
    <div id="main-tabs" className="main-tabs" hidden={tabs.length === 0 && !keepVisible}>
      {/* Overflow chrome (phases 2–3 of the tab-strip study): arrows exist
          only while tabs overflow and stay visible then (NN/g: hover-only
          arrows go unnoticed); the one at a reached edge is disabled, the
          fade on that edge drops, an arrow wears the needs-you dot when a
          tab beyond it wants the user, and a 3px indicator under the tabs
          shows where the viewport sits while the pointer is over the strip
          or it moves — pointer-events off, so no VS Code spurious clicks.
          The "All tabs" list (JetBrains, Sublime, Firefox) puts the
          out-of-view tabs first. */}
      {strip.overflow ? (
        <button type="button" className="tab-arrow" disabled={strip.atStart} title="Scroll tabs left" aria-label="Scroll tabs left" onClick={() => strip.step(-1)}>
          <IconChevronLeft />
          {attnLeft ? <span className="mtab-dot attn tab-arrow-dot" title="A tab to the left needs you" /> : null}
        </button>
      ) : null}
      <div className="tab-scroller" data-scrolling={strip.scrolling ? "" : undefined}>
      <div id="tab-strip" className="tab-strip" ref={stripRef} role="tablist" aria-orientation="horizontal" aria-label="Open tabs" data-at-start={strip.atStart} data-at-end={strip.atEnd} onKeyDown={onStripKey}>
        {entries.map(({ id, d }) => (
          <Tab key={id} id={id} active={id === selectedId} attn={d.status === "attn"} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle={d.closeTitle}>
            <span className="mtab-term">{d.icon}</span>
            <span className="mtab-label" title={d.title && d.title !== d.label ? `${d.label} — ${d.title}` : d.label}>{d.label}</span>
            <StatusDot status={d.status} />
          </Tab>
        ))}
      </div>
      {strip.overflow ? (
        <div className="tab-indicator" aria-hidden="true" style={{ left: `${strip.thumb.left * 100}%`, width: `${strip.thumb.width * 100}%` }} />
      ) : null}
      </div>
      {strip.overflow ? (
        <button type="button" className="tab-arrow right" disabled={strip.atEnd} title="Scroll tabs right" aria-label="Scroll tabs right" onClick={() => strip.step(1)}>
          <IconChevronRight />
          {attnRight ? <span className="mtab-dot attn tab-arrow-dot" title="A tab to the right needs you" /> : null}
        </button>
      ) : null}
      {strip.overflow ? (
        <DropdownMenu.Root>
          <DropdownMenu.Trigger asChild>
            <button type="button" className="tab-arrow tab-list-trigger" title="All tabs" aria-label="All tabs">
              <IconList />
            </button>
          </DropdownMenu.Trigger>
          <DropdownMenu.Portal>
            <DropdownMenu.Content className="um-popover tab-list" side="bottom" align="end" sideOffset={4} collisionPadding={8}>
              {outOfView.length ? <div className="um-label">Out of view</div> : null}
              {outOfView.map(listItem)}
              {outOfView.length && inView.length ? <DropdownMenu.Separator className="um-divider" /> : null}
              {inView.map(listItem)}
            </DropdownMenu.Content>
          </DropdownMenu.Portal>
        </DropdownMenu.Root>
      ) : null}
      {endSlot ? <div className="main-tabs-end" data-align-row>{endSlot}</div> : null}
    </div>
    {tabs.length > 0 && !isTermTab(selectedId) && !isFileTab(selectedId) && !isGitTab(selectedId) && !isTreeTab(selectedId) && !isAppTab(selectedId) ? sessionSlot : null}
    </>
  );
}

function Tab({ id, active, attn, onSelect, onClose, onReorder, closeTitle, children }) {
  const [over, setOver] = useState(false);
  const [dragging, setDragging] = useState(false);
  return (
    <div
      className={"mtab" + (active ? " active" : "") + (over ? " drag-over" : "") + (dragging ? " dragging" : "")}
      role="tab"
      aria-selected={active}
      tabIndex={active ? 0 : -1}
      data-tab={id}
      data-attn={attn ? "" : undefined}
      draggable="true"
      onDragStart={(e) => {
        if (e.target.closest(".mtab-close")) { e.preventDefault(); return; }
        e.dataTransfer.setData("text/plain", id);
        e.dataTransfer.effectAllowed = "move";
        setDragging(true);
      }}
      onDragEnd={() => { setDragging(false); setOver(false); }}
      onDragOver={(e) => {
        if (!onReorder) return;
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        e.preventDefault();
        setOver(false);
        const from = e.dataTransfer.getData("text/plain");
        if (from && from !== id && onReorder) onReorder(from, id);
      }}
      onClick={(e) => { if (e.target.closest(".mtab-close")) return; onSelect(id); }}
      onFocus={(e) => { if (e.target === e.currentTarget) e.currentTarget.scrollIntoView({ inline: "nearest", block: "nearest" }); }}
    >
      <span className="mtab-bg" aria-hidden="true" />
      {children}
      <button type="button" className="mtab-close" draggable="false" tabIndex={-1} title={closeTitle + " · Alt+W, or Delete on a focused tab"} onClick={() => onClose(id)}>×</button>
    </div>
  );
}
