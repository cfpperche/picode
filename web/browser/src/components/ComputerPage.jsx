import { useCallback, useEffect, useMemo, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { AuditList, Item, SwitchCtl, WsTag, distinctOutcomes } from "./settingsControls.jsx";
import { IconWarn } from "./Icons.jsx";
import { relTime } from "@picode/shared/domain/relTime.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { toast } from "../lib/toast.js";

// Computer (ADR-0148): who may use the desktop through the app, and what
// they did. One switch per principal — a managed agent or a CLI running in
// a PiCode terminal — and nothing else in v1: with the switch on, the agent
// has the whole desktop with your own permissions. The refinements (a
// window binding, tiers, confirmations) come later as amendments.
//
// Desktop-only by nature: the entry renders only inside the desktop shell,
// because only the shell can see and drive the desktop.
//
// Grants/audit paging (Carbon table language): a page renders free, the
// footer pages the rest instead of virtualizing a settings surface.
const GRANT_PAGE = 100;
const AUDIT_PAGE = 50;

export default function ComputerPage({ hidden, onCreateAgent }) {
  const [rows, setRows] = useState(null);
  const [steps, setSteps] = useState([]);
  const [busy, setBusy] = useState(null);
  // Access toolbar: search + workspace filter, paged at GRANT_PAGE.
  const [grantQuery, setGrantQuery] = useState("");
  const [grantWs, setGrantWs] = useState("all");
  const [grantLimit, setGrantLimit] = useState(GRANT_PAGE);
  const [wsNames, setWsNames] = useState({});
  // Recent steps (Linear/Stripe language): search + outcome filter, paged.
  const [auditQuery, setAuditQuery] = useState("");
  const [auditOutcome, setAuditOutcome] = useState("all");
  const [auditLimit, setAuditLimit] = useState(AUDIT_PAGE);

  const load = useCallback(async () => {
    try {
      const r = await fetch("/api/computer/policies");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setRows(data.policies ?? []);
    } catch {
      toast("Could not load the computer grants.");
    }
  }, []);
  const loadSteps = useCallback(async () => {
    try {
      const r = await fetch("/api/computer/audit?limit=100");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setSteps(data.steps ?? []);
    } catch {
      setSteps([]);
    }
  }, []);

  // Workspace names for the provenance chips (id → name; the raw id stays
  // in the chip's title). Loaded once per mount, not per feed event.
  const loadWs = useCallback(async () => {
    try {
      const r = await fetch("/api/workspaces");
      if (!r.ok) return;
      const data = await r.json();
      const names = {};
      for (const w of Array.isArray(data) ? data : (data.workspaces ?? [])) {
        const id = w.id ?? w.ID;
        const name = w.name ?? w.Name;
        if (id) names[id] = name || id;
      }
      setWsNames(names);
    } catch {
      /* chips fall back to the raw id */
    }
  }, []);

  useEffect(() => {
    if (hidden) return;
    load();
    loadWs();
    loadSteps();
  }, [hidden, load, loadWs, loadSteps]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "setting.updated") load();
    if (ev.type === "computer.step") loadSteps();
  }), [load, loadSteps]);

  const keyOf = (row) => (row.termId ? "term:" + row.termId : row.agentId);
  const flip = async (row, enabled) => {
    setBusy(keyOf(row));
    try {
      const body = row.termId ? { term: row.termId, enabled } : { agent: row.agentId, enabled };
      const r = await fetch("/api/computer/policy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!r.ok) throw new Error(String(r.status));
      setRows((cur) => (cur || []).map((x) => (keyOf(x) === keyOf(row) ? { ...x, enabled, saved: true } : x)));
    } catch {
      toast("Saving the grant failed.");
    } finally {
      setBusy(null);
    }
  };

  const grantWsOptions = useMemo(() => {
    const seen = new Map();
    for (const row of rows || []) {
      if (row.workspaceId && !seen.has(row.workspaceId)) {
        seen.set(row.workspaceId, wsNames[row.workspaceId] || row.workspaceId);
      }
    }
    return [...seen.entries()].sort((a, b) => String(a[1]).localeCompare(String(b[1])));
  }, [rows, wsNames]);
  const filteredGrants = useMemo(() => {
    const q = grantQuery.trim().toLowerCase();
    return (rows || []).filter((row) => {
      if (grantWs === "none") {
        if (row.workspaceId) return false;
      } else if (grantWs !== "all" && row.workspaceId !== grantWs) return false;
      if (q) {
        const hay = ((row.name || "") + " " + (row.agentId || "") + " " + (row.termId || "")).toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [rows, grantQuery, grantWs]);
  const shownGrants = filteredGrants.slice(0, grantLimit);
  const clearGrantFilters = () => {
    setGrantQuery("");
    setGrantWs("all");
    setGrantLimit(GRANT_PAGE);
  };

  const stepOutcomes = useMemo(() => distinctOutcomes(steps), [steps]);
  const filteredSteps = useMemo(() => {
    const q = auditQuery.trim().toLowerCase();
    return steps.filter((s) => {
      if (auditOutcome !== "all" && String(s.outcome || "").toLowerCase() !== auditOutcome) return false;
      if (q) {
        const hay = ((s.action || "") + " " + (s.reason || "") + " " + (s.agentId || "") + " " + (s.termId || "") + " " + (s.principal || "")).toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    }).map((s) => ({ ...s, key: s.id }));
  }, [steps, auditQuery, auditOutcome]);
  const shownSteps = filteredSteps.slice(0, auditLimit);
  const auditFiltersActive = auditQuery !== "" || auditOutcome !== "all";
  const clearAuditFilters = () => {
    setAuditQuery("");
    setAuditOutcome("all");
    setAuditLimit(AUDIT_PAGE);
  };

  return (
    <PageFrame id="computer-view" title="Computer" hidden={hidden}>
      <div className="set-page">
        <h1 className="set-title">Computer</h1>
        <p className="set-lede">Let an agent use this computer: see the screen, click, type, open programs — with your own permissions.</p>
        <section className="set-group">
          <h2 className="set-grouph">Agent access</h2>
          <p className="set-groupdesc">
            <span className="set-risk"><IconWarn /> Elevated risk</span> An agent you switch on acts on your desktop as you would: any window, the clipboard, any program. Off by default; off is one click away.
          </p>
          {rows !== null && rows.length > 0 ? (
            <div className="set-toolbar" role="search">
              <input
                type="search"
                className="set-search"
                placeholder="Search agents…"
                value={grantQuery}
                onChange={(e) => { setGrantQuery(e.target.value); setGrantLimit(GRANT_PAGE); }}
                aria-label="Search agents"
              />
              <select
                className="set-select"
                value={grantWs}
                onChange={(e) => { setGrantWs(e.target.value); setGrantLimit(GRANT_PAGE); }}
                aria-label="Filter by workspace"
              >
                <option value="all">All workspaces</option>
                {grantWsOptions.map(([id, name]) => <option key={id} value={id}>{name}</option>)}
                <option value="none">No workspace</option>
              </select>
              <span className="set-count">{filteredGrants.length} of {rows.length}</span>
            </div>
          ) : null}
          <div className="set-panel">
            {rows === null ? (
              <div className="set-item"><div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div></div>
            ) : rows.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No agents yet</span>
                  <span className="set-item-d">Grants are given per agent. A CLI you start from PiCode is an agent.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={onCreateAgent}>Create an agent</button>
                </div>
              </div>
            ) : shownGrants.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No grants match</span>
                  <span className="set-item-d">No agent matches the current search and filters.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={clearGrantFilters}>Clear search</button>
                </div>
              </div>
            ) : (
              shownGrants.map((row) => (
                <Item
                  key={keyOf(row)}
                  title={<><span>{row.name}</span> <WsTag id={row.workspaceId} name={row.workspaceId ? (wsNames[row.workspaceId] || null) : null} /> <span className="devs-tag devs-tag-off">{row.kind === "terminal" ? "terminal" : "managed"}</span></>}
                  desc={row.kind === "terminal" ? "A CLI running in a PiCode terminal" : "Managed agent"}
                >
                  <SwitchCtl
                    checked={row.enabled}
                    onChange={(v) => busy !== keyOf(row) && flip(row, v)}
                    label={"Computer use for " + row.name}
                  />
                </Item>
              ))
            )}
            {filteredGrants.length > shownGrants.length ? (
              <div className="set-more">
                <span className="set-count">Showing {shownGrants.length} of {filteredGrants.length}</span>
                <button type="button" className="set-btn" onClick={() => setGrantLimit((l) => l + GRANT_PAGE)}>Show more</button>
                <button type="button" className="set-btn" onClick={() => setGrantLimit(filteredGrants.length)}>Show all</button>
              </div>
            ) : null}
          </div>
          <p className="set-groupdesc">
            A CLI started outside PiCode has no identity here and can never use the computer.
          </p>
        </section>
        <section className="set-group">
          <h2 className="set-grouph">Recent steps</h2>
          <p className="set-groupdesc">Every call an agent made, allowed or refused, newest first.</p>
          {(steps.length > 0 || auditFiltersActive) ? (
            <div className="set-toolbar" role="search">
              <input
                type="search"
                className="set-search"
                placeholder="Search steps…"
                value={auditQuery}
                onChange={(e) => { setAuditQuery(e.target.value); setAuditLimit(AUDIT_PAGE); }}
                aria-label="Search recent steps"
              />
              <select
                className="set-select"
                value={auditOutcome}
                onChange={(e) => { setAuditOutcome(e.target.value); setAuditLimit(AUDIT_PAGE); }}
                aria-label="Filter by outcome"
              >
                <option value="all">All outcomes</option>
                {stepOutcomes.map((out) => <option key={out} value={out}>{out.charAt(0).toUpperCase() + out.slice(1)}</option>)}
              </select>
              <span className="set-count">{filteredSteps.length} of {steps.length}</span>
            </div>
          ) : null}
          <div className="set-panel">
            {steps.length === 0 && !auditFiltersActive ? (
              <div className="set-item"><div className="set-item-body"><span className="set-item-d">Nothing yet.</span></div></div>
            ) : shownSteps.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No steps match</span>
                  <span className="set-item-d">No call matches the current search and filters.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={clearAuditFilters}>Clear search</button>
                </div>
              </div>
            ) : (
              <div className="set-item set-item-col">
                <div className="set-item-body">
                  <AuditList
                    rows={shownSteps}
                    method={(s) => s.action + (s.window && s.window.exe ? " · " + s.window.exe : "")}
                    outcome={(s) => s.outcome}
                    actor={(s) => s.agentId || s.termId || s.principal || "unknown"}
                    when={(s) => relTime(s.calledAt)}
                    detail={(s) => (
                      <>
                        <div className="set-audit-detail-row"><span className="set-audit-detail-k">Action</span><code>{s.action}{s.window && s.window.exe ? " · " + s.window.exe : ""}</code></div>
                        {s.reason ? <div className="set-audit-detail-row"><span className="set-audit-detail-k">Reason</span><span>{s.reason}</span></div> : null}
                        <div className="set-audit-detail-row"><span className="set-audit-detail-k">Actor</span><code>{s.agentId || s.termId || s.principal || "unknown"}</code></div>
                        <div className="set-audit-detail-row"><span className="set-audit-detail-k">Called</span><span>{new Date(s.calledAt).toLocaleString()} ({relTime(s.calledAt)})</span></div>
                      </>
                    )}
                  />
                </div>
              </div>
            )}
            {filteredSteps.length > shownSteps.length ? (
              <div className="set-more">
                <span className="set-count">Showing {shownSteps.length} of {filteredSteps.length}</span>
                <button type="button" className="set-btn" onClick={() => setAuditLimit((l) => l + AUDIT_PAGE)}>Show more</button>
                <button type="button" className="set-btn" onClick={() => setAuditLimit(filteredSteps.length)}>Show all</button>
              </div>
            ) : null}
          </div>
        </section>
      </div>
    </PageFrame>
  );
}
