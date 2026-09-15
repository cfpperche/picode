import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
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

// Clear browsing data: the time ranges and the kinds the dialog offers.
// Each kind names the WebView2 browsing-data mask half the shell applies;
// the other half is our own history store (only "history" writes there).
const WIPE_RANGES = [
  { seconds: 3600, label: "Last hour" },
  { seconds: 86400, label: "Last 24 hours" },
  { seconds: 604800, label: "Last 7 days" },
  { seconds: 2419200, label: "Last 4 weeks" },
  { seconds: 0, label: "All time" },
];

const WIPE_KINDS = [
  { value: "history", title: "Browsing history", on: true },
  { value: "cookies", title: "Cookies and site data", d: "Cookies and other site storage", on: true },
  { value: "cache", title: "Cached images and files", d: "Images and files kept so pages load faster", on: true },
  { value: "downloads", title: "Download history", d: "No downloads", on: true },
  { value: "autofill", title: "Autofill form data", d: "Saved names, addresses, and other form entries", on: false },
  { value: "siteSettings", title: "Site settings", d: "Website permissions and preferences", on: false },
];

// 16px stroke icons for the wipe list — inline so no icon package is
// needed for six glyphs.
const WIPE_ICONS = {
  history: <><circle cx="8" cy="8" r="6" /><path d="M2 8h12M8 2c2 2 2 10 0 12M8 2c-2 2-2 10 0 12" /></>,
  cookies: <><circle cx="8" cy="8" r="6" /><circle cx="6" cy="6" r=".9" fill="currentColor" stroke="none" /><circle cx="10" cy="7" r=".9" fill="currentColor" stroke="none" /><circle cx="7.5" cy="10.5" r=".9" fill="currentColor" stroke="none" /></>,
  cache: <><rect x="2" y="3" width="12" height="10" rx="1.5" /><circle cx="6" cy="7" r="1.2" /><path d="M2.5 11.5l3.2-3 2.4 2.2 2.3-2.2 3.1 3" /></>,
  downloads: <><path d="M8 2v8" /><path d="M5 7.5L8 10.5 11 7.5" /><path d="M2.5 13h11" /></>,
  autofill: <><path d="M10.5 2.5l3 3-7.5 7.5-3.6.6.6-3.6z" /></>,
  siteSettings: <><circle cx="8" cy="8" r="2.4" /><path d="M8 1.5v2M8 12.5v2M1.5 8h2M12.5 8h2M3.4 3.4l1.4 1.4M11.2 11.2l1.4 1.4M12.6 3.4l-1.4 1.4M4.8 11.2l-1.4 1.4" /></>,
};

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

// A visit's timestamp arrives as RFC3339 with nanoseconds; Date wants at
// most milliseconds.
const visitDate = (v) => new Date(v.visitedAt.replace(/\.(\d{3})\d+/, ".$1"));

const dayKey = (d) => d.toDateString();

const dayLabel = (d) => {
  const today = new Date();
  const yesterday = new Date(today.getTime() - 86400000);
  if (dayKey(d) === dayKey(today)) return "Today";
  if (dayKey(d) === dayKey(yesterday)) return "Yesterday";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
};

const timeLabel = (d) => d.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });

// The row's site icon: the site's own /favicon.ico over https, and the
// host's first letter when that is impossible (http, or no icon).
function Favicon({ url, host }) {
  const [bad, setBad] = useState(false);
  let origin = "";
  try {
    const parsed = new URL(url);
    if (parsed.protocol === "https:") origin = parsed.origin;
  } catch {
    origin = "";
  }
  if (origin && !bad) {
    return (
      <span className="hist-fav" aria-hidden="true">
        <img src={origin + "/favicon.ico"} alt="" loading="lazy" onError={() => setBad(true)} />
      </span>
    );
  }
  return <span className="hist-fav" aria-hidden="true">{host.slice(0, 1)}</span>;
}

