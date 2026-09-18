import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import * as Dialog from "./ResponsiveDialog.jsx";
import FolderPicker from "./FolderPicker.jsx";
import { DEFAULT_BROWSER_PREFS, readBrowserPrefs } from "../lib/browserPrefs.js";
import { openWindowsSignIn, shellInvoke } from "../lib/windowsSettings.js";
import { describeDomainField } from "../lib/browserDomains.js";
import { ALL_SITES, permissionPush } from "../lib/browserPermissions.js";
import { takeBrowserDialog } from "../lib/browserDialogs.js";
import { IconWarn } from "./Icons.jsx";
import { relTime } from "@picode/shared/domain/relTime.js";
import PageFrame from "./PageFrame.jsx";
import { Item, SwitchCtl } from "./settingsControls.jsx";
import { BROWSER_PERMISSION_KINDS, browserGrantSchema, browserSiteSchema } from "@picode/shared/contracts/schemas.js";
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

// The kinds the dialog offers, in the order the reference lists them. The
// value is the daemon's own vocabulary (the shell maps the platform's kinds
// onto it); "*" as the origin is the standing for every site — that is what
// the per-kind allow/block choice writes.
const PERMISSION_KINDS = [
  { value: "camera", title: "Camera", d: "Sites can ask to use your camera" },
  { value: "microphone", title: "Microphone", d: "Sites can ask to use your microphone" },
  { value: "location", title: "Location", d: "Sites can ask for your location" },
  { value: "notifications", title: "Notifications", d: "Sites can ask to send notifications" },
  { value: "clipboard", title: "Clipboard", d: "Sites can ask to read your clipboard" },
  { value: "autoplay", title: "Autoplay", d: "Sites can start playing media" },
];

