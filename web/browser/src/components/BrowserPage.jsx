import { useCallback, useEffect, useState } from "react";
import PageFrame from "./PageFrame.jsx";
import { browserGrantSchema } from "@picode/shared/contracts/schemas.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { toast } from "../lib/toast.js";

// Browser (ADR-0128): the work browser's own settings surface, reached
// from the user menu. Desktop-only by nature — the menu entry renders
// only inside the desktop shell; the web /browser/ app has link-handoff
// settings of its own instead (v2).
//
// Agent grants (slice 4, ADR-0134): one row per managed agent. Read is
// the default (the tab on screen); Act and Full name the domains the
// agent may reach in its own pane. Saving goes through
// /api/browser/policy → browser.Save and announces itself on the feed.

const TIERS = [
  { value: "read", label: "Read" },
  { value: "act", label: "Act" },
  { value: "full", label: "Full" },
];

const domainsText = (row) => (row.domains ?? []).join(", ");

export default function BrowserPage({ hidden, onCreateAgent }) {
  const [rows, setRows] = useState(null); // null = first load: skeleton
  const [drafts, setDrafts] = useState({}); // agentId -> { tier, domainsText }
  const [flash, setFlash] = useState("");
  const [history, setHistory] = useState(null);
  const [confirmClear, setConfirmClear] = useState(false);
  const [prefs, setPrefs] = useState({ showFullUrl: true, webOpenDest: "app", localOpenDest: "app" });

  const load = useCallback(async () => {
    try {
      const r = await fetch("/api/browser/policies");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setRows(data.policies ?? []);
    } catch {
      toast("Could not load the browser grants.");
    }
  }, []);

  const loadHistory = useCallback(async () => {
    try {
      const r = await fetch("/api/browser/history?limit=50");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setHistory(data.visits ?? []);
    } catch {
      /* the list is disposable; the next event retries */
    }
  }, []);

  useEffect(() => {
    if (!hidden) {
      load();
      loadHistory();
      fetch("/api/browser/prefs").then((r) => r.json()).then((p) => setPrefs({ showFullUrl: p.showFullUrl !== false, webOpenDest: p.webOpenDest || "app", localOpenDest: p.localOpenDest || "app" })).catch(() => {});
    }
  }, [hidden, load, loadHistory]);

  const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;
  const [confirmWipe, setConfirmWipe] = useState(false);

  // PUT saves the editor's whole state (the endpoint replaces, not patches);
  // autofill flags are pushed to the shell so every webview picks them up.
  const setPref = (patch) => {
    const next = { ...prefs, ...patch };
    setPrefs(next);
    fetch("/api/browser/prefs", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(next) })
      .then((r) => r.json())
      .then((p) => setPrefs(p))
      .catch(() => {});
    invoke?.("btab_set_prefs", { passwordAutosave: next.passwordAutosave, generalAutofill: next.generalAutofill }).catch(() => {});
  };
  const setShowFullUrl = (on) => setPref({ showFullUrl: on });
  const DESTS = [
    { value: "app", label: "PiCode" },
    { value: "external", label: "Default browser" },
  ];
  const destRow = (label, key, sub) => (
    <div className="grant-row">
      <span className="grant-name">{label}</span>
      <span className="settings-hint" style={{ flex: 1 }}>{sub}</span>
      <select className="grant-tier" value={prefs[key]} onChange={(e) => setPref({ [key]: e.target.value })} aria-label={label}>
        {DESTS.map((d) => <option key={d.value} value={d.value}>{d.label}</option>)}
      </select>
    </div>
  );

  // Saves land as setting.updated; history mutations as browserhistory.updated.
  // Both refetch the surface they feed instead of polling.
  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "setting.updated") load();
    if (ev.type === "browserhistory.updated") loadHistory();
  }), [load, loadHistory]);

  const removeVisit = async (vid) => {
    await fetch("/api/browser/history/" + vid, { method: "DELETE" }).catch(() => {});
    loadHistory();
  };

  const clearHistory = async () => {
    if (!confirmClear) {
      setConfirmClear(true);
      setTimeout(() => setConfirmClear(false), 4000);
      return;
    }
    setConfirmClear(false);
    await fetch("/api/browser/history/clear", { method: "POST" }).catch(() => {});
    loadHistory();
  };

  // Clear browsing data: the profile is shared, so one call wipes cookies,
  // site storage and cache for every tab; our own history store clears too.
  const clearBrowsingData = async () => {
    if (!confirmWipe) {
      setConfirmWipe(true);
      setTimeout(() => setConfirmWipe(false), 4000);
      return;
    }
    setConfirmWipe(false);
    if (invoke) await invoke("btab_clear_data").catch((e) => toast("Clearing site data failed: " + (e?.message || e)));
    await fetch("/api/browser/history/clear", { method: "POST" }).catch(() => {});
    loadHistory();
    toast.ok("Browsing data cleared.");
  };

  const draftOf = (row) => drafts[row.agentId] ?? { tier: row.tier, domainsText: domainsText(row) };
  const setDraft = (agentId, next) => setDrafts((d) => ({ ...d, [agentId]: next }));

  const save = async (row) => {
    const d = draftOf(row);
    const parsed = browserGrantSchema.safeParse({ tier: d.tier, domains: d.domainsText });
    if (!parsed.success) return;
    try {
      const r = await fetch("/api/browser/policy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ agent: row.agentId, tier: parsed.data.tier, domains: parsed.data.domains }),
      });
      if (!r.ok) throw new Error((await r.json().catch(() => ({})))?.error || String(r.status));
      setFlash(row.agentId);
      setTimeout(() => setFlash((cur) => (cur === row.agentId ? "" : cur)), 2000);
      setDrafts((dd) => {
        const next = { ...dd };
        delete next[row.agentId];
        return next;
      });
    } catch (e) {
      toast("Saving the grant failed: " + (e?.message || e));
    }
  };

  return (
    <PageFrame id="browser-view" title="Browser" hidden={hidden}>
      <h4 className="settings-sub">Work browser</h4>
      <div className="settings-card">
        <p className="settings-hint">Browser tabs open inside the app and share one sign-in profile, so site logins persist on this machine. Site permissions (camera, location, downloads) are asked per site.</p>
      </div>
      <h4 className="settings-sub">Agent grants</h4>
      <div className="settings-card">
        {rows === null ? (
          <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
        ) : rows.length === 0 ? (
          <div className="mcp-empty">
            <p className="settings-hint">No managed agents yet — grants are given per agent. Sessions from the sidebar always read the tab on screen.</p>
            <button type="button" className="btn btn-primary" onClick={onCreateAgent}>Create an agent</button>
          </div>
        ) : (
          <ul className="grant-list">
            {rows.map((row) => (
              <GrantRow
                key={row.agentId}
                row={row}
                draft={draftOf(row)}
                onDraft={(next) => setDraft(row.agentId, next)}
                onSave={() => save(row)}
                flash={flash === row.agentId}
              />
            ))}
          </ul>
        )}
      </div>
      <p className="settings-hint">
        Read — sees the tab you have on screen, nothing else. Act — also opens and drives pages in its own pane, inside the domains you list. Full — everything Act does, plus developer access.
      </p>
      <h4 className="settings-sub">Open destinations</h4>
      <div className="settings-card">
        {destRow("Web links and popups", "webOpenDest", "Where new web pages open by default")}
        {destRow("Local development sites", "localOpenDest", "Where localhost and dev servers open by default")}
      </div>
      <h4 className="settings-sub">Address bar</h4>
      <div className="settings-card">
        <div className="grant-row">
          <span className="grant-name">Show full URL</span>
          <span className="settings-hint" style={{ flex: 1 }}>Include the path, query, and fragment in the address bar — off shows the site only.</span>
          <button type="button" className="btn btn-ghost" role="switch" aria-checked={prefs.showFullUrl} aria-label="Show full URL" onClick={() => setShowFullUrl(!prefs.showFullUrl)}>{prefs.showFullUrl ? "On" : "Off"}</button>
        </div>
      </div>
      <h4 className="settings-sub">Autofill and passwords</h4>
      <div className="settings-card">
        <div className="grant-row">
          <span className="grant-name">Password manager</span>
          <span className="settings-hint" style={{ flex: 1 }}>Save and fill sign-ins in the built-in browser.</span>
          <button type="button" className="btn btn-ghost" role="switch" aria-checked={prefs.passwordAutosave} aria-label="Password autosave" onClick={() => setPref({ passwordAutosave: !prefs.passwordAutosave })}>{prefs.passwordAutosave ? "On" : "Off"}</button>
        </div>
        <div className="grant-row">
          <span className="grant-name">Contact info</span>
          <span className="settings-hint" style={{ flex: 1 }}>Save and fill addresses, phone numbers, and email addresses.</span>
          <button type="button" className="btn btn-ghost" role="switch" aria-checked={prefs.generalAutofill} aria-label="General autofill" onClick={() => setPref({ generalAutofill: !prefs.generalAutofill })}>{prefs.generalAutofill ? "On" : "Off"}</button>
        </div>
      </div>
      <h4 className="settings-sub">Browsing history</h4>
      <div className="settings-card">
        {history === null ? (
          <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
        ) : history.length === 0 ? (
          <p className="settings-hint">No history yet — pages you visit in the app are listed here.</p>
        ) : (
          <>
            <div className="grant-row">
              <span className="grant-name">{history.length + (history.length === 50 ? "+" : "")} {history.length === 1 ? "page" : "pages"}</span>
              <span style={{ flex: 1 }} />
              <button type="button" className={"btn" + (confirmWipe ? " btn-danger" : "")} onClick={clearBrowsingData}>
                {confirmWipe ? "Really clear site data?" : "Clear browsing data"}
              </button>
              <button type="button" className={"btn" + (confirmClear ? " btn-danger" : "")} onClick={clearHistory}>
                {confirmClear ? "Really clear all?" : "Clear history"}
              </button>
            </div>
            <ul className="grant-list">
              {history.map((v) => (
                <li key={v.id} className="grant-row">
                  <span className="grant-name" title={v.url}>{v.title || v.host}</span>
                  <span className="grant-error" style={{ color: "var(--text-secondary)" }}>{v.host}{v.typed ? " · typed" : ""} · {new Date(v.visitedAt.replace(/\.(\d{3})\d+/, ".$1")).toLocaleString()}</span>
                  <span style={{ flex: 1 }} />
                  <button type="button" className="btn" onClick={() => removeVisit(v.id)} aria-label={"Remove " + (v.title || v.host) + " from history"}>Remove</button>
                </li>
              ))}
            </ul>
          </>
        )}
      </div>
    </PageFrame>
  );
}

