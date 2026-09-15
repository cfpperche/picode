import { useCallback, useEffect, useState } from "react";
import * as Switch from "@radix-ui/react-switch";
import * as Dialog from "./ResponsiveDialog.jsx";
import PageFrame from "./PageFrame.jsx";
import { browserGrantSchema } from "@picode/shared/contracts/schemas.js";
import { subscribeFeed } from "@picode/shared/client/feed.js";
import { toast } from "../lib/toast.js";

// Browser (ADR-0128): the work browser's own settings surface, reached
// from the user menu. Desktop-only by nature — the menu entry renders
// only inside the desktop shell; the web /browser/ app has link-handoff
// settings of its own instead (v2).
//
// The page follows the ChatGPT Work Browser settings shape the owner
// asked for (2026-09-14): a title and lede, section headers, and cards of
// rows that carry a title + description on the left with the control
// pinned right, divided by hairlines. The chrome around the page stays
// PiCode's. `.set-page` and friends in styles/app.css are that shape.
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

const DESTS = [
  { value: "app", label: "PiCode" },
  { value: "external", label: "Default browser" },
];

const domainsText = (row) => (row.domains ?? []).join(", ");

// One row of a section card: title + description left, control right.
function Item({ title, desc, children }) {
  return (
    <div className="set-item">
      <div className="set-item-body">
        <span className="set-item-t">{title}</span>
        {desc ? <span className="set-item-d">{desc}</span> : null}
      </div>
      <div className="set-item-ctl">{children}</div>
    </div>
  );
}

function SwitchCtl({ checked, onChange, label }) {
  return (
    <Switch.Root className="rx-switch" checked={!!checked} onCheckedChange={onChange} aria-label={label}>
      <Switch.Thumb className="rx-switch-thumb" />
    </Switch.Root>
  );
}