export default function BrowserPage({ hidden, onCreateAgent }) {
  const [rows, setRows] = useState(null); // null = first load: skeleton
  const [drafts, setDrafts] = useState({}); // agentId -> { tier, domainsText }
  const [flash, setFlash] = useState("");
  const [history, setHistory] = useState(null);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [histQuery, setHistQuery] = useState("");
  const [openDays, setOpenDays] = useState(() => new Set());
  const [selected, setSelected] = useState(() => new Set());
  const [histLoading, setHistLoading] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const [wipeOpen, setWipeOpen] = useState(false);
  const [wipeRange, setWipeRange] = useState(3600);
  const [wipeKinds, setWipeKinds] = useState(() => WIPE_KINDS.filter((k) => k.on).map((k) => k.value));
  const [wipeBusy, setWipeBusy] = useState(false);
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

  const histQueryRef = useRef("");
  const loadHistory = useCallback(async () => {
    setHistLoading(true);
    try {
      const q = histQueryRef.current;
      const r = await fetch("/api/browser/history?limit=200" + (q ? "&q=" + encodeURIComponent(q) : ""));
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setHistory(data.visits ?? []);
    } catch {
      /* the list is disposable; the next event retries */
    } finally {
      setHistLoading(false);
    }
  }, []);

  // The search box filters server-side; 250ms of quiet is enough to ask.
  useEffect(() => {
    histQueryRef.current = histQuery;
    const t = setTimeout(() => {
      if (!hidden && historyOpen) loadHistory();
    }, histQuery ? 250 : 0);
    return () => clearTimeout(t);
  }, [histQuery, hidden, historyOpen, loadHistory]);

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

  // Clear browsing data: the profile is shared, so one call reaches every
  // tab; the shell takes the kinds the dialog checked and, for a range, how
  // far back to start. Our own history store clears only when the dialog
  // asked for history (and honors the same range).
  const runWipe = async () => {
    if (wipeKinds.length === 0 || wipeBusy) return;
    setWipeBusy(true);
    const since = wipeRange > 0 ? Math.floor(Date.now() / 1000) - wipeRange : null;
    let failed = "";
    if (invoke) {
      await invoke("btab_clear_data", { kinds: wipeKinds, since }).catch((e) => { failed = String(e?.message || e); });
    }
    if (wipeKinds.includes("history")) {
      const q = since ? "?since=" + encodeURIComponent(new Date(since * 1000).toISOString()) : "";
      await fetch("/api/browser/history/clear" + q, { method: "POST" }).catch(() => {});
      loadHistory();
    }
    setWipeBusy(false);
    setWipeOpen(false);
    if (failed) toast("Clearing site data failed: " + failed);
    else toast.ok("Browsing data cleared.");
  };

  // The history count the dialog shows for the picked range, from the
  // loaded list (capped at 50 — a full list reads as "50+").
  const historyInRange = () => {
    if (!Array.isArray(history)) return null;
    if (wipeRange === 0) return { n: history.length, capped: history.length >= 50 };
    const cutoff = Date.now() - wipeRange * 1000;
    const n = history.filter((v) => new Date(v.visitedAt.replace(/\.(\d{3})\d+/, ".$1")).getTime() >= cutoff).length;
    return { n, capped: history.length >= 50 && n === history.length };
  };

  // One group per day, newest first (the API already orders that way).
  const histGroups = useMemo(() => {
    if (!Array.isArray(history)) return [];
    const byDay = new Map();
    for (const v of history) {
      const d = visitDate(v);
      const key = dayKey(d);
      if (!byDay.has(key)) byDay.set(key, { key, label: dayLabel(d), items: [] });
      byDay.get(key).items.push({ ...v, when: d });
    }
    return [...byDay.values()];
  }, [history]);

  // The newest day starts open; a search opens them all so nothing hides.
  useEffect(() => {
    setOpenDays((cur) => {
      if (histQuery) return new Set(histGroups.map((g) => g.key));
      if (cur.size) return cur;
      return new Set(histGroups.slice(0, 1).map((g) => g.key));
    });
  }, [histGroups, histQuery]);

  const toggleDay = (key) => setOpenDays((cur) => {
    const next = new Set(cur);
    if (next.has(key)) next.delete(key);
    else next.add(key);
    return next;
  });

  const toggleSelected = (id) => setSelected((cur) => {
    const next = new Set(cur);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    return next;
  });

  const removeSelected = async () => {
    const ids = [...selected];
    if (!ids.length) return;
    try {
      await fetch("/api/browser/history/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids }),
      });
    } catch {
      toast("Removing those visits failed.");
      return;
    }
    setSelected(new Set());
    loadHistory();
  };

  const copyLink = async (url) => {
    try {
      await navigator.clipboard.writeText(url);
      toast.ok("Link copied.");
    } catch {
      toast("Could not copy the link.");
    }
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
              <button type="button" className="set-btn" onClick={() => setWipeOpen(true)}>Clear browsing data</button>
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

      <Dialog.Root open={historyOpen} onOpenChange={(open) => { setHistoryOpen(open); if (!open) { setSelected(new Set()); setHistQuery(""); } }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-hist">
            <Dialog.Title className="dlg-title">Browsing history</Dialog.Title>
            <input
              className="hist-search"
              type="search"
              placeholder="Search browsing history"
              value={histQuery}
              onChange={(e) => setHistQuery(e.target.value)}
              aria-label="Search browsing history"
            />
            <div className="hist-head">
              <span className="hist-head-t">{histQuery ? `Results for “${histQuery}”` : "All-time history"}</span>
              <button
                type="button"
                className="set-btn"
                onClick={() => { setHistoryOpen(false); setWipeOpen(true); }}
              >
                Clear browsing data
              </button>
            </div>
            {history === null || (histLoading && histGroups.length === 0) ? (
              <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
            ) : histGroups.length === 0 ? (
              <p className="set-groupdesc">
                {histQuery
                  ? `No pages match “${histQuery}”.`
                  : "No history yet — pages you visit in the app are listed here."}
              </p>
            ) : (
              <div className="hist-days">
                {histGroups.map((g) => {
                  const open = openDays.has(g.key);
                  return (
                    <div className="hist-day" key={g.key}>
                      <button type="button" className="hist-day-h" aria-expanded={open} onClick={() => toggleDay(g.key)}>
                        <span>{g.label}</span>
                        <span className="hist-day-n">{g.items.length}</span>
                        <svg width="10" height="6" viewBox="0 0 10 6" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" aria-hidden="true">
                          <path d="M1 1l4 4 4-4" />
                        </svg>
                      </button>
                      {open && (
                        <ul className="hist-rows">
                          {g.items.map((v) => (
                            <li className="hist-row" key={v.id}>
                              <input
                                type="checkbox"
                                checked={selected.has(v.id)}
                                onChange={() => toggleSelected(v.id)}
                                aria-label={"Select " + (v.title || v.host)}
                              />
                              <Favicon url={v.url} host={v.host} />
                              <span className="hist-main">
                                <span className="hist-t" title={v.url}>{v.title || v.host}</span>
                                <span className="hist-host">{v.host}{v.typed ? " · typed" : ""}</span>
                              </span>
                              <span className="hist-time">{timeLabel(v.when)}</span>
                              <DropdownMenu.Root>
                                <DropdownMenu.Trigger className="hist-more" aria-label={"More actions for " + (v.title || v.host)}>⋯</DropdownMenu.Trigger>
                                <DropdownMenu.Portal>
                                  <DropdownMenu.Content className="um-popover" align="end" sideOffset={4} collisionPadding={8}>
                                    <DropdownMenu.Item className="um-item" onSelect={() => copyLink(v.url)}>Copy link</DropdownMenu.Item>
                                    <DropdownMenu.Item className="um-item" onSelect={() => removeVisit(v.id)}>Remove from history</DropdownMenu.Item>
                                  </DropdownMenu.Content>
                                </DropdownMenu.Portal>
                              </DropdownMenu.Root>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
            <div className="dlg-actions" data-align-row>
              {selected.size > 0 ? (
                <>
                  <span className="hist-sel">{selected.size} selected</span>
                  <button type="button" className="btn btn-sm btn-danger" onClick={removeSelected}>Remove</button>
                </>
              ) : null}
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setHistoryOpen(false)}>Close</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <Dialog.Root open={wipeOpen} onOpenChange={setWipeOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-wipe">
            <Dialog.Title className="dlg-title">Clear browsing data</Dialog.Title>
            <div className="wipe-pills" role="group" aria-label="Time range">
              {WIPE_RANGES.map((r) => (
                <button
                  key={r.seconds}
                  type="button"
                  className="wipe-pill"
                  aria-pressed={wipeRange === r.seconds}
                  onClick={() => setWipeRange(r.seconds)}
                >
                  {r.label}
                </button>
              ))}
            </div>
            <div className="wipe-list">
              {WIPE_KINDS.map((k) => {
                const range = k.value === "history" ? historyInRange() : null;
                const desc = k.value === "history"
                  ? (range === null ? "Pages visited in the app" : range.n === 0 ? "No sites visited" : `${range.n}${range.capped ? "+" : ""} page${range.n === 1 && !range.capped ? "" : "s"} visited`)
                  : k.d;
                return (
                  <label key={k.value} className="wipe-row">
                    <span className="wipe-ico" aria-hidden="true">
                      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round">{WIPE_ICONS[k.value]}</svg>
                    </span>
                    <span className="wipe-body">
                      <span className="wipe-t">{k.title}</span>
                      <span className="wipe-d">{desc}</span>
                    </span>
                    <input
                      type="checkbox"
                      checked={wipeKinds.includes(k.value)}
                      onChange={(e) => setWipeKinds((cur) => e.target.checked ? [...cur, k.value] : cur.filter((v) => v !== k.value))}
                      aria-label={k.title}
                    />
                  </label>
                );
              })}
            </div>
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setWipeOpen(false)}>Cancel</button>
              <button
                type="button"
                className="btn btn-sm btn-primary"
                onClick={runWipe}
                disabled={wipeKinds.length === 0 || wipeBusy}
              >
                {wipeBusy ? "Deleting…" : "Delete data"}
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
