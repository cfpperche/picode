import assert from "node:assert/strict";
import { test } from "node:test";
import { assetKind, assetSides, gitRevBlobUrl, gitWorkBlobUrl } from "./gitAsset.js";

test("assetKind covers the previewable families and refuses the rest", () => {
  assert.equal(assetKind("docs-videos/assets/stills/v2-3-inbox.png"), "image");
  assert.equal(assetKind("a.JPG"), "image");
  assert.equal(assetKind("render/take-it-anywhere.mp4"), "video");
  assert.equal(assetKind("note.mp3"), "audio");
  assert.equal(assetKind("doc.pdf"), "pdf");
  assert.equal(assetKind("model.glb"), "model3d");
  // SVG diffs as text — the text diff is the better answer there.
  assert.equal(assetKind("icon.svg"), "");
  assert.equal(assetKind("main.go"), "");
  assert.equal(assetKind("archive.zip"), "");
});

test("assetSides follows the change kind", () => {
  assert.deepEqual(assetSides("modified"), ["before", "after"]);
  assert.deepEqual(assetSides("renamed"), ["before", "after"]);
  assert.deepEqual(assetSides("added"), ["after"]);
  assert.deepEqual(assetSides("untracked"), ["after"]);
  assert.deepEqual(assetSides("deleted"), ["before"]);
  assert.deepEqual(assetSides(undefined), ["before", "after"]);
});

test("urls are owner-scoped and escape their pieces", () => {
  assert.equal(
    gitRevBlobUrl("/api/agents/", "a1", "abc123", "stills/a b.png"),
    "/api/agents/a1/git/blob?hash=abc123&path=stills%2Fa%20b.png",
  );
  assert.equal(
    gitRevBlobUrl("/api/terminals/", "t1", "HEAD", "v.png"),
    "/api/terminals/t1/git/blob?hash=HEAD&path=v.png",
  );
  assert.equal(
    gitWorkBlobUrl("/api/workspaces/", "w1", "v.png"),
    "/api/workspaces/w1/blob?path=v.png",
  );
});