export default function BrowserPage({ hidden, onCreateAgent }) {
  const [rows, setRows] = useState(null); // null = first load: skeleton
  const [drafts, setDrafts] = useState({}); // agentId -> { tier, domainsText }
  const [flash, setFlash] = useState("");
  const [history, setHistory] = useState(null);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const [confirmWipe, setConfirmWipe] = useState(false);
  const [prefs, setPrefs] = useState({ showFullUrl: true, webOpenDest: "app", localOpenDest: "app", passwordAutosave: true, generalAutofill: true, agentAccess: true });

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
      fetch("/api/browser/prefs")
        .then((r) => r.json())
        .then((p) => setPrefs({
          showFullUrl: p.showFullUrl !== false,
          webOpenDest: p.webOpenDest || "app",
          localOpenDest: p.localOpenDest || "app",
          passwordAutosave: p.passwordAutosave !== false,
          generalAutofill: p.generalAutofill !== false,
          agentAccess: p.agentAccess !== false,
        }))
        .catch(() => {});
    }
  }, [hidden, load, loadHistory]);

  const invoke = typeof window !== "undefined" && window.__TAURI__ ? window.__TAURI__.core.invoke : null;

  // PUT saves the editor's whole state (the endpoint replaces, not patches);
  // autofill flags are pushed to the shell so every webview picks them up.
  const setPref = (patch) => {
    const next = { ...prefs, ...patch };
    setPrefs(next);
    fetch("/api/browser/prefs", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(next) })
      .then((r) => r.json())
      .then((p) => setPrefs({
        showFullUrl: p.showFullUrl !== false,
        webOpenDest: p.webOpenDest || "app",
        localOpenDest: p.localOpenDest || "app",
        passwordAutosave: p.passwordAutosave !== false,
        generalAutofill: p.generalAutofill !== false,
        agentAccess: p.agentAccess !== false,
      }))
      .catch(() => {});
    invoke?.("btab_set_prefs", { passwordAutosave: next.passwordAutosave, generalAutofill: next.generalAutofill }).catch(() => {});
  };

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
      <div className="set-page">
        <div>
          <h1 className="set-title">Browser</h1>
          <p className="set-lede">Manage your work browser preferences and site access.</p>
        </div>

        <section className="set-group">
          <div className="set-panel">
            <Item title="Browser" desc="Let agents use the built-in browser">
              <SwitchCtl
                checked={prefs.agentAccess}
                onChange={(v) => setPref({ agentAccess: v })}
                label="Let agents use the built-in browser"
              />
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">General</h2>
          <div className="set-panel">
            <Item title="Web URL and link open destination" desc="Where links open by default">
              <select className="set-select" value={prefs.webOpenDest} onChange={(e) => setPref({ webOpenDest: e.target.value })} aria-label="Web URL and link open destination">
                {DESTS.map((d) => <option key={d.value} value={d.value}>{d.label}</option>)}
              </select>
            </Item>
            <Item title="Local URL open destination" desc="Where local development sites open by default">
              <select className="set-select" value={prefs.localOpenDest} onChange={(e) => setPref({ localOpenDest: e.target.value })} aria-label="Local URL open destination">
                {DESTS.map((d) => <option key={d.value} value={d.value}>{d.label}</option>)}
              </select>
            </Item>
            <Item title="Show full URL" desc="Include the path, query, and fragment in the address bar">
              <SwitchCtl checked={prefs.showFullUrl} onChange={(v) => setPref({ showFullUrl: v })} label="Show full URL" />
            </Item>
            <Item title="Browsing data" desc="Clear browsing history, site data, cache, and download history from the in-app browser">
              <button
                type="button"
                className={"set-btn" + (confirmWipe ? " set-btn-danger" : "")}
                onClick={clearBrowsingData}
              >
                {confirmWipe ? "Really clear site data?" : "Clear browsing data"}
              </button>
            </Item>
            <Item title="Browsing history" desc="View and manage pages visited in the built-in browser">
              <button type="button" className="set-btn" onClick={() => setHistoryOpen(true)}>Manage</button>
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Autofill and passwords</h2>
          <div className="set-panel">
            <Item title="Password manager" desc="Save and fill sign-ins in the built-in browser">
              <SwitchCtl checked={prefs.passwordAutosave} onChange={(v) => setPref({ passwordAutosave: v })} label="Password autosave" />
            </Item>
            <Item title="Contact info" desc="Save and fill addresses, phone numbers, and email addresses">
              <SwitchCtl checked={prefs.generalAutofill} onChange={(v) => setPref({ generalAutofill: v })} label="General autofill" />
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Agent permissions</h2>
          <p className="set-groupdesc">Which agents may use the built-in browser, and on which sites.</p>
          <div className="set-panel">
            {rows === null ? (
              <div className="set-item"><div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div></div>
            ) : rows.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No managed agents yet</span>
                  <span className="set-item-d">Grants are given per agent; sessions from the sidebar always read the tab on screen.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={onCreateAgent}>Create an agent</button>
                </div>
              </div>
            ) : (
              rows.map((row) => (
                <GrantRow
                  key={row.agentId}
                  row={row}
                  draft={draftOf(row)}
                  onDraft={(next) => setDraft(row.agentId, next)}
                  onSave={() => save(row)}
                  flash={flash === row.agentId}
                />
              ))
            )}
          </div>
          <p className="set-groupdesc">
            Read — sees the tab you have on screen, nothing else. Act — also opens and drives pages in its own pane, inside the domains you list. Full — everything Act does, plus developer access.
          </p>
        </section>
      </div>

      <Dialog.Root open={historyOpen} onOpenChange={setHistoryOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg">
            <Dialog.Title className="dlg-title">Browsing history</Dialog.Title>
            <Dialog.Description className="dlg-body">
              Pages visited in the built-in browser, newest first.
            </Dialog.Description>
            {history === null ? (
              <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
            ) : history.length === 0 ? (
              <p className="set-groupdesc" style={{ marginTop: 12 }}>No history yet — pages you visit in the app are listed here.</p>
            ) : (
              <ul className="set-hist">
                {history.map((v) => (
                  <li key={v.id}>
                    <span className="set-hist-t" title={v.url}>{v.title || v.host}</span>
                    <span style={{ color: "var(--text-secondary)", flex: "none" }}>{v.host}</span>
                    <span style={{ color: "var(--text-secondary)", flex: "none" }}>
                      {new Date(v.visitedAt.replace(/\.(\d{3})\d+/, ".$1")).toLocaleDateString()}
                    </span>
                    <button type="button" className="set-btn" onClick={() => removeVisit(v.id)} aria-label={"Remove " + (v.title || v.host) + " from history"}>Remove</button>
                  </li>
                ))}
              </ul>
            )}
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setHistoryOpen(false)}>Close</button>
              <button
                type="button"
                className={"btn btn-sm" + (confirmClear ? " btn-danger" : "")}
                onClick={clearHistory}
                disabled={!history || history.length === 0}
              >
                {confirmClear ? "Really clear all?" : "Clear history"}
              </button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
    </PageFrame>
  );
}

function GrantRow({ row, draft, onDraft, onSave, flash }) {
  const parsed = browserGrantSchema.safeParse({ tier: draft.tier, domains: draft.domainsText });
  const valid = parsed.success;
  const dirty = draft.tier !== row.tier || draft.domainsText !== domainsText(row);
  const actLike = draft.tier !== "read";
  return (
    <div className="set-item">
      <div className="set-item-body">
        <span className="set-item-t">
          {row.name}
          {row.saved ? <span className="devs-tag">custom</span> : <span className="devs-tag devs-tag-off">default</span>}
        </span>
        {!valid && actLike ? <span className="set-item-d" style={{ color: "var(--danger)" }}>{parsed.error.issues[0].message}</span> : null}
      </div>
      <div className="set-item-ctl">
        <select
          className="set-select"
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
        {flash ? (
          <span className="devs-tag">saved</span>
        ) : dirty && valid ? (
          <button type="button" className="set-btn" onClick={onSave}>Save</button>
        ) : null}
      </div>
    </div>
  );
}
