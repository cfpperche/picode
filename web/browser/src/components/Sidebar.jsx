import { Fragment, useEffect, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { parseRoute, appRoute } from "../lib/routes.js";
import UserMenu from "./UserMenu.jsx";
import RailTabs from "./RailTabs.jsx";
import ShareDrawer, { OPEN_EVENT } from "./ShareDrawer.jsx";
import { IconTerminal, IconPlus, IconFolder, IconFolders, IconAgent, IconChevronRight, IconPin, IconGrid } from "./Icons.jsx";
import Pins from "./Pins.jsx";
import AppsGrid from "./AppsGrid.jsx";
import { agentsOf, displayAgentName } from "@picode/shared/domain/tree.js";
import { agentRowStatus, agentStatusLabel, agentTerm, bucketAgentsByState } from "@picode/shared/domain/agentStatus.js";
import { freeTerminals, workspaceTerminals, FREE_WS } from "../lib/termGroups.js";
import { moveId, movePhrase, sameIds, sortIdsBy } from "../lib/sidebarOrder.js";
import { sidebarBody } from "../lib/sidebarBody.js";
import ProviderFaces from "./ProviderFaces.jsx";
import { AgentRow, TermRow } from "./WorkspaceRows.jsx";
import WorkspaceMenu from "./WorkspaceMenu.jsx";
import { SortableList, SortableRow } from "./SortableRows.jsx";

// Workspace cards wear the project's favicon when it has one (ADR-0027).
// The list advertises whether one exists, so a normal workspace without an
// icon never creates an expected 404. Failures are still remembered per
// page-load when a file disappears between the list and image request.
const faviconFailed = new Set();
export function WsFavicon({ ws }) {
  const [failed, setFailed] = useState(faviconFailed.has(ws.id));
  if (failed || ws.hasFavicon === false) return <IconFolder size={16} />;
  return (
    <img
      className="ws-favicon" width={16} height={16} alt="" loading="lazy"
      src={"/api/workspaces/" + encodeURIComponent(ws.id) + "/favicon"}
      onError={() => { faviconFailed.add(ws.id); setFailed(true); }}
    />
  );
}

const SIDE_MIN = 180;
const SIDE_MAX = 480;
const SIDE_KEY = "picode-sidebar-w";
// "Working first" is a per-workspace view preference, device-local by
// design (ADR-0173 amendment): it never writes positions, so the phone and
// a second browser keep the order this one saved.
const WORKING_FIRST_KEY = "picode-ws-working-first";

// Where a list's rows will be, until the boot has read the fleet.
function SideSkeleton() {
  return <div className="side-skel" aria-label="Loading"><div className="skel-line w-80" /><div className="skel-line w-50" /></div>;
}

export default function Sidebar({
  inShell = false,
  tab, selectTab,
  workspaces, selectedId, fleetLoaded = true,
  onNew, onSelect, onRun, onStop, onRemove,
  userMenu, termView, onChat, onTerm, onOpenDocs,
  freeAgents, onNewFree, onNewAgent, onRemoveAgent, onRenameAgent,
  workingId,
  workingIds,
  waitingId,
  checklists,
  terminals, onNewTerm, onSelectTerm, onRemoveTerm, onRenameTerm, onLaunchAction, onContinueTerm, onForkAgent, clis,
  onReorder,
  onGitGraph,
  onFileTree,
  onInstructions,
  onOpenDashboard,
  onOpenClis,
  apps, nativeApps, onOpenApp, webapps, webappsErr, webappsLoaded, onRetryWebapps, onOpenWebapp, onSavedWebapp, onRemoveWebapp, onRefreshWebapp, onClearWebappData, desktop,
}) {
  const [width, setWidth] = useState(() => {
    const n = parseInt(localStorage.getItem(SIDE_KEY) || "", 10);
    return Number.isFinite(n) ? Math.min(SIDE_MAX, Math.max(SIDE_MIN, n)) : 244;
  });
  const [resizing, setResizing] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);
  useEffect(() => {
    const on = () => setShareOpen(true);
    window.addEventListener(OPEN_EVENT, on);
    return () => window.removeEventListener(OPEN_EVENT, on);
  }, []);
  useEffect(() => {
    const sync = () => {
      if (parseRoute() === "pins") selectTab("pins");
      else if (appRoute()) selectTab("apps");
    };
    sync();
    window.addEventListener("hashchange", sync);
    return () => window.removeEventListener("hashchange", sync);
  }, []);
  const [orderLive, setOrderLive] = useState("");
  const [openWs, setOpenWs] = useState(() => {
    try { return JSON.parse(localStorage.getItem("picode-ws-open") || "{}"); }
    catch { return {}; }
  });
  const [workingFirst, setWorkingFirst] = useState(() => {
    try { return JSON.parse(localStorage.getItem(WORKING_FIRST_KEY) || "{}"); }
    catch { return {}; }
  });
  // The workspace id whose header is pressed or dragged, "" otherwise. Two
  // gesture-scoped consequences hang off it: that folder renders collapsed
  // (a tall expanded slot is an invisible well and drags badly), and sticky
  // headers stand down for the whole list until release. It is set on the
  // header's pointerdown — BEFORE dnd-kit measures anything — so the drag
  // runs over the already-collapsed layout and no neighbour's rect goes
  // stale mid-gesture. A plain click collapses the folder anyway, so the
  // temporary state always agrees with the toggle the click will land on.
  // The SortableList callback keeps it in sync once dnd-kit takes over;
  // window pointerup clears it when no drag started.
  const [dragWsId, setDragWsId] = useState("");
  useEffect(() => {
    const up = () => setDragWsId("");
    window.addEventListener("pointerup", up);
    return () => window.removeEventListener("pointerup", up);
  }, []);

  function onSizerDown(e) {
    e.preventDefault();
    const startX = e.clientX;
    const startW = width;
    let latest = startW;
    setResizing(true);
    const move = (ev) => {
      latest = Math.min(SIDE_MAX, Math.max(SIDE_MIN, Math.round(startW + (ev.clientX - startX))));
      setWidth(latest);
    };
    const up = () => {
      setResizing(false);
      localStorage.setItem(SIDE_KEY, String(latest));
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  // Collapsed header advertises everyone inside: managed agents first, then
  // terminals (agent CLI favicons, shell marks) — never a fake "empty".
  function collapsedMark(agents, terms) {
    return <ProviderFaces agents={agents} terms={terms} />;
  }

  function isOpen(id) { return openWs[id] !== false && id !== dragWsId; }
  function toggleWs(id) {
    setOpenWs((s) => {
      const n = { ...s, [id]: !isOpen(id) };
      try { localStorage.setItem("picode-ws-open", JSON.stringify(n)); } catch { /* ignore */ }
      return n;
    });
  }

  function stateFirstOf(ws, wsAgents) {
    return !!workingFirst[ws.id] && wsAgents.length > 1;
  }
  function toggleWorkingFirst(id) {
    setWorkingFirst((s) => {
      const n = { ...s, [id]: !s[id] };
      try { localStorage.setItem(WORKING_FIRST_KEY, JSON.stringify(n)); } catch { /* ignore */ }
      return n;
    });
  }

  function commitOrder(kind, workspaceId, before, next, id, label) {
    if (next === before || !onReorder) return;
    setOrderLive(movePhrase(before, next, id, label));
    onReorder(kind, workspaceId, next);
  }

  // The row pill's own derivation, so a bucket and its rows can never
  // disagree about which bucket a row belongs to.
  function statusOf(ag) {
    return agentRowStatus(ag, { workingId, workingIds, waitingId, term: agentTerm(ag, terminals) });
  }

  // "Sort agents by name" (ADR-0173): a one-shot reorder, not a view — the
  // sorted ids go through the same PUT as a drop, so every client keeps one
  // order and the next drag still lands where it is dropped.
  function sortAgentsByName(ws) {
    const wsAgents = agentsOf(ws);
    const before = wsAgents.map((a) => a.id);
    const next = sortIdsBy(before, (id) => displayAgentName(wsAgents.find((a) => a.id === id) || {}, ws));
    if (sameIds(before, next)) return;
    setOrderLive("Sorted " + before.length + " agents by name.");
    onReorder("agents", ws.id, next);
  }

  function agentRow(ag, ws, ids, kind, workspaceId) {
    const selected = ag.terminalId && selectedId === "t:" + ag.terminalId ? ag.id : selectedId;
    const i = ids.indexOf(ag.id);
    const label = (id) => displayAgentName((ws ? agentsOf(ws) : sortedFreeAgents).find((a) => a.id === id) || ag, ws);
    // kind === null is the "Working first" view: presentation only, so the
    // row renders without a drag handle and without Move up / Move down —
    // both would write positions the view is not showing (ADR-0173
    // amendment). Stored order is untouched underneath.
    const row = (drag) => (
      <AgentRow
        drag={drag}
        onMoveUp={kind && i > 0 ? () => commitOrder(kind, workspaceId, ids, moveId(ids, ag.id, -1), ag.id, label) : null}
        onMoveDown={kind && i >= 0 && i < ids.length - 1 ? () => commitOrder(kind, workspaceId, ids, moveId(ids, ag.id, 1), ag.id, label) : null}
        agent={ag} ws={ws}
        selectedId={selected} onSelect={(id) => {
          if (ag.terminalId && ag.cli && ag.cli !== "pi") onSelectTerm && onSelectTerm(ag.terminalId);
          else onSelect(id);
        }}
        workingId={workingId} workingIds={workingIds} waitingId={waitingId} checklists={checklists}
        onFileTree={onFileTree} onGitGraph={onGitGraph}
        onRenameAgent={onRenameAgent}
        onRun={onRun} onStop={onStop}
        onRemoveAgent={onRemoveAgent} onRemove={onRemove}
        onChat={onChat} onTerm={onTerm} termView={termView}
        clis={clis} terms={terminals} onLaunchAction={onLaunchAction} onContinueTerm={onContinueTerm} onForkAgent={onForkAgent}
      />
    );
    if (!kind) return <Fragment key={ag.id}>{row(null)}</Fragment>;
    return <SortableRow key={ag.id} id={ag.id}>{row}</SortableRow>;
  }

  function termRow(t, ids, kind, workspaceId) {
    const i = ids.indexOf(t.id);
    const label = (id) => {
      const found = (terminals || []).find((row) => row.id === id);
      return (found && found.name) || "Terminal";
    };
    return (
      <SortableRow key={t.id} id={t.id}>
      {(drag) => (
      <TermRow
        drag={drag}
        onMoveUp={i > 0 ? () => commitOrder(kind, workspaceId, ids, moveId(ids, t.id, -1), t.id, label) : null}
        onMoveDown={i >= 0 && i < ids.length - 1 ? () => commitOrder(kind, workspaceId, ids, moveId(ids, t.id, 1), t.id, label) : null}
        term={t}
        selectedId={selectedId} onSelectTerm={onSelectTerm}
        onFileTree={onFileTree} onGitGraph={onGitGraph}
        onRenameTerm={onRenameTerm} onRemoveTerm={onRemoveTerm}
        onLaunchAction={onLaunchAction} onContinueTerm={onContinueTerm} clis={clis}
      />
      )}
      </SortableRow>
    );
  }

  // Server order (ADR-0173). Sorting here would undo a reorder.
  const sortedFreeAgents = freeAgents || [];
  const ownedTerminalIds = new Set();
  for (const w of workspaces || []) for (const a of agentsOf(w)) if (a.terminalId) ownedTerminalIds.add(a.terminalId);
  for (const a of freeAgents || []) if (a && a.terminalId) ownedTerminalIds.add(a.terminalId);
  const shellFree = freeTerminals(terminals).filter((t) => !ownedTerminalIds.has(t.id));

  // The shell's merged row sizes its brand+rail cluster to this sidebar's
  // live width, so the rail tabs sit exactly above the column they control.
  useEffect(() => {
    if (!inShell) return;
    document.documentElement.style.setProperty("--shell-nav-w", width + "px");
  }, [inShell, width]);

  return (
    <aside id="sidebar" className={resizing ? "resizing" : ""} style={{ width }}>
      {/* In the desktop shell the brand row lives in the window's merged
          top row (ADR-0122); the sidebar starts at its own content. */}
      {!inShell && <header className="brand">
        <span className="brand-title">
          <button type="button" className="brand-name" title="Dashboard" onClick={() => onOpenDashboard && onOpenDashboard()}>PiCode</button>
        </span>
        <RailTabs tab={tab} selectTab={selectTab} apps={apps} pkgUpdates={userMenu?.pkgUpdates} tight={width < 260} onOpenClis={onOpenClis} />
      </header>
      }

      {tab === "pins" ? (
        <Pins />
      ) : tab === "apps" ? (
        <AppsGrid apps={apps} appsLoaded={fleetLoaded} nativeApps={nativeApps} onOpen={onOpenApp} webapps={webapps} webappsErr={webappsErr} webappsLoaded={webappsLoaded} onRetryWebapps={onRetryWebapps} onOpenWebapp={onOpenWebapp} onSavedWebapp={onSavedWebapp} onRemoveWebapp={onRemoveWebapp} onRefreshWebapp={onRefreshWebapp} onClearWebappData={onClearWebappData} desktop={desktop} />
      ) : tab === "terms" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Terminals</span>
          <button type="button" className="ws-icon-btn" title="New terminal" onClick={() => onNewTerm && onNewTerm()}><IconPlus /></button>
        </div>
        <div className="side-scroll">
        {sidebarBody(fleetLoaded, shellFree.length) === "skeleton" ? <SideSkeleton /> : sidebarBody(fleetLoaded, shellFree.length) === "empty" ? (
          <p className="side-empty pins-empty">No terminals yet. <button type="button" className="side-empty-act" onClick={() => onNewTerm && onNewTerm()}>New terminal</button></p>
        ) : (
          <SortableList ids={shellFree.map((t) => t.id)} onReorder={(ids, activeId) => commitOrder("terminals", FREE_WS, shellFree.map((t) => t.id), ids, activeId, (id) => ((terminals || []).find((row) => row.id === id) || {}).name || "Terminal")}>
            <ul className="ws-list">{shellFree.map((t) => termRow(t, shellFree.map((row) => row.id), "terminals", FREE_WS))}</ul>
          </SortableList>
        )}
        </div>
      </div>
      ) : tab === "agents" ? (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Agents</span>
          <button type="button" className="ws-icon-btn" title="New agent" onClick={() => onNewFree()}><IconPlus /></button>
        </div>
        <div className="side-scroll">
        {sidebarBody(fleetLoaded, sortedFreeAgents.length) === "skeleton" ? <SideSkeleton /> : sidebarBody(fleetLoaded, sortedFreeAgents.length) === "empty" ? (
          <p className="side-empty pins-empty">No agents yet. <button type="button" className="side-empty-act" onClick={() => onNewFree()}>New agent</button></p>
        ) : (
          <SortableList ids={sortedFreeAgents.map((a) => a.id)} onReorder={(ids, activeId) => commitOrder("agents", FREE_WS, sortedFreeAgents.map((a) => a.id), ids, activeId, (id) => displayAgentName(sortedFreeAgents.find((a) => a.id === id), null))}>
            <ul className="ws-list">{sortedFreeAgents.map((ag) => agentRow(ag, null, sortedFreeAgents.map((a) => a.id), "agents", FREE_WS))}</ul>
          </SortableList>
        )}
        </div>
      </div>
      ) : (
      <div className="side-section">
        <div className="pins-head">
          <span className="pins-title">Workspaces</span>
          <button id="btn-new" type="button" className="ws-icon-btn" title="New workspace" onClick={() => onNew()}><IconPlus /></button>
        </div>
        <div className="side-scroll">
        {sidebarBody(fleetLoaded, workspaces.length) === "skeleton" ? <SideSkeleton /> : sidebarBody(fleetLoaded, workspaces.length) === "empty" ? (
          <p className="side-empty pins-empty">No workspaces yet. <button type="button" className="side-empty-act" onClick={() => onNew()}>Add workspace</button></p>
        ) : (
        <SortableList ids={workspaces.map((w) => w.id)} onDragActive={setDragWsId} onReorder={(ids, activeId) => commitOrder("workspaces", null, workspaces.map((w) => w.id), ids, activeId, (id) => (workspaces.find((w) => w.id === id) || {}).name || "Workspace")}>
        <ul id="ws-list" className={"ws-list" + (dragWsId ? " is-ws-dragging" : "")}>
          {workspaces.map((ws) => {
            const wsAgents = agentsOf(ws);
            const wsTerms = workspaceTerminals(terminals, ws.id).filter((t) => !ownedTerminalIds.has(t.id));
            const wsAgentIds = wsAgents.map((a) => a.id);
            const wsTermIds = wsTerms.map((t) => t.id);
            const wsIndex = workspaces.findIndex((w) => w.id === ws.id);
            const stateFirst = stateFirstOf(ws, wsAgents);
            return (
            <SortableRow key={ws.id} id={ws.id}>
            {(drag) => (
            <li
              ref={drag.setNodeRef}
              style={drag.style}
              data-drag-id={ws.id}
              className={"ws-group" + (drag.isDragging ? " is-placeholder" : "")}
            >
              <div className="ws-group-head" onClick={() => toggleWs(ws.id)}>
                <span className={"ws-chev" + (isOpen(ws.id) ? " open" : "")}><IconChevronRight /></span>
                <span className="tree-icon"><WsFavicon ws={ws} /></span>
                <span className="ws-group-name" title={ws.path} onPointerDown={(e) => { if (e.button === 0) setDragWsId(ws.id); drag.onPointerDown(e); }}>{ws.name}</span>
                <span className="tree-meta">{!isOpen(ws.id) ? collapsedMark(wsAgents, wsTerms) : null}</span>
                {/* Two controls, not five: one creates, one holds the rest
                    (VS Code caps a row at three; the child rows already read
                    this way). Both stay visible — a control hidden until
                    hover is unreachable by touch (WCAG 1.4.13). */}
                <span className="ws-group-actions" onClick={(e) => e.stopPropagation()}>
                  <DropdownMenu.Root><DropdownMenu.Trigger asChild><button type="button" className="ws-icon-btn" title="New in this folder" aria-label={"New in " + ws.name}><IconPlus /></button></DropdownMenu.Trigger><DropdownMenu.Portal><DropdownMenu.Content className="um-popover" side="bottom" align="end" sideOffset={6} collisionPadding={12}>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewAgent && onNewAgent(ws.id)}>Agent</DropdownMenu.Item>
                    <DropdownMenu.Item className="um-item" onSelect={() => onNewTerm?.(ws.id)}>Shell terminal</DropdownMenu.Item>
                  </DropdownMenu.Content></DropdownMenu.Portal></DropdownMenu.Root>
                  <WorkspaceMenu
                    ws={ws}
                    onMoveUp={wsIndex > 0 ? () => commitOrder("workspaces", null, workspaces.map((w) => w.id), moveId(workspaces.map((w) => w.id), ws.id, -1), ws.id, (id) => (workspaces.find((w) => w.id === id) || {}).name || "Workspace") : null}
                    onMoveDown={wsIndex >= 0 && wsIndex < workspaces.length - 1 ? () => commitOrder("workspaces", null, workspaces.map((w) => w.id), moveId(workspaces.map((w) => w.id), ws.id, 1), ws.id, (id) => (workspaces.find((w) => w.id === id) || {}).name || "Workspace") : null}
                    onSortAgents={wsAgents.length > 1 ? () => sortAgentsByName(ws) : null}
                    workingFirst={stateFirst}
                    onToggleWorkingFirst={wsAgents.length > 1 ? () => toggleWorkingFirst(ws.id) : null}
                    onFileTree={onFileTree}
                    onGitGraph={onGitGraph}
                    onInstructions={onInstructions}
                    onRemove={onRemove}
                  />
                </span>
              </div>
              {isOpen(ws.id) ? (
                (wsAgents.length || wsTerms.length) ? (
                  <>
                    {wsAgents.length ? (
                      stateFirst ? (
                        <ul className="ws-list tree-children">
                          {bucketAgentsByState(wsAgents, statusOf).map((b) => (
                            <Fragment key={b.status}>
                              <li className="ws-state-head">
                                <span className={"ws-state-dot is-" + b.status} aria-hidden="true" />
                                <span>{agentStatusLabel(b.status)}</span>
                                <span className="ws-state-count">{b.agents.length}</span>
                              </li>
                              {b.agents.map((ag) => agentRow(ag, ws, wsAgentIds, null, ws.id))}
                            </Fragment>
                          ))}
                        </ul>
                      ) : (
                        <SortableList ids={wsAgentIds} onReorder={(ids, activeId) => commitOrder("agents", ws.id, wsAgentIds, ids, activeId, (id) => displayAgentName(wsAgents.find((a) => a.id === id), ws))}>
                          <ul className="ws-list tree-children">{wsAgents.map((ag) => agentRow(ag, ws, wsAgentIds, "agents", ws.id))}</ul>
                        </SortableList>
                      )
                    ) : null}
                    {wsTerms.length ? (
                      <SortableList ids={wsTermIds} onReorder={(ids, activeId) => commitOrder("terminals", ws.id, wsTermIds, ids, activeId, (id) => (wsTerms.find((t) => t.id === id) || {}).name || "Terminal")}>
                        <ul className="ws-list tree-children">{wsTerms.map((t) => termRow(t, wsTermIds, "terminals", ws.id))}</ul>
                      </SortableList>
                    ) : null}
                  </>
                ) : (
                  <p className="side-empty">
                    Empty — <button type="button" className="side-empty-act" onClick={() => onNewAgent && onNewAgent(ws.id)}>add an agent</button>
                    {" or "}<button type="button" className="side-empty-act" onClick={() => onNewTerm && onNewTerm(ws.id)}>a terminal</button>.
                  </p>
                )
              ) : null}
            </li>
            )}
            </SortableRow>
            );
          })}
        </ul>
        </SortableList>
        )}
        </div>
      </div>
      )}

      <div className="sr-only" aria-live="polite">{orderLive}</div>
      <footer className="side-foot">
        <UserMenu {...userMenu} onShare={() => setShareOpen(true)} onDocs={onOpenDocs} />
      </footer>
      <div id="sidebar-sizer" className="sidebar-sizer" title="Drag to resize" onPointerDown={onSizerDown} />
      <ShareDrawer open={shareOpen} onClose={() => setShareOpen(false)} />
    </aside>
  );
}
