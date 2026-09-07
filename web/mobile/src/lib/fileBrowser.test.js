import { test } from "node:test";
import assert from "node:assert/strict";
import { folderPage, parentFolder, searchFileFolders } from "./fileBrowser.js";

const pages = {
  "": { dirs: [{ name: "src", path: "src" }], files: [{ name: "README.md", path: "README.md" }] },
  src: { dirs: [{ name: "deep", path: "src/deep" }], files: [{ name: "app.js", path: "src/app.js" }] },
  "src/deep": { dirs: [], files: [{ name: "app.test.js", path: "src/deep/app.test.js" }] },
};

test("recursive search matches every query word and retains full paths", async () => {
  const reads = [];
  const found = await searchFileFolders({ read: async dir => { reads.push(dir); return pages[dir]; }, query: "SRC APP" });
  assert.deepEqual(found.results.map(row => row.path), ["src/app.js", "src/deep/app.test.js"]);
  assert.deepEqual(reads, ["", "src", "src/deep"]);
  assert.equal(found.limited, false);
});

test("empty search makes no requests; nested search starts at its folder", async () => {
  await searchFileFolders({ read: () => assert.fail("empty search"), query: " " });
  const reads = [];
  await searchFileFolders({ read: async dir => { reads.push(dir); return pages[dir]; }, dir: "src/deep", query: "app" });
  assert.deepEqual(reads, ["src/deep"]);
});

for (const bounds of [{ maxFolders: 1 }, { maxEntries: 1 }]) {
  test(`search reports its bound: ${JSON.stringify(bounds)}`, async () => {
    const found = await searchFileFolders({ read: async dir => pages[dir], query: "app", ...bounds });
    assert.equal(found.limited, true);
  });
}

test("repeated directory rows never loop and file rows are never traversed", async () => {
  const reads = [];
  const found = await searchFileFolders({ query: "link", read: async dir => {
    reads.push(dir);
    return { dirs: [{ name: "again", path: "" }], files: [{ name: "link", path: "symlink" }] };
  } });
  assert.deepEqual(reads, [""]);
  assert.equal(found.results[0].path, "symlink");
});

test("abort and moved-root errors stop search; inaccessible descendants are explicit", async () => {
  const stopped = new AbortController(); stopped.abort();
  await assert.rejects(searchFileFolders({ read: () => assert.fail("aborted"), query: "app", signal: stopped.signal }), /abort/i);
  await assert.rejects(searchFileFolders({ read: async dir => { if (dir) throw new Error("This folder changed."); return pages[dir]; }, query: "app" }), /folder changed/);
  const partial = await searchFileFolders({ read: async dir => { if (dir) throw new Error("permission denied"); return pages[dir]; }, query: "readme" });
  assert.equal(partial.skipped, 1);
  assert.equal(partial.results[0].path, "README.md");
});

test("folder responses require a working root and valid rows", () => {
  for (const value of [null, { cwdOk: false }, { root: "/root" }, { root: "/root", dirs: [], files: null }]) assert.throws(() => folderPage(value));
  assert.equal(folderPage({ root: "/root", dirs: [], files: [] }).root, "/root");
  assert.equal(parentFolder("a/b/c.js"), "a/b");
  assert.equal(parentFolder("readme.md"), "");
});
