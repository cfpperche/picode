import { useCallback, useEffect, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { Item, SwitchCtl } from "./settingsControls.jsx";
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
export default function ComputerPage({ hidden, onCreateAgent }) {
  const [rows, setRows] = useState(null);
  const [steps, setSteps] = useState([]);
  const [busy, setBusy] = useState(null);

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
      const r = await fetch("/api/computer/audit?limit=20");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setSteps(data.steps ?? []);
    } catch {
      setSteps([]);
    }
  }, []);

  useEffect(() => {
    if (hidden) return;
    load();
    loadSteps();
  }, [hidden, load, loadSteps]);

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
          <div className="set-panel">
            {rows === null ? (
              <div className="set-item"><div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div></div>
            ) : rows.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No agents or terminals yet</span>
                  <span className="set-item-d">Grants are given per agent, or per terminal with a CLI running in it.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={onCreateAgent}>Create an agent</button>
                </div>
              </div>
            ) : (
              rows.map((row) => (
                <Item
                  key={keyOf(row)}
                  title={row.name}
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
          </div>
          <p className="set-groupdesc">
            A <code>pi</code> started outside PiCode has no identity here and can never use the computer.
          </p>
        </section>
        <section className="set-group">
          <h2 className="set-grouph">Recent steps</h2>
          <p className="set-groupdesc">Every call an agent made, allowed or refused, newest first.</p>
          <div className="set-panel">
            {steps.length === 0 ? (
              <div className="set-item"><div className="set-item-body"><span className="set-item-d">Nothing yet.</span></div></div>
            ) : (
              <ul className="set-audit">
                {steps.map((s) => (
                  <li key={s.id} className="set-audit-row">
                    <code className="set-audit-method">{s.action}{s.window && s.window.exe ? " · " + s.window.exe : ""}</code>
                    <span className={"set-audit-outcome" + (s.outcome === "allowed" ? " is-ok" : "")}>{s.outcome}{s.reason ? " — " + s.reason : ""}</span>
                    <span className="set-audit-actor">{s.agentId || s.termId || s.principal || "unknown"}</span>
                    <span className="set-audit-when">{relTime(s.calledAt)}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      </div>
    </PageFrame>
  );
}
