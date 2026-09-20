import { test } from "node:test";
import assert from "node:assert/strict";
import { webappBadge, updateWebappMeta, webappTabId, submitWebappForm, webappOpenPlan, webappChromeless, webappIdFromTab, removedWebappTabs, webappError, DESKTOP_REQUIRED } from "./webapps.js";
;

test("install forwards allowDuplicate so a second account can coexist", async () => {
  const calls = [];
  const api = async (path, options) => {
    calls.push({ path, ...options });
    return { id: "x-1", name: "N", url: "https://x.com" };
  };
  await submitWebappForm(api, {
    mode: "install",
    preview: { url: "https://x.com" },
    url: "https://x.com",
    name: "N",
    allowDuplicate: true,
  });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].path, "/api/webapps");
  assert.deepEqual(JSON.parse(calls[0].body), { url: "https://x.com", name: "N", allowDuplicate: true });
});

test("rename submits the edited name instead of the original name", async () => {
  const app = { id: "mail-abc123", name: "Original name" };
  const calls = [];
  const api = async (path, options) => {
    calls.push({ path, ...options });
    return { ...app, ...JSON.parse(options.body) };
  };
  const result = await submitWebappForm(api, { mode: "rename", app, name: "  Edited name  " });
  assert.equal(calls.length, 1);
  assert.equal(calls[0].path, "/api/webapps/mail-abc123");
  assert.equal(calls[0].method, "PATCH");
  assert.deepEqual(JSON.parse(calls[0].body), { name: "Edited name" });
  assert.equal(result.saved.name, "Edited name");
  assert.equal(app.name, "Original name");
});

test("rename rejects an empty edited name before making a request", async () => {
  let calls = 0;
  await assert.rejects(submitWebappForm(async () => { calls++; }, {
    mode: "rename", app: { id: "mail-abc123", name: "Original name" }, name: "  ",
  }));
  assert.equal(calls, 0);
});

test("WebTab onMeta updates the installed tile badge; closing ignores late metadata", () => {
  const app = { id: "mail-abc123", name: "Mail", url: "http://localhost:3000", hasIcon: false };
  const tab = webappTabId(app.id);
  const tabs = ["w:1", tab];
  let meta = {};
  assert.equal(webappBadge(app, tabs, meta), 0);
  meta = updateWebappMeta(meta, tabs, tab, { url: app.url, title: "(12) Inbox" });
  assert.equal(webappBadge(app, tabs, meta), 12);
  meta = updateWebappMeta(meta, tabs, tab, { title: "Inbox" });
  assert.equal(webappBadge(app, tabs, meta), 0);
  meta = updateWebappMeta(meta, tabs, tab, { title: "(120) Inbox" });
  assert.equal(webappBadge(app, tabs, meta), 120);
  const closed = tabs.filter((id) => id !== tab);
  assert.equal(webappBadge(app, closed, meta), 0);
  assert.equal(updateWebappMeta(meta, closed, tab, { title: "(9) Late reply" }), meta);
});

test("open plan: focus an open tab, open a new one, refuse without the desktop", () => {
  const app = { id: "mail-abc123", name: "Mail", url: "http://localhost:3000", hasIcon: false };
  const tab = webappTabId(app.id);
  const open = webappOpenPlan(app, ["w:1"], true);
  assert.deepEqual(open, { action: "open", tab, id: tab.slice(2), url: app.url });
  assert.equal(webappOpenPlan(app, ["w:1", tab], true).action, "focus");
  assert.equal(webappOpenPlan(app, [], false).action, "desktop-required");
  assert.equal(webappOpenPlan({ id: "bad id!" }, [], true).action, "invalid");
  assert.equal(webappIdFromTab(tab), "mail-abc123");
  assert.equal(webappIdFromTab("w:app-../escape"), null);
});

test("launch url: a bare address defers to the manifest start_url; a typed deep link wins", () => {
  const bare = { id: "app-abc123", url: "https://app.example.com", startUrl: "https://app.example.com/app/?source=pwa" };
  const deep = { id: "app-abc123", url: "https://app.example.com/#/inbox", startUrl: "https://app.example.com/app/?source=pwa" };
  const plain = { id: "app-abc123", url: "https://app.example.com" };
  assert.equal(webappOpenPlan(bare, [], true).url, "https://app.example.com/app/?source=pwa");
  assert.equal(webappOpenPlan(deep, [], true).url, "https://app.example.com/#/inbox");
  assert.equal(webappOpenPlan(plain, [], true).url, "https://app.example.com");
});

test("removed tabs are exactly the installed-app tabs whose app is gone", () => {
  const live = [{ id: "mail-abc123" }];
  const tabs = ["w:1", webappTabId("mail-abc123"), webappTabId("gone-000000")];
  assert.deepEqual(removedWebappTabs(tabs, live), [webappTabId("gone-000000")]);
});

test("app mode: standalone, fullscreen and minimal-ui hide the browser toolbar", () => {
  assert.equal(webappChromeless("standalone"), true);
  assert.equal(webappChromeless("fullscreen"), true);
  assert.equal(webappChromeless("minimal-ui"), true);
  assert.equal(webappChromeless("browser"), false);
  assert.equal(webappChromeless(undefined), false);
  assert.equal(webappChromeless("browser-extensions"), false);
});

test("error mapping: duplicate, unreachable and validation", () => {
  assert.equal(webappError({ body: { reason: "duplicate" } }), "A web app for this address is already installed.");
  assert.match(webappError(new Error("site is not reachable: refused")), /did not answer/);
  assert.match(webappError({ issues: [{ message: "URL is required." }] }), /URL is required/);
  assert.equal(DESKTOP_REQUIRED, "Web apps require PiCode Desktop. Open PiCode Desktop to continue.");
});

test("shell errors arrive as strings and pass through untouched", () => {
  assert.match(
    webappError("could not clear the app's data — its files were still in use: locked"),
    /files were still in use/,
  );
  assert.equal(webappError("only installed web apps have their own data to clear"), "only installed web apps have their own data to clear");
});