const bytesLabel = (n) => {
  if (!n || n <= 0) return "";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = n;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value >= 10 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`;
};

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

// The browser writes on Windows; the folder picker browses the daemon's own
// tree, where a Windows drive is /mnt/<letter>. The two name the same place.
const toWindowsPath = (p) => {
  const m = /^\/mnt\/([a-z])(\/.*)?$/.exec(p || "");
  if (!m) return "";
  const rest = (m[2] || "").split("/").filter(Boolean).join("\\");
  return m[1].toUpperCase() + ":\\" + rest;
};

const toPickerPath = (p) => {
  const m = /^([A-Za-z]):[\\/]?(.*)$/.exec(p || "");
  if (!m) return "";
  const rest = (m[2] || "").split(/[\\/]/).filter(Boolean).join("/");
  return "/mnt/" + m[1].toLowerCase() + "/" + rest;
};

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
  const [manageOpen, setManageOpen] = useState(""); // "" | "passwords" | "contact"
  const [confirmPurge, setConfirmPurge] = useState(false);
  const [purgeBusy, setPurgeBusy] = useState(false);
  const [downloadDir, setDownloadDir] = useState("");
  const [dirOpen, setDirOpen] = useState(false);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [downloads, setDownloads] = useState(null);
  const [downloadsOpen, setDownloadsOpen] = useState(false);
  const [dlQuery, setDlQuery] = useState("");
  const [confirmClearAll, setConfirmClearAll] = useState(false);
  const [siteOpen, setSiteOpen] = useState(false);
  const [stands, setStands] = useState(null);
  const [confirmClear, setConfirmClear] = useState(false);
  const [wipeOpen, setWipeOpen] = useState(false);
  const [wipeRange, setWipeRange] = useState(3600);
  const [wipeKinds, setWipeKinds] = useState(() => WIPE_KINDS.filter((k) => k.on).map((k) => k.value));
  const [wipeBusy, setWipeBusy] = useState(false);
  const [prefs, setPrefs] = useState(DEFAULT_BROWSER_PREFS);
  // The dialog's "+ Add" (the reference's own affordance): author an exception
  // for a site instead of waiting for that site to ask.
  const [newSite, setNewSite] = useState({ site: "", kind: "camera", decision: "allow" });
  const [newSiteError, setNewSiteError] = useState("");
  const [addingSite, setAddingSite] = useState(false);
  // The raw-CDP audit (ADR-0144): null while loading, [] when empty, so the
  // card can tell "nothing yet" from "not read yet".
  const [devAudit, setDevAudit] = useState(null);

  // The work-tab options menu asks for one of these dialogs (option A, owner
  // 2026-09-15): the request rode the route change, and this page opens it
  // as soon as it is on screen.
  useEffect(() => {
    if (hidden) return;
    const want = takeBrowserDialog();
    if (want === "history") setHistoryOpen(true);
    else if (want === "downloads") setDownloadsOpen(true);
    else if (want === "wipe") setWipeOpen(true);
    else if (want === "passwords" || want === "contact") setManageOpen(want);
  }, [hidden]);

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
  const dlQueryRef = useRef("");
  const loadDownloads = useCallback(async () => {
    try {
      const q = dlQueryRef.current;
      const r = await fetch("/api/browser/downloads?limit=200" + (q ? "&q=" + encodeURIComponent(q) : ""));
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setDownloads(data.downloads ?? []);
    } catch {
      /* the list is disposable; the next event retries */
    }
  }, []);

  const loadStands = useCallback(async () => {
    try {
      const r = await fetch("/api/browser/permissions?limit=200");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setStands(data.permissions ?? []);
    } catch {
      /* disposable; the next event retries */
    }
  }, []);

  // The audit is what makes Developer mode defensible: every raw CDP call,
  // allowed or refused, newest first (ADR-0144).
  const loadDevAudit = useCallback(async () => {
    try {
      const r = await fetch("/api/browser/developer/audit?limit=20");
      if (!r.ok) throw new Error(String(r.status));
      const data = await r.json();
      setDevAudit(data.calls ?? []);
    } catch {
      setDevAudit([]);
    }
  }, []);

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
    dlQueryRef.current = dlQuery;
    const t = setTimeout(() => {
      if (!hidden && downloadsOpen) loadDownloads();
    }, dlQuery ? 250 : 0);
    return () => clearTimeout(t);
  }, [dlQuery, hidden, downloadsOpen, loadDownloads]);

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
      loadDownloads();
      loadStands();
      loadDevAudit();
      if (invoke) invoke("btab_download_dir").then((p) => setDownloadDir(p || "")).catch(() => {});
      fetch("/api/browser/prefs")
        .then((r) => r.json())
        .then((p) => {
          const next = readBrowserPrefs(p);
          setPrefs(next);
          // The shell keeps its own copy of the switches it applies per page
          // (JavaScript on creation, developer mode at every raw CDP call).
          // Push on load as well as on save: after an app restart the shell's
          // copy is at its default until something tells it otherwise, and a
          // settings page that only pushes on save leaves that gap until the
          // next click.
          invoke?.("btab_set_scripts", { enabled: next.scriptsEnabled !== false }).catch(() => {});
          invoke?.("btab_set_developer_mode", { enabled: next.developerMode === true }).catch(() => {});
        })
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
      .then((p) => setPrefs(readBrowserPrefs(p)))
      .catch(() => {});
    // The shell is the half that applies these to the pages; a refusal here
    // (ACL, no shell) must be visible, not a switch that quietly does nothing.
    invoke?.("btab_set_prefs", { passwordAutosave: next.passwordAutosave, generalAutofill: next.generalAutofill })
      .catch((e) => toast("The app did not accept the autofill settings: " + (e?.message || e)));
    invoke?.("btab_set_ask_download", { ask: !!next.askDownload })
      .catch((e) => toast("The app did not accept the download setting: " + (e?.message || e)));
    invoke?.("btab_set_scripts", { enabled: next.scriptsEnabled !== false })
      .catch((e) => toast("The app did not accept the JavaScript setting: " + (e?.message || e)));
    invoke?.("btab_set_developer_mode", { enabled: next.developerMode === true })
      .catch((e) => toast("The app did not accept the developer mode setting: " + (e?.message || e)));
  };

  // Saves land as setting.updated; history mutations as browserhistory.updated.
  // Both refetch the surface they feed instead of polling.
  // The shell keeps the policy in memory; the rows in the daemon are the
  // truth, so every load (and every feed event) hands back the policy the
  // shell is allowed to know: the every-site rows, and a site row only when
  // it is a standing (a one-off decision must not become permanent).
  useEffect(() => {
    if (!invoke || !Array.isArray(stands)) return;
    for (const st of stands) {
      const push = permissionPush(st);
      if (push) invoke("btab_set_permission_policy", push).catch(() => {});
    }
  }, [stands, invoke]);

  useEffect(() => subscribeFeed((ev) => {
    if (ev.type === "setting.updated") load();
    if (ev.type === "browserpermission.updated") loadStands();
    if (ev.type === "browserhistory.updated") loadHistory();
    if (ev.type === "browserdownload.updated") loadDownloads();
    // The audit refetches on its own events: a raw call the owner is watching
    // for shows up while the page is open (ADR-0144).
    if (ev.type === "browser.cdp") loadDevAudit();
  }), [load, loadHistory, loadDownloads]);

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

  // Only the browser profile's own saved data can be cleared here: WebView2
  // exposes the autosave switches and a wipe by kind, and nothing that
  // lists entries. The line in the dialog says exactly that.
  const purgeSaved = async (kind) => {
    if (!confirmPurge) {
      setConfirmPurge(true);
      setTimeout(() => setConfirmPurge(false), 4000);
      return;
    }
    setConfirmPurge(false);
    setPurgeBusy(true);
    let failed = "";
    if (invoke) {
      await invoke("btab_clear_data", { kinds: [kind], since: null }).catch((e) => { failed = String(e?.message || e); });
    }
    setPurgeBusy(false);
    if (failed) toast("Deleting the saved data failed: " + failed);
    else toast.ok(kind === "passwords" ? "Saved passwords deleted." : "Saved form data deleted.");
  };

  const saveDownloadDir = async (path) => {
    try {
      if (invoke) await invoke("btab_set_download_dir", { path });
    } catch (e) {
      toast("Changing the download folder failed: " + (e?.message || e));
      return;
    }
    setDownloadDir(path);
    setDirOpen(false);
    toast.ok(path ? "Downloads will be saved to " + path : "Downloads go to the system Downloads folder.");
  };

  // A picked path only counts when it names a Windows drive: the browser
  // cannot write into the Linux side of the tree.
  const pickDownloadDir = (picked) => {
    const win = toWindowsPath(picked);
    if (!win) {
      toast("Choose a folder inside a Windows drive (C:, E:, …) — the browser writes on Windows.");
      return;
    }
    setPickerOpen(false);
    saveDownloadDir(win);
  };

  const removeDownload = async (id) => {
    await fetch("/api/browser/downloads/" + id, { method: "DELETE" }).catch(() => {});
    loadDownloads();
  };

  const clearDownloads = async () => {
    if (!confirmClearAll) {
      setConfirmClearAll(true);
      setTimeout(() => setConfirmClearAll(false), 4000);
      return;
    }
    setConfirmClearAll(false);
    await fetch("/api/browser/downloads/clear", { method: "POST" }).catch(() => {});
    loadDownloads();
  };

  const openDownload = async (d, reveal) => {
    if (!invoke) return;
    try {
      await invoke(reveal ? "btab_reveal_path" : "btab_open_path", { path: d.path });
    } catch (e) {
      toast("Opening the file failed: " + (e?.message || e));
    }
  };

  const policyOf = (kind) => (stands || []).find((st) => st.origin === ALL_SITES && st.kind === kind)?.decision || "";

  // The dialog's per-kind choice writes the every-site standing. Platform
  // default forgets that row (only that row — a site's remembered answers
  // are not the kind policy), and the live shell drops the entry so the
  // platform default takes over now, not at the next restart.
  const setPolicy = async (kind, decision) => {
    const row = (stands || []).find((st) => st.origin === ALL_SITES && st.kind === kind);
    try {
      if (!decision) {
        if (row) {
          const r = await fetch("/api/browser/permissions/" + row.id, { method: "DELETE" });
          if (!r.ok && r.status !== 404) throw new Error(String(r.status));
        }
      } else {
        const r = await fetch("/api/browser/permissions", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ origin: ALL_SITES, kind, decision, standing: true }),
        });
        if (!r.ok) throw new Error(String(r.status));
      }
    } catch {
      toast("Saving the permission failed.");
      return;
    }
    if (invoke) await invoke("btab_set_permission_policy", { kind, state: decision || "default", origin: null }).catch(() => {});
    loadStands();
  };

  // Author one exception (the reference's "+ Add"): the store already owns
  // the vocabulary and the shell already applies standings pushed from this
  // page, so this is only the missing way to write a row by hand.
  const addSite = async () => {
    const parsed = browserSiteSchema.safeParse(newSite);
    if (!parsed.success) {
      setNewSiteError(parsed.error.issues[0].message);
      return;
    }
    setNewSiteError("");
    setAddingSite(true);
    try {
      const res = await fetch("/api/browser/permissions", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        // The store's field names: `origin` is the site or pattern the row
        // keys on (the dialog's word is "site").
        body: JSON.stringify({
          origin: parsed.data.site,
          kind: parsed.data.kind,
          decision: parsed.data.decision,
          standing: true,
        }),
      });
      if (!res.ok) {
        const why = (await res.text()).replace(/^store:\s*/, "").trim();
        throw new Error(why || `the daemon answered ${res.status}`);
      }
      setNewSite({ site: "", kind: parsed.data.kind, decision: parsed.data.decision });
      loadStands();
    } catch (e) {
      setNewSiteError("Could not save it — " + (e?.message || "the daemon is not reachable"));
    } finally {
      setAddingSite(false);
    }
  };

  // Reset forgets one row and tells the live shell to drop the matching
  // entry, so a reset takes effect now instead of at the next restart.
  // The one operating-system screen PiCode opens on purpose: Windows keeps
  // the face/finger/PIN and the passkeys, and we say so instead of imitating
  // it. Hidden outside the shell — a plain browser has no such screen.
  const openSignIn = async () => {
    const outcome = await openWindowsSignIn(window);
    if (outcome === "asked") toast.ok("Asked Windows to open Sign-in options.");
    else if (outcome === "failed") toast("Windows did not open that screen.");
  };

  // "Forget unused": rows whose site has not been visited in 90 days. The
  // store forgets them and answers with what it forgot, so the live shell's
  // copy is cleared too — a standing that survived only there would come back
  // on the next report (the same rule the single Reset follows).
  const pruneUnused = async () => {
    let removed = 0;
    let forgotten = [];
    try {
      const r = await fetch("/api/browser/permissions/prune", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ days: 90 }),
      });
      if (!r.ok) throw new Error(String(r.status));
      const body = await r.json();
      removed = body.removed ?? 0;
      forgotten = Array.isArray(body.forgotten) ? body.forgotten : [];
    } catch {
      toast("Forgetting unused sites failed.");
      return;
    }
    for (const row of forgotten) {
      await invoke?.("btab_set_permission_policy", { kind: row.kind, state: "default", origin: row.origin }).catch(() => {});
    }
    if (removed === 0) toast.ok("Nothing to forget — every site has been visited in the last 90 days.");
    else toast.ok(`Forgot ${removed} unused ${removed === 1 ? "entry" : "entries"}.`);
    loadStands();
  };

  const forgetStanding = async (st) => {
    await fetch("/api/browser/permissions/" + st.id, { method: "DELETE" }).catch(() => {});
    if (invoke) {
      await invoke("btab_set_permission_policy", {
        kind: st.kind,
        state: "default",
        origin: st.origin === ALL_SITES ? null : st.origin,
      }).catch(() => {});
    }
    loadStands();
  };

  // ADR-0143: a principal is a managed agent or a terminal, and the row key
  // carries which — the same namespace rule the daemon keys grants with.
  const keyOf = (row) => (row.termId ? "term:" + row.termId : row.agentId);

  const draftOf = (row) => drafts[keyOf(row)] ?? { tier: row.tier, domainsText: domainsText(row) };
  const setDraft = (id, next) => setDrafts((d) => ({ ...d, [id]: next }));

  const save = async (row) => {
    const d = draftOf(row);
    const parsed = browserGrantSchema.safeParse({ tier: d.tier, domains: d.domainsText });
    if (!parsed.success) return;
    try {
      const r = await fetch("/api/browser/policy", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(row.termId
          ? { term: row.termId, tier: parsed.data.tier, domains: parsed.data.domains }
          : { agent: row.agentId, tier: parsed.data.tier, domains: parsed.data.domains }),
      });
      if (!r.ok) throw new Error((await r.json().catch(() => ({})))?.error || String(r.status));
      setFlash(keyOf(row));
      setTimeout(() => setFlash((cur) => (cur === keyOf(row) ? "" : cur)), 2000);
      setDrafts((dd) => {
        const next = { ...dd };
        delete next[keyOf(row)];
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

            <Item title="Annotation screenshots" desc="What the browser sends when you point at an element: the cropped image, a question each time you annotate, or the element alone.">
              <select className="set-select" value={prefs.annotationShots} onChange={(e) => setPref({ annotationShots: e.target.value })} aria-label="Annotation screenshots">
                <option value="always">Always include</option>
                <option value="ask">Ask each time</option>
                <option value="never">Never</option>
              </select>
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
            <Item title="Password manager" desc="Add, delete, and edit saved passwords">
              <button type="button" className="set-btn" onClick={() => setManageOpen("passwords")}>Manage</button>
            </Item>
            <Item title="Contact info" desc="Add, delete, and edit saved addresses, phone numbers, and email addresses">
              <button type="button" className="set-btn" onClick={() => setManageOpen("contact")}>Manage</button>
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Downloads</h2>
          <div className="set-panel">
            <Item title="Location" desc={downloadDir || "System Downloads folder"}>
              <button type="button" className="set-btn" onClick={() => setDirOpen(true)}>Change</button>
            </Item>
            <Item title="Ask where to save downloads" desc="Show a save prompt for downloads you start in the built-in browser">
              <SwitchCtl checked={prefs.askDownload} onChange={(v) => setPref({ askDownload: v })} label="Ask where to save downloads" />
            </Item>
            <Item title="Download history" desc="View and manage files downloaded from the built-in browser">
              <button type="button" className="set-btn" onClick={() => setDownloadsOpen(true)}>Manage</button>
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Browser permissions</h2>
          <div className="set-panel">
            <Item title="Site settings" desc="Camera, microphone, and the other permissions sites ask for">
              <button type="button" className="set-btn" onClick={() => setSiteOpen(true)}>Manage</button>
            </Item>
            <Item title="JavaScript" desc="Sites can use JavaScript">
              <SwitchCtl checked={prefs.scriptsEnabled} onChange={(v) => setPref({ scriptsEnabled: v })} label="Sites can use JavaScript" />
            </Item>
            <Item title="Agent history access" desc="Whether agents may read where you have been — off unless you allow it">
              <select
                className="set-select"
                value={prefs.historyAccess}
                onChange={(e) => setPref({ historyAccess: e.target.value })}
                aria-label="Agent history access"
              >
                <option value="never">Never</option>
                <option value="allow">Allow</option>
              </select>
            </Item>
          </div>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Agent permissions</h2>
          <p className="set-groupdesc">Which agents and terminals may use the built-in browser, and on which sites.</p>
          <div className="set-panel">
            {rows === null ? (
              <div className="set-item"><div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div></div>
            ) : rows.length === 0 ? (
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">No agents or terminals yet</span>
                  <span className="set-item-d">Grants are given per principal; anything without one reads the tab on screen.</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={onCreateAgent}>Create an agent</button>
                </div>
              </div>
            ) : (
              rows.map((row) => (
                <GrantRow
                  key={keyOf(row)}
                  row={row}
                  draft={draftOf(row)}
                  onDraft={(next) => setDraft(keyOf(row), next)}
                  onSave={() => save(row)}
                  flash={flash === keyOf(row)}
                />
              ))
            )}
          </div>
          <p className="set-groupdesc">
            Read — sees the tab you have on screen, nothing else. Act — also opens and drives pages in its own pane, inside the domains you list. Full — everything Act does, plus developer access.
          </p>
          <p className="set-groupdesc">
            Terminals you started in PiCode get a row too — a CLI agent there is a principal like any other. A <code>pi</code> started outside PiCode has no identity at all and always reads the tab on screen.
          </p>
        </section>

        <section className="set-group">
          <h2 className="set-grouph">Developer mode</h2>
          <div className="set-panel">
            <div className="set-item set-item-risk">
              <div className="set-item-body">
                <span className="set-risk"><IconWarn /> Elevated risk</span>
                <span className="set-item-t">Enable full CDP access</span>
                <span className="set-item-d">
                  Lets an agent at the Full tier name any Chrome DevTools Protocol method, not only the ones PiCode allows on its own. Off, everything keeps working — the agent just stays inside the curated list.
                </span>
              </div>
              <div className="set-item-ctl">
                <SwitchCtl checked={prefs.developerMode} onChange={(v) => setPref({ developerMode: v })} label="Enable full CDP access" />
              </div>
            </div>
            <div className="set-item">
              <div className="set-item-body">
                <span className="set-item-d">
                  Raw access reaches what the curated list deliberately keeps away: the browser's stored cookies, sessions and page storage. Turn it on while you need it, and check the calls below afterwards.
                </span>
              </div>
            </div>
          </div>
          <div className="set-panel">
            <div className="set-item set-item-col">
              <div className="set-item-body">
                <span className="set-item-t">Raw calls</span>
                {devAudit === null ? (
                  <span className="set-item-d">Reading…</span>
                ) : devAudit.length === 0 ? (
                  <span className="set-item-d">No raw calls yet. Every call an agent makes with the switch on is listed here, allowed or refused.</span>
                ) : (
                  <ul className="set-audit">
                    {devAudit.map((call) => (
                      <li key={call.id} className="set-audit-row">
                        <code className="set-audit-method">{call.method}</code>
                        <span className={"set-audit-outcome" + (call.outcome === "allowed" ? " is-ok" : "")}>{call.outcome}</span>
                        <span className="set-audit-actor">{call.agentId || call.termId || "unknown"}</span>
                        <span className="set-audit-when">{relTime(call.calledAt)}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </div>
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

      <Dialog.Root open={!!manageOpen} onOpenChange={(open) => { if (!open) { setManageOpen(""); setConfirmPurge(false); } }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-manage">
            <Dialog.Title className="dlg-title">{manageOpen === "passwords" ? "Password manager" : "Contact info"}</Dialog.Title>
            <p className="dlg-lede">
              {manageOpen === "passwords"
                ? "Saved passwords live in this machine's browser profile. PiCode can turn saving on or off and delete everything the profile stored — it cannot show or edit single entries."
                : "Saved names, addresses, phone numbers, and email addresses live in this machine's browser profile. PiCode can turn saving on or off and delete everything the profile stored — it cannot show or edit single entries."}
            </p>
            <div className="set-panel">
              {manageOpen === "passwords" ? (
                <>
                  <Item title="Offer to save passwords" desc="Save and fill sign-ins in the built-in browser">
                    <SwitchCtl checked={prefs.passwordAutosave} onChange={(v) => setPref({ passwordAutosave: v })} label="Password autosave" />
                  </Item>
                  <Item title="Saved passwords" desc="Passwords stored by the browser profile on this machine.">
                    <button
                      type="button"
                      className={"set-btn" + (confirmPurge ? " set-btn-danger" : "")}
                      onClick={() => purgeSaved("passwords")}
                      disabled={purgeBusy}
                    >
                      {purgeBusy ? "Deleting…" : confirmPurge ? "Really delete?" : "Delete data"}
                    </button>
                  </Item>
                  {shellInvoke(window) ? (
                    <Item
                      title="Windows Hello and passkeys"
                      desc="Your face, finger or PIN — and the passkeys bound to them — are kept and unlocked by Windows, not by PiCode. This opens Windows' own Sign-in options."
                    >
                      <button type="button" className="set-btn" onClick={openSignIn}>
                        Open Windows settings
                      </button>
                    </Item>
                  ) : null}
                </>
              ) : (
                <>
                  <Item title="Save and fill addresses" desc="Includes information like phone numbers, email addresses, and shipping addresses">
                    <SwitchCtl checked={prefs.generalAutofill} onChange={(v) => setPref({ generalAutofill: v })} label="General autofill" />
                  </Item>
                  <Item title="Saved form data" desc="Names, addresses, and other entries saved to fill forms on this machine.">
                    <button
                      type="button"
                      className={"set-btn" + (confirmPurge ? " set-btn-danger" : "")}
                      onClick={() => purgeSaved("autofill")}
                      disabled={purgeBusy}
                    >
                      {purgeBusy ? "Deleting…" : confirmPurge ? "Really delete?" : "Delete data"}
                    </button>
                  </Item>
                </>
              )}
            </div>
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => { setManageOpen(""); setConfirmPurge(false); }}>Close</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
      <Dialog.Root open={dirOpen} onOpenChange={(open) => { if (!open) setDirOpen(false); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-manage">
            <Dialog.Title className="dlg-title">Download folder</Dialog.Title>
            <p className="dlg-lede">Where the built-in browser writes downloads.</p>
            <div className="set-panel">
              <div className="set-item">
                <div className="set-item-body">
                  <span className="set-item-t">Folder</span>
                  <span className="set-item-d">{downloadDir || "System Downloads folder"}</span>
                </div>
                <div className="set-item-ctl">
                  <button type="button" className="set-btn" onClick={() => setPickerOpen(true)}>Browse…</button>
                </div>
              </div>
            </div>
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => saveDownloadDir("")}>Use the system folder</button>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setDirOpen(false)}>Close</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <FolderPicker
        open={pickerOpen}
        start={toPickerPath(downloadDir) || "/mnt/c"}
        onPick={pickDownloadDir}
        onClose={() => setPickerOpen(false)}
      />

      <Dialog.Root open={siteOpen} onOpenChange={setSiteOpen}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-manage">
            <Dialog.Title className="dlg-title">Site settings</Dialog.Title>
            <p className="dlg-lede">
              What sites may do in the built-in browser. Allow or block each kind for every site, choose Ask to be prompted when a site asks, or leave the platform's own default (deny) in place. Every decision the browser makes is listed below; a site you answered "Always allow" for stays remembered there.
            </p>
            <div className="set-panel">
              {PERMISSION_KINDS.map((k) => (
                <Item key={k.value} title={k.title} desc={k.d}>
                  <select
                    className="set-select"
                    value={policyOf(k.value)}
                    onChange={(e) => setPolicy(k.value, e.target.value)}
                    aria-label={k.title + " policy"}
                  >
                    <option value="">Platform default</option>
                    <option value="allow">Allow</option>
                    <option value="deny">Block</option>
                    <option value="ask">Ask</option>
                  </select>
                </Item>
              ))}
            </div>
            {/* The reference's "+ Add": a site exception authored by hand,
                instead of only reacting to a prompt. Same rows the Ask bar
                writes, with standing on. It sits above the log because it is
                the action; the log is the record. */}
            <div className="hist-head">
              <span className="hist-head-t">Add an exception</span>
            </div>
            <form
              className="site-add"
              noValidate
              onSubmit={(e) => {
                e.preventDefault();
                addSite();
              }}
            >
              <input
                className={"site-add-site" + (newSiteError ? " grant-domains-bad" : "")}
                placeholder="x.com, *.example.com, *"
                value={newSite.site}
                onChange={(e) => setNewSite({ ...newSite, site: e.target.value })}
                aria-label="Site or pattern for the exception"
                spellCheck={false}
              />
              <select
                className="set-select"
                value={newSite.kind}
                onChange={(e) => setNewSite({ ...newSite, kind: e.target.value })}
                aria-label="Permission kind"
              >
                {BROWSER_PERMISSION_KINDS.map((k) => (
                  <option key={k} value={k}>{k}</option>
                ))}
              </select>
              <select
                className="set-select"
                value={newSite.decision}
                onChange={(e) => setNewSite({ ...newSite, decision: e.target.value })}
                aria-label="Decision for the exception"
              >
                <option value="allow">Always allow</option>
                <option value="ask">Ask</option>
                <option value="deny">Block</option>
              </select>
              <button type="submit" className="set-btn" disabled={addingSite || !newSite.site.trim()}>
                {addingSite ? "Adding…" : "Add"}
              </button>
            </form>
            {newSiteError ? <p className="grant-error">{newSiteError}</p> : null}
            <div className="hist-head">
              <span className="hist-head-t">Recent decisions</span>
              {stands && stands.length > 0 ? (
                <button
                  type="button"
                  className="set-btn"
                  onClick={pruneUnused}
                  title="Forgets the sites you have not visited in the last 90 days"
                >
                  Forget unused
                </button>
              ) : null}
            </div>
            {stands === null ? (
              <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
            ) : stands.length === 0 ? (
              <p className="set-groupdesc">Nothing yet — add one above, or wait for a site to ask.</p>
            ) : (
              <div className="hist-days">
                <ul className="hist-rows">
                  {stands.map((st) => (
                    <li className="hist-row" key={st.id}>
                      <span className="hist-main">
                        <span className="hist-t" title={st.origin}>{st.origin === ALL_SITES ? "Every site" : st.origin}</span>
                        <span className="hist-host">{st.kind}{st.standing ? " · saved" : ""}</span>
                      </span>
                      <span className="hist-time" style={st.decision === "deny" ? { color: "var(--danger)" } : undefined}>{st.decision}</span>
                      <button type="button" className="set-btn" onClick={() => forgetStanding(st)} aria-label={"Forget the " + st.kind + " decision"}>Reset</button>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setSiteOpen(false)}>Close</button>
            </div>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>

      <Dialog.Root open={downloadsOpen} onOpenChange={(open) => { setDownloadsOpen(open); if (!open) setDlQuery(""); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dlg-overlay" />
          <Dialog.Content className="dlg dlg-hist">
            <Dialog.Title className="dlg-title">Download history</Dialog.Title>
            <input
              className="hist-search"
              type="search"
              placeholder="Search download history"
              value={dlQuery}
              onChange={(e) => setDlQuery(e.target.value)}
              aria-label="Search download history"
            />
            <div className="hist-head">
              <span className="hist-head-t">{dlQuery ? `Results for “${dlQuery}”` : "All-time history"}</span>
              <button
                type="button"
                className={"set-btn" + (confirmClearAll ? " set-btn-danger" : "")}
                onClick={clearDownloads}
                disabled={!downloads || downloads.length === 0}
              >
                {confirmClearAll ? "Really clear all?" : "Clear all"}
              </button>
            </div>
            {downloads === null ? (
              <div className="mcp-skel" aria-hidden="true"><span className="skel-line w-40" /><span className="skel-line w-70" /></div>
            ) : downloads.length === 0 ? (
              <div className="dl-empty">
                <svg width="22" height="22" viewBox="0 0 22 22" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path d="M11 3v10" /><path d="M6.5 9.5L11 14l4.5-4.5" /><path d="M3.5 18h15" />
                </svg>
                <span className="dl-empty-t">{dlQuery ? `No downloads match “${dlQuery}”.` : "No downloads yet"}</span>
                {dlQuery ? null : <span className="dl-empty-d">Files downloaded from the built-in browser will appear here.</span>}
              </div>
            ) : (
              <div className="hist-days">
                <ul className="hist-rows">
                  {downloads.map((d) => (
                    <li className="hist-row" key={d.id}>
                      <span className="hist-fav" aria-hidden="true">
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.3" strokeLinecap="round" strokeLinejoin="round">{WIPE_ICONS.downloads}</svg>
                      </span>
                      <span className="hist-main">
                        <span className="hist-t" title={d.path || d.name}>{d.name}</span>
                        <span className="hist-host">{d.host}</span>
                      </span>
                      <span className="hist-time" style={d.status === "interrupted" ? { color: "var(--danger)" } : undefined}>
                        {d.status === "interrupted" ? "interrupted" : d.status === "started" ? "downloading" : bytesLabel(d.total || d.received)}
                      </span>
                      <span className="hist-time">{timeLabel(visitDate({ visitedAt: d.startedAt }))}</span>
                      <DropdownMenu.Root>
                        <DropdownMenu.Trigger className="hist-more" aria-label={"More actions for " + d.name}>⋯</DropdownMenu.Trigger>
                        <DropdownMenu.Portal>
                          <DropdownMenu.Content className="um-popover" align="end" sideOffset={4} collisionPadding={8}>
                            <DropdownMenu.Item className="um-item" onSelect={() => openDownload(d, false)}>Open file</DropdownMenu.Item>
                            <DropdownMenu.Item className="um-item" onSelect={() => openDownload(d, true)}>Show in folder</DropdownMenu.Item>
                            <DropdownMenu.Item className="um-item" onSelect={() => copyLink(d.path || d.url)}>Copy path</DropdownMenu.Item>
                            <DropdownMenu.Item className="um-item" onSelect={() => removeDownload(d.id)}>Remove from history</DropdownMenu.Item>
                          </DropdownMenu.Content>
                        </DropdownMenu.Portal>
                      </DropdownMenu.Root>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            <div className="dlg-actions" data-align-row>
              <button type="button" className="btn btn-sm btn-ghost" onClick={() => setDownloadsOpen(false)}>Close</button>
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
  // What the field currently means, in words (browserDomains.js is the same
  // reading the matchers do — a hint, never a check). An entry that opens
  // nothing is named here instead of being saved in silence.
  const covers = actLike ? describeDomainField(draft.domainsText) : [];
  // A row that carries a domain list stacks (name and hints, then the
  // controls): side by side, the field was squeezed to a stub and the select
  // wrapped above it (seen in review, 2026-09-15).
  return (
    <div className={"set-item" + (actLike ? " set-item-stack" : "")}>
      <div className="set-item-body">
        <span className="set-item-t">
          {row.name}
          {row.saved ? <span className="devs-tag">custom</span> : <span className="devs-tag devs-tag-off">default</span>}
        </span>
        {!valid && actLike ? <span className="set-item-d" style={{ color: "var(--danger)" }}>{parsed.error.issues[0].message}</span> : null}
        {actLike && valid ? (
          covers.length === 0 ? (
            <span className="set-item-d">
              No sites yet — this agent can only read the tab you have on screen. Add a host, or * for any site.
            </span>
          ) : (
            <ul className="grant-covers">
              {covers.map((c) => (
                <li key={c.entry} className={"grant-cover" + (c.kind === "dead" ? " is-dead" : "")}>
                  <span className="grant-cover-label">{c.label}</span>
                  {c.covers ? <span className="grant-cover-d"> — {c.covers}</span> : null}
                  {c.kind === "dead" ? <span className="grant-cover-d"> — {c.miss}</span> : null}
                </li>
              ))}
            </ul>
          )
        ) : null}
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
            placeholder="example.com, *.example.com, *"
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
