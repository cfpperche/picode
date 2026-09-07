import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { createAgentDrafts, submissionKind, isSubmitKey } from "./agentDrafts.js";

const image = { id: "image-1", data: "bytes", url: "data:image/png;base64,bytes", mime: "image/png" };
const accepted = async () => ({ accepted: true });

// Decision table: conditions -> retained/cleared content and accepted delivery.
describe("mobile agent drafts", () => {
  it("retains text, kind and images independently across route subscriptions", () => {
    const drafts = createAgentDrafts();
    const unsubscribe = drafts.subscribe(() => {});
    drafts.update("a", { text: "A draft", kind: "steer", images: [image] });
    unsubscribe();
    drafts.update("b", { text: "B draft", kind: "follow_up" });
    assert.equal(drafts.read("a").text, "A draft");
    assert.equal(drafts.read("a").kind, "steer");
    assert.deepEqual(drafts.read("a").images, [image]);
    assert.equal(drafts.read("b").text, "B draft");
    assert.deepEqual(drafts.read("b").images, []);
  });
  for (const [name, text, images] of [["text", "hello", []], ["image only", "", [image]], ["text and image", "hello", [image]]]) {
    it(`accepted ${name} clears only that agent's content`, async () => {
      const drafts = createAgentDrafts();
      drafts.update("a", { text, images, kind: "steer" });
      drafts.update("b", { text: "other" });
      assert.equal(await drafts.submit("a", accepted), true);
      assert.equal(drafts.read("a").text, "");
      assert.deepEqual(drafts.read("a").images, []);
      assert.equal(drafts.read("a").kind, "steer");
      assert.equal(drafts.read("b").text, "other");
    });
  }
  for (const [name, send] of [
    ["request error", async () => { throw new Error("Offline"); }],
    ["refused send", async () => ({ accepted: false, error: "Use the terminal" })],
    ["missing acknowledgement", async () => undefined],
  ]) {
    it(`${name} keeps recoverable content and retry clears it only after acceptance`, async () => {
      const drafts = createAgentDrafts();
      drafts.update("a", { text: "keep", images: [image] });
      assert.equal(await drafts.submit("a", send), false);
      assert.equal(drafts.read("a").text, "keep");
      assert.deepEqual(drafts.read("a").images, [image]);
      assert.ok(drafts.read("a").error);
      assert.equal(drafts.read("a").sending, false);
      assert.equal(await drafts.submit("a", accepted), true);
      assert.equal(drafts.read("a").error, "");
    });
  }
  it("empty or missing agent never sends", async () => {
    const drafts = createAgentDrafts();
    let calls = 0;
    const send = () => { calls++; return { accepted: true }; };
    drafts.update("a", { text: "  " });
    assert.equal(await drafts.submit("a", send), false);
    assert.equal(await drafts.submit("", send, "text"), false);
    assert.equal(calls, 0);
  });
  it("pending send suppresses duplicate requests and protects subsequent edits", async () => {
    const drafts = createAgentDrafts();
    drafts.update("a", { text: "first", images: [image] });
    let finish;
    const pending = drafts.submit("a", () => new Promise(resolve => { finish = resolve; }));
    assert.equal(drafts.read("a").sending, true);
    assert.equal(await drafts.submit("a", accepted), false);
    drafts.update("a", { text: "next", images: [] });
    finish({ accepted: true });
    assert.equal(await pending, true);
    assert.equal(drafts.read("a").text, "next");
    assert.deepEqual(drafts.read("a").images, []);
  });
  it("voice text is retained when its send fails", async () => {
    const drafts = createAgentDrafts();
    await drafts.submit("a", async () => { throw new Error("Offline"); }, "dictated message");
    assert.equal(drafts.read("a").text, "dictated message");
  });
});

describe("mobile submission controls", () => {
  for (const [streaming, waiting] of [[false, false], [true, false], [false, true], [true, true]]) {
    for (const kind of ["prompt", "steer", "follow_up"]) {
      it(`${kind}, streaming=${streaming}, waiting=${waiting}`, () => {
        assert.equal(submissionKind(kind, { streaming, waiting }), kind === "prompt" && (streaming || waiting) ? "follow_up" : kind);
      });
    }
  }
  for (const [name, event, submit] of [
    ["mobile Enter inserts a newline", { key: "Enter" }, false],
    ["Shift Enter inserts a newline", { key: "Enter", shiftKey: true }, false],
    ["Ctrl Enter sends", { key: "Enter", ctrlKey: true }, true],
    ["Command Enter sends", { key: "Enter", metaKey: true }, true],
    ["composition Enter does not send", { key: "Enter", ctrlKey: true, isComposing: true }, false],
    ["legacy IME Enter does not send", { key: "Enter", ctrlKey: true, keyCode: 229 }, false],
  ]) it(name, () => assert.equal(isSubmitKey(event), submit));
});
