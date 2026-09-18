import assert from "node:assert/strict";
import { test } from "node:test";
import {
  API, OPAQUE, PAGE,
  ageLabel, canHide, canStop, defaultVisible, hiddenRows, isOpenable,
  otherRows, sections, serverMeta, serverName, serverNote, stopPrompt,
} from "./devServers.js";

// The two rows the panel's first version got wrong, verbatim from the owner's
// machine (2026-09-17): an agent CLI's HTTPS control channel on 45683 and its
// plain-HTTP API on 46579 — both attributed to a terminal, neither a page.
const agyHTTPS = {
  port: 45683, url: "https://localhost:45683/", scheme: "https", kind: API, state: "live",
  tool: "agy", ownerKind: "term", ownerId: "sidebar-6fabbd", ownerName: "sidebar", workspace: "PiCode",
  pid: 3257366, startKey: "987654", startedAt: "2026-09-17T19:40:00Z",
};
const agyAPI = { ...agyHTTPS, port: 46579, url: "http://localhost:46579/", scheme: "http" };
const vite = { port: 5173, url: "http://localhost:5173/", scheme: "http", kind: PAGE, state: "live", title: "Vite + React", ownerKind: "term", ownerName: "dashboard", workspace: "PiCode", pid: 4242, startKey: "900" };
const outside = { port: 3000, url: "http://localhost:3000/", scheme: "http", kind: PAGE, state: "live", title: "Acme" };
const strangerAPI = { port: 8080, url: "http://localhost:8080/", scheme: "http", kind: API, state: "live" };
const booting = { port: 8000, url: "http://localhost:8000/", scheme: "http", kind: OPAQUE, state: "starting", tool: "uvicorn", ownerKind: "term", ownerName: "web", pid: 4243, startKey: "901", startedAt: "2026-09-17T21:00:00Z" };

test("only a page can be opened", () => {
  assert.equal(isOpenable(vite), true);
  assert.equal(isOpenable(agyHTTPS), false);
  assert.equal(isOpenable(agyAPI), false);
  assert.equal(isOpenable(strangerAPI), false);
});

test("stop and hide need the identity of the process holding the socket", () => {
  assert.equal(canStop(vite), true);
  assert.equal(canHide(vite), true);
  assert.equal(canStop(agyHTTPS), true);
  // A port PiCode cannot attribute is neither stoppable nor hideable: it has
  // no identity to re-derive at action time.
  assert.equal(canStop(strangerAPI), false);
  assert.equal(canHide(strangerAPI), false);
  assert.equal(canStop({ ...vite, pid: 0 }), false);
  assert.equal(canStop({ ...vite, startKey: "" }), false);
});

test("the two agy rows are named as what they are", () => {
  assert.equal(serverName(agyHTTPS), "agy");
  assert.equal(serverNote(agyHTTPS), "HTTPS · not a page");
  assert.equal(serverNote(agyAPI), "not a page");
  assert.equal(serverNote(vite), "");
  assert.equal(serverNote({ ...vite, scheme: "https" }), "HTTPS");
  assert.equal(serverNote(booting), "starting");
  assert.equal(serverNote({ ...booting, state: "silent" }), "not answering");
});

test("a port that is not a page has no name, and never borrows one", () => {
  // "unnamed page" beside a "not a page" badge is the panel arguing with
  // itself — the port number is the identity of an API.
  assert.equal(serverName(strangerAPI), "");
  assert.equal(serverName({ ...strangerAPI, tool: "uvicorn" }), "uvicorn");
  assert.equal(serverName({ ...vite, title: "" }), "unnamed page");
});

test("the meta line names the owner, the workspace and the age", () => {
  const now = Date.parse("2026-09-17T21:44:00Z");
  assert.equal(serverMeta(agyHTTPS, now), 'terminal "sidebar" · PiCode · 2 h');
  assert.equal(serverMeta(vite, now), 'terminal "dashboard" · PiCode');
  assert.equal(serverMeta(outside, now), "started outside PiCode");
  assert.equal(serverMeta({ ...agyHTTPS, ownerKind: "agent", startedAt: "" }, now), 'agent "sidebar" · PiCode');
});

test("ages are coarse and never invented", () => {
  const now = Date.parse("2026-09-17T22:00:00Z");
  assert.equal(ageLabel("2026-09-17T21:59:30Z", now), "just now");
  assert.equal(ageLabel("2026-09-17T21:52:00Z", now), "8 min");
  assert.equal(ageLabel("2026-09-17T19:00:00Z", now), "3 h");
  assert.equal(ageLabel("2026-09-14T22:00:00Z", now), "3 d");
  assert.equal(ageLabel("", now), "");
  assert.equal(ageLabel("not a date", now), "");
});

test("pages and PiCode's own listeners are visible; a stranger's API waits", () => {
  const rows = [vite, agyHTTPS, agyAPI, outside, strangerAPI, booting, { ...strangerAPI, port: 9000, hidden: true, hideId: 7 }];
  // The agy rows stay: PiCode can see the process, so it can stop or hide it.
  // What they no longer do is claim to be a page.
  assert.deepEqual(rows.filter(defaultVisible).map((s) => s.port), [5173, 45683, 46579, 3000, 8000]);
  assert.deepEqual(otherRows(rows).map((s) => s.port), [8080]);
  assert.deepEqual(hiddenRows(rows).map((s) => s.port), [9000]);
});

test("mine first, pages first inside each section, then by port", () => {
  const rows = [vite, agyHTTPS, agyAPI, outside, strangerAPI, booting];
  const { mine, other } = sections(rows);
  assert.deepEqual(mine.map((s) => s.port), [5173, 45683, 46579, 8000]);
  assert.deepEqual(other.map((s) => s.port), [3000]);
});

test("a hide is a hide, not a filter over kinds", () => {
  const all = [vite, { ...vite, hidden: true }];
  assert.deepEqual(sections(all).mine.map((s) => !!s.hidden), [false]);
  assert.equal(hiddenRows(all).length, 1);
  assert.equal(otherRows([{ ...strangerAPI, hidden: true }]).length, 0);
});

test("the stop confirmation names the port, the owner and the cost", () => {
  const prompt = stopPrompt(agyHTTPS);
  assert.equal(prompt.title, "Stop agy on port 45683?");
  assert.ok(prompt.message.includes('terminal "sidebar"'));
  assert.ok(prompt.message.includes("Anything it was running stops too."));
});
