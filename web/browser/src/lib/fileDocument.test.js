import { test } from "node:test";
import assert from "node:assert/strict";
import { createDocumentGuard, createFileDocument } from "./fileDocument.js";
import { ownerFileURL, withFileRoot } from "./fileIO.js";

const deferred = () => {
  let resolve, reject;
  const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
};
const page = (text = "disk", mtime = 1) => ({ kind: "text", text, mtime });

for (const choice of ["cancel", "discard", "save"]) {
  test(`dirty navigation: ${choice}`, async () => {
    const writes = [];
    const doc = createFileDocument({ read: async () => page(), write: async (...args) => { writes.push(args); return { mtime: 2 }; } });
    await doc.refresh();
    doc.edit("draft");
    assert.equal(await createDocumentGuard(doc, async () => choice)(), choice !== "cancel");
    assert.deepEqual(writes, choice === "save" ? [["draft", 1]] : []);
    assert.equal(doc.getSnapshot().text, "draft");
    assert.equal(doc.getSnapshot().dirty, choice !== "save");
  });
}

test("clean navigation and undo to the saved text need no prompt", async () => {
  const doc = createFileDocument({ read: async () => page() });
  await doc.refresh();
  doc.edit("draft");
  doc.edit("disk");
  assert.equal(await createDocumentGuard(doc, () => assert.fail("unneeded prompt"))(), true);
});

for (const error of ["connection lost", "This file changed on disk.", "This folder changed. Refresh the file tree."]) {
  test(`save failure preserves the draft and cancels navigation: ${error}`, async () => {
    const doc = createFileDocument({ read: async () => page(), write: async () => { throw new Error(error); } });
    await doc.refresh();
    doc.edit("precious draft");
    const before = doc.getSnapshot().revision;
    assert.equal(await createDocumentGuard(doc, async () => "save")(), false);
    assert.equal(doc.getSnapshot().kind, "text");
    assert.equal(doc.getSnapshot().text, "precious draft");
    assert.equal(doc.getSnapshot().dirty, true);
    assert.equal(doc.getSnapshot().revision, before);
    assert.equal(doc.getSnapshot().error, error);
  });
}

test("edits made during a save stay dirty and cannot release navigation", async () => {
  const write = deferred();
  const doc = createFileDocument({ read: async () => page(), write: () => write.promise });
  await doc.refresh();
  doc.edit("first");
  const saved = doc.save();
  assert.equal(doc.save(), saved, "a second save shares the in-flight write");
  doc.edit("second");
  write.resolve({ mtime: 2 });
  assert.equal(await saved, false);
  assert.equal(doc.getSnapshot().dirty, true);
  assert.equal(doc.getSnapshot().text, "second");
});

test("navigation waits for an existing save and overlapping clicks share a prompt", async () => {
  const write = deferred();
  const choice = deferred();
  let asks = 0;
  const doc = createFileDocument({ read: async () => page(), write: () => write.promise });
  await doc.refresh();
  doc.edit("draft");
  const guard = createDocumentGuard(doc, () => { asks++; return choice.promise; });
  const save = doc.save();
  const left = guard();
  assert.equal(guard(), left);
  assert.equal(asks, 0);
  write.reject(new Error("offline"));
  await save;
  // The failed write is still dirty; the one pending decision belongs to it.
  await Promise.resolve();
  choice.resolve("cancel");
  assert.equal(await left, false);
  assert.equal(asks, 1);
});

test("a successful in-flight save releases navigation without another prompt", async () => {
  const write = deferred();
  const doc = createFileDocument({ read: async () => page(), write: () => write.promise });
  await doc.refresh();
  doc.edit("draft");
  doc.save();
  const left = createDocumentGuard(doc, () => assert.fail("already saved"))();
  write.resolve({ mtime: 2 });
  assert.equal(await left, true);
});