function GrantRow({ row, draft, onDraft, onSave, flash }) {
  const parsed = browserGrantSchema.safeParse({ tier: draft.tier, domains: draft.domainsText });
  const valid = parsed.success;
  const dirty = draft.tier !== row.tier || draft.domainsText !== domainsText(row);
  const actLike = draft.tier !== "read";
  return (
    <li className="grant-row">
      <span className="grant-name">
        {row.name}
        {row.saved ? <span className="devs-tag">custom</span> : <span className="devs-tag devs-tag-off">default</span>}
      </span>
      <select
        className="grant-tier"
        value={draft.tier}
        onChange={(e) => onDraft({ ...draft, tier: e.target.value })}
        aria-label={"Access level for " + row.name}
      >
        {TIERS.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
      </select>
      {actLike && (
        <input
          className={"grant-domains" + (valid ? "" : " grant-domains-bad")}
          placeholder="example.com, *.example.com"
          value={draft.domainsText}
          onChange={(e) => onDraft({ ...draft, domainsText: e.target.value })}
          aria-label={"Allowed domains for " + row.name}
        />
      )}
      {!valid && actLike && <span className="grant-error">{parsed.error.issues[0].message}</span>}
      {flash ? (
        <span className="devs-tag">saved</span>
      ) : dirty && valid ? (
        <button type="button" className="btn" onClick={onSave}>Save</button>
      ) : null}
    </li>
  );
}
