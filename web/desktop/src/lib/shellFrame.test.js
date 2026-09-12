import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { isShell, claimFrame, releaseFrame } from "./shellFrame.js";

describe("shellFrame", () => {
  beforeEach(() => {
    const root = { dataset: {} };
    globalThis.window = {};
    globalThis.document = { documentElement: root };
  });

  it("is not a shell in a browser", () => {
    assert.equal(isShell(), false);
    claimFrame();
    assert.equal(document.documentElement.dataset.picodeFrame, undefined);
  });

  it("claims and releases the frame in shell mode", () => {
    globalThis.window = { __PICODE_SHELL__: true };
    assert.equal(isShell(), true);
    claimFrame();
    assert.equal(document.documentElement.dataset.picodeFrame, "1");
    releaseFrame();
    assert.equal("picodeFrame" in document.documentElement.dataset, false);
  });

  it("release is safe without a document", () => {
    globalThis.document = undefined;
    assert.doesNotThrow(() => releaseFrame());
  });
});