test("refresh keeps clean content until it arrives and preserves editor identity for unchanged bytes", async () => {
  let read = async () => page();
  const doc = createFileDocument({ read: () => read() });
  await doc.refresh();
  const revision = doc.getSnapshot().revision;
  const delayed = deferred();
  read = () => delayed.promise;
  const refresh = doc.refresh();
  assert.equal(doc.getSnapshot().text, "disk");
  assert.equal(doc.getSnapshot().refreshing, true);
  delayed.resolve(page("disk", 2));
  await refresh;
  assert.equal(doc.getSnapshot().revision, revision);
});

test("refresh never overwrites dirty edits, including edits made after the read started", async () => {
  let reads = 0;
  const delayed = deferred();
  const doc = createFileDocument({ read: () => ++reads === 1 ? Promise.resolve(page()) : delayed.promise });
  await doc.refresh();
  const refresh = doc.refresh();
  doc.edit("draft");
  delayed.resolve(page("changed outside"));
  assert.equal(await refresh, false);
  assert.equal(await doc.refresh(), false);
  assert.equal(reads, 2);
  assert.equal(doc.getSnapshot().text, "draft");
});

test("a removed file or failed reload preserves its current buffer", async () => {
  let fail = false;
  const doc = createFileDocument({ read: async () => { if (fail) throw new Error("gone"); return page(); } });
  await doc.refresh();
  doc.edit("draft");
  fail = true;
  assert.equal(await doc.refresh({ discard: true }), false);
  assert.equal(doc.getSnapshot().text, "draft");
  assert.equal(doc.getSnapshot().dirty, true);
});

test("out-of-order reads and disposal cannot publish stale content; unused blobs are released", async () => {
  const first = deferred(), second = deferred();
  const released = [];
  let reads = 0;
  const doc = createFileDocument({ read: () => ++reads === 1 ? first.promise : second.promise, release: (p) => { if (p.src) released.push(p.src); } });
  const a = doc.refresh(), b = doc.refresh();
  second.resolve(page("latest"));
  await b;
  first.resolve({ kind: "bin", src: "stale" });
  await a;
  assert.equal(doc.getSnapshot().text, "latest");
  assert.deepEqual(released, ["stale"]);
  const final = deferred();
  const old = createFileDocument({ read: () => final.promise, release: (p) => released.push(p.src) });
  const pending = old.refresh();
  old.dispose();
  final.resolve({ kind: "bin", src: "closed" });
  await pending;
  assert.equal(old.getSnapshot().kind, "load");
  assert.equal(released.at(-1), "closed");
});

for (const [kind, family] of [["agent", "agents"], ["term", "terminals"], ["workspace", "workspaces"]]) {
  test(`${kind} file requests encode paths and carry the pinned root`, () => {
    const owner = { kind, id: "id /?" };
    const url = new URL(ownerFileURL(owner, "text", "space/#?.md", "/repo/worktree"), "http://local");
    assert.equal(url.pathname, `/api/${family}/id%20%2F%3F/text`);
    assert.equal(url.searchParams.get("path"), "space/#?.md");
    assert.equal(url.searchParams.get("root"), "/repo/worktree");
    const browse = new URL(ownerFileURL(owner, "browse", "a b", "/repo"), "http://local");
    assert.equal(browse.searchParams.get("dir"), "a b");
    assert.equal(browse.searchParams.has("path"), false);
    const scoped = new URL(ownerFileURL(owner, "text", "a.txt", "/repo/side", "side"), "http://local");
    assert.equal(scoped.searchParams.get("worktree"), "side");
    assert.equal(scoped.searchParams.get("root"), "/repo/side");
    const unscoped = new URL(ownerFileURL(owner, "text", "a.txt", "/repo"), "http://local");
    assert.equal(unscoped.searchParams.has("worktree"), false);
    assert.equal(withFileRoot("", "/repo"), "");
    assert.equal(withFileRoot("/blob?path=a", "/repo"), "/blob?path=a&root=%2Frepo");
  });
}
