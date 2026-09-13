import assert from "node:assert/strict";
import { test } from "node:test";
import { openRepoKeys, openTreeKeys, pickAction, pickTarget, ownerIdOf, ownerRoute, graphUrl, headUrl } from "./workspacePicker.js";

test("openRepoKeys drops provisional tabs and everything that is not a graph", () => {
  assert.deepEqual(
    openRepoKeys(["g:/home/goat/picode/.git", "g:@t:desktop-51c42d", "t:desktop-51c42d", "d:/home/goat/picode", ""]),
    ["/home/goat/picode/.git"],
  );
  assert.deepEqual(openRepoKeys(null), []);
});

test("openTreeKeys reads folder tabs, and drops the provisional ones", () => {
  assert.deepEqual(
    openTreeKeys(["d:/home/goat/picode", "d:@w:desktop-3da77a", "g:/home/goat/picode/.git", "t:desktop-51c42d", ""]),
    ["/home/goat/picode"],
  );
  assert.deepEqual(openTreeKeys(null), []);
});

test("pickAction: same folder keeps the tab, a new one renames it, an open one adopts it", () => {
  const key = "/home/goat/picode"; // a tree root; the graph passes a repository key
  // The same folder read through another owner: the tab stays, the owner moves.
  assert.equal(pickAction(key, key, [key]), "same");
  // Another folder with no tab: this tab becomes it (one tab per folder).
  assert.equal(pickAction("/home/goat/orikami", key, [key]), "rename");
  // Another folder already on the strip: select it, hand it the pick.
  assert.equal(pickAction("/home/goat/orikami", key, [key, "/home/goat/orikami"]), "adopt");
  // Nothing resolved is nothing to do — never a retarget.
  assert.equal(pickAction("", key, [key]), "none");
});

test("pickTarget: a head that throws and a head that names no repository both move nothing", async () => {
  const key = "/home/goat/picode/.git";
  const boom = new Error("no such workspace");
  assert.deepEqual(
    await pickTarget({ tabKey: key, openKeys: [key], readTarget: () => Promise.reject(boom) }),
    { key: "", action: "none", error: boom },
  );
  // A workspace whose folder stopped being a repository answers 200 with no key.
  assert.deepEqual(
    await pickTarget({ tabKey: key, openKeys: [key], readTarget: async () => ({}) }),
    { key: "", action: "none", error: null },
  );
  assert.deepEqual(
    await pickTarget({ tabKey: key, openKeys: [key], readTarget: async () => null }),
    { key: "", action: "none", error: null },
  );
});

test("pickTarget: the answer is what decides same, rename or adopt", async () => {
  const key = "/home/goat/picode/.git";
  const other = "/home/goat/orikami/.git";
  const answers = (k) => async () => ({ key: k, token: "t" });
  const same = await pickTarget({ tabKey: key, openKeys: [key], readTarget: answers(key) });
  assert.equal(same.action, "same");
  assert.equal(same.key, key);
  const rename = await pickTarget({ tabKey: key, openKeys: [key], readTarget: answers(other) });
  assert.equal(rename.action, "rename");
  assert.equal(rename.key, other);
  const adopt = await pickTarget({ tabKey: key, openKeys: [key, other], readTarget: answers(other) });
  assert.equal(adopt.action, "adopt");
});

test("ownerIdOf separates two owners that share an id, and names none for an absent one", () => {
  const ws = { kind: "workspace", id: "x" };
  assert.equal(ownerIdOf(ws), "workspace:x");
  assert.notEqual(ownerIdOf(ws), ownerIdOf({ kind: "agent", id: "x" }));
  assert.notEqual(ownerIdOf(ws), ownerIdOf({ kind: "term", id: "x" }));
  // The same owner across renders is the same string: the reset must not fire
  // on a re-render, or the open commit would close under the reader.
  assert.equal(ownerIdOf(ws), ownerIdOf({ kind: "workspace", id: "x", name: "renamed" }));
  assert.equal(ownerIdOf(null), "");
  assert.equal(ownerIdOf({}), "");
  assert.equal(ownerIdOf({ kind: "agent" }), "");
});

test("ownerRoute is the URL an owner is asked at — the state key never rides it", () => {
  // The regression: composing the two identities sent
  // `/api/workspaces/workspace%3A<id>/git` and the graph answered 404.
  const ws = { kind: "workspace", id: "w1" };
  const r = ownerRoute(ws);
  assert.equal(r.base + r.id + "/git", "/api/workspaces/w1/git");
  assert.equal((r.base + r.id).includes(":"), false);
  assert.equal(ownerRoute({ kind: "term", id: "t1" }).base + "t1" + "/git", "/api/terminals/t1/git");
  assert.equal(ownerRoute({ kind: "agent", id: "a1" }).base + "a1" + "/git", "/api/agents/a1/git");
  // An absent owner keeps the agent default the callers had (never "/api/undefined/").
  assert.equal(ownerRoute(null).base, "/api/agents/");
  assert.equal(ownerRoute(null).id, "");
  assert.equal(ownerIdOf(ws), "workspace:w1");
});

test("graphUrl and headUrl are the two graph requests — route + plain id, never the key", () => {
  const key = "workspace:w1";
  const { base, id } = ownerRoute({ kind: "workspace", id: "w1" });
  assert.equal(graphUrl(base, id, "limit=250"), "/api/workspaces/w1/git?limit=250");
  assert.equal(headUrl(base, id), "/api/workspaces/w1/git/head");
  // The requests this surface makes carry no kind prefix at all.
  assert.equal(graphUrl(base, id, "limit=1").includes("%3A"), false);
  assert.equal(headUrl(base, id).includes("%3A"), false);
  // And passing the state key by mistake is visible in the string the server
  // would see: a workspace literally named "workspace:w1" (the 404 that cost
  // the graph a load and the pending watch its token, twice on 2026-09-13).
  assert.equal(headUrl(base, key), "/api/workspaces/workspace%3Aw1/git/head");
  assert.equal(headUrl("/api/terminals/", "t 1"), "/api/terminals/t%201/git/head");
});
