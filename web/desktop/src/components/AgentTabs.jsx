import { useLayoutEffect, useRef, useState } from "react";
import { locate, displayAgentName } from "@picode/shared/domain/tree.js";
import { isTermTab, tabTermId, isFileTab, parseFileTab, isGitTab, gitTabKey, isTreeTab, treeTabRoot, isAppTab, tabAppId } from "../lib/routes.js";
import { repoNameFromKey } from "../lib/gitgraph.js";
import { revealLeft } from "../lib/tabStrip.js";
import { useTabStrip } from "../lib/useTabStrip.js";
import { IconFile, IconGit, IconFolders, IconChevronLeft, IconChevronRight } from "./Icons.jsx";
import AppIcon from "./AppIcon.jsx";
import TerminalCliBadge from "./TerminalCliBadge.jsx";
import { ProviderFace } from "./ProviderFaces.jsx";
import { terminalCli, terminalCliLabel, terminalStatus } from "@picode/shared/domain/terminalCli.js";

export default function AgentTabs({ tabs, workspaces, freeAgents, terminals, apps, selectedId, onSelect, onClose, onReorder, sessionSlot, endSlot }) {
  const terms = terminals || [];
  const appList = apps || [];
  const stripRef = useRef(null);
  const revealed = useRef(false);
  const strip = useTabStrip(stripRef, [tabs]);
  // The strip has no scrollbar, so selecting or opening a tab must bring it
  // into view (every benchmark does; see lib/tabStrip.js). The first reveal
  // after mount jumps — a smooth glide from 0 on page load reads as motion
  // nobody asked for; later reveals follow the strip's scroll-behavior.
  useLayoutEffect(() => {
    const strip = stripRef.current;
    if (!strip) return;
    const el = strip.querySelector(".mtab.active");
    if (!el) return;
    const left = revealLeft(strip, { left: el.offsetLeft, width: el.offsetWidth });
    if (left != null) strip.scrollTo(revealed.current ? { left } : { left, behavior: "instant" });
    revealed.current = true;
  }, [selectedId, tabs]);
  return (
    <>
    <div id="main-tabs" className="main-tabs" hidden={tabs.length === 0}>
      {/* Overflow chrome (phase 2 of the tab-strip study): arrows exist
          only while tabs overflow and stay visible then (NN/g: hover-only
          arrows go unnoticed); the one at a reached edge is disabled, the
          fade on that edge drops, and a 3px indicator under the tabs shows
          where the viewport sits while the pointer is over the strip or it
          moves — pointer-events off, so no VS Code spurious clicks. */}
      {strip.overflow ? (
        <button type="button" className="tab-arrow" disabled={strip.atStart} title="Scroll tabs left" aria-label="Scroll tabs left" onClick={() => strip.step(-1)}>
          <IconChevronLeft />
        </button>
      ) : null}
      <div className="tab-scroller" data-scrolling={strip.scrolling ? "" : undefined}>
      <div id="tab-strip" className="tab-strip" ref={stripRef} data-at-start={strip.atStart} data-at-end={strip.atEnd}>
        {tabs.map((id) => {
          if (isTermTab(id)) {
            const tid = tabTermId(id);
            const term = terms.find((t) => t.id === tid);
            if (!term) return null;
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab (terminal keeps running)">
                <span className="mtab-term"><TerminalCliBadge term={term} /></span>
                <span title={terminalCli(term) ? terminalCliLabel(terminalCli(term)) : "Terminal"}>{term.name}</span>
                {/* Terminal CLI state on the tab (ADR-0056/0060): needs-you is
                    the user's move, so it gets the accent dot; working gets
                    the same accent motion as a running agent. */}
                {terminalStatus(term) === "needs-you" ? <span className="mtab-dot attn" title="Needs you" /> : null}
                {terminalStatus(term) === "working" ? <span className="mtab-dot running" title="Working" /> : null}
              </Tab>
            );
          }
          if (isFileTab(id)) {
            const f = parseFileTab(id);
            if (!f) return null;
            const name = (f.path || "").split("/").pop() || f.path || "File";
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconFile size={13} /></span>
                <span title={f.path}>{name}</span>
              </Tab>
            );
          }
          if (isGitTab(id)) {
            // The tab is the repository (ADR-0022), so its name comes from the
            // key rather than from whichever owner happened to open it.
            const name = repoNameFromKey(gitTabKey(id)) || "Git";
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconGit size={13} /></span>
                <span title={gitTabKey(id)}>{name}</span>
              </Tab>
            );
          }
          if (isTreeTab(id)) {
            // The tab is the folder (ADR-0030); its name is the folder's own.
            const root = treeTabRoot(id);
            const name = root.startsWith("@") ? "Files" : root.split("/").filter(Boolean).pop() || root;
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><IconFolders size={13} /></span>
                <span title={root}>{name}</span>
              </Tab>
            );
          }
          if (isAppTab(id)) {
            // The manifest may not have arrived yet — render the raw id
            // rather than null (a null tab silently vanishes, ADR-0036).
            const aid = tabAppId(id);
            const m = appList.find((a) => a.id === aid);
            return (
              <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab">
                <span className="mtab-term"><AppIcon name={m ? m.icon : ""} label={m ? m.name : aid} size={13} /></span>
                <span>{m ? m.name : aid}</span>
              </Tab>
            );
          }
          const loc = locate(workspaces, freeAgents, id);
          if (!loc || !loc.agent) return null;
          const ag = loc.agent;
          const mode = ag.mode || "stopped";
          return (
            <Tab key={id} id={id} active={id === selectedId} onSelect={onSelect} onClose={onClose} onReorder={onReorder} closeTitle="Close tab (agent keeps running)">
              {/* Agent tabs mirror terminal tabs (ADR-0056/0060): identity
                  leads as the favicon — the agent's provider face, the same
                  mark the sidebar wears — and the trailing dot carries
                  activity: needs-you is the user's move (accent), running
                  gets the green working dot. */}
              <span className="mtab-term"><ProviderFace agent={ag} /></span>
              <span>{displayAgentName(ag, loc.workspace)}</span>
              {ag.waiting ? <span className="mtab-dot attn" title="Needs you" /> : null}
              {!ag.waiting && mode !== "stopped" ? <span className="mtab-dot running" title="Working" /> : null}
            </Tab>
          );
        })}
      </div>
      {strip.overflow ? (
        <div className="tab-indicator" aria-hidden="true" style={{ left: `${strip.thumb.left * 100}%`, width: `${strip.thumb.width * 100}%` }} />
      ) : null}
      </div>
      {strip.overflow ? (
        <button type="button" className="tab-arrow right" disabled={strip.atEnd} title="Scroll tabs right" aria-label="Scroll tabs right" onClick={() => strip.step(1)}>
          <IconChevronRight />
        </button>
      ) : null}
      {endSlot ? <div className="main-tabs-end">{endSlot}</div> : null}
    </div>
    {tabs.length > 0 && !isTermTab(selectedId) && !isFileTab(selectedId) && !isGitTab(selectedId) && !isTreeTab(selectedId) && !isAppTab(selectedId) ? sessionSlot : null}
    </>
  );
}

function Tab({ id, active, onSelect, onClose, onReorder, closeTitle, children }) {
  const [over, setOver] = useState(false);
  const [dragging, setDragging] = useState(false);
  return (
    <div
      className={"mtab" + (active ? " active" : "") + (over ? " drag-over" : "") + (dragging ? " dragging" : "")}
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
    >
      {children}
      <button type="button" className="mtab-close" draggable="false" title={closeTitle} onClick={() => onClose(id)}>×</button>
    </div>
  );
}
