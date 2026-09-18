import { webappUrlSchema, webappInstallSchema, webappNameSchema } from "@picode/shared/contracts/schemas.js";

export const DESKTOP_REQUIRED = "Web apps require PiCode Desktop. Open PiCode Desktop to continue.";

export function webappError(error) {
  if (error?.body?.reason === "duplicate") return "A web app for this address is already installed.";
  if (/site is not reachable|answered HTTP/.test(error?.message || "")) return "That address did not answer. Check it and try again — nothing was installed.";
  return error?.issues?.[0]?.message || error?.message || "Could not save the web app. Try again.";
}

export async function submitWebappForm(api, { mode, app, preview, url, name }) {
  const schema = mode === "rename" ? webappNameSchema : preview ? webappInstallSchema : webappUrlSchema;
  const payload = schema.parse({ url: preview?.url || url, name });
  const path = mode === "rename" ? "/api/webapps/" + encodeURIComponent(app.id) : preview ? "/api/webapps" : "/api/webapps/resolve";
  const result = await api(path, {
    method: mode === "rename" ? "PATCH" : "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  return mode === "rename" || preview ? { saved: result } : { preview: result };
}

export function webappTabId(id) {
  return typeof id === "string" && /^[a-z0-9][a-z0-9-]{0,79}$/.test(id) ? "w:app-" + id : null;
}

export function webappIdFromTab(tab) {
  if (typeof tab !== "string" || !tab.startsWith("w:app-")) return null;
  const id = tab.slice(6);
  return webappTabId(id) === tab ? id : null;
}

export function titleBadge(title) {
  if (typeof title !== "string") return 0;
  const match = /^\(([0-9]{1,6})\)(?:\s|$)/.exec(title.slice(0, 10));
  return match ? Math.min(999999, Number(match[1])) : 0;
}

export function removedWebappTabs(tabs, apps) {
  const live = new Set(apps.map((a) => a.id));
  return tabs.filter((tab) => {
    const id = webappIdFromTab(tab);
    return id !== null && !live.has(id);
  });
}

export function webappOpenPlan(app, tabs, desktop) {
  const tab = webappTabId(app?.id);
  if (!tab) return { action: "invalid" };
  if (!desktop) return { action: "desktop-required" };
  return { action: tabs.includes(tab) ? "focus" : "open", tab, id: tab.slice(2), url: app.url };
}

export function updateWebappMeta(meta, tabs, tab, update) {
  if (!tabs.includes(tab)) return meta;
  const id = tab.slice(2);
  return { ...meta, [id]: { ...meta[id], ...update } };
}

export function webappBadge(app, tabs, meta) {
  const tab = webappTabId(app.id);
  return tabs.includes(tab) ? titleBadge(meta[tab.slice(2)]?.title) : 0;
}

export function watchWebapps({ api, subscribe, onList, onError, onRemoved }) {
  let live = true;
  let version = 0;
  async function refresh() {
    if (!live) return;
    const request = ++version;
    try {
      const apps = await api("/api/webapps");
      if (!Array.isArray(apps)) throw new Error("Could not load web apps.");
      if (live && request === version) { onList(apps); onError(""); }
    } catch (e) {
      if (live && request === version) onError(e.message || "Could not load web apps.");
    }
  }
  const off = subscribe((ev) => {
    if (ev.type === "webapp.removed" && webappTabId(ev.data?.id)) onRemoved(ev.data.id);
    if (ev.type === "feed.open" || ev.type === "feed.reset" || ev.type?.startsWith("webapp.")) refresh();
  });
  refresh();
  return { refresh, stop() { live = false; version++; off(); } };
}
