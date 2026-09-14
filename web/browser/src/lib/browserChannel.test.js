import assert from "node:assert/strict";
import { test } from "node:test";
import { createBrowserChannel, EVENTS_VERB } from "./browserChannel.js";

// A stand-in for the EventSource: the test decides when a frame arrives.
function fakeSource() {
  const listeners = new Map();
  return {
    addEventListener(name, fn) {
      listeners.set(name, fn);
    },
    removeEventListener(name) {
      listeners.delete(name);
    },
    close() {
      listeners.clear();
    },
    // frame delivers one SSE command frame to the channel.
    frame(data) {
      listeners.get("command")?.({ data });
    },
    get listening() {
      return listeners.has("command");
    },
  };
}

function harness({ tab = "w:7", invoke, post } = {}) {
  const source = fakeSource();
  const calls = { invoke: [], post: [], errors: [] };
  const channel = createBrowserChannel({
    activeTabId: () => tab,
    invoke:
      invoke ??
      (async (cmd, args) => {
        calls.invoke.push([cmd, args]);
        return { ok: cmd };
      }),
    source,
    post:
      post ??
      (async (result) => {
        calls.post.push(result);
      }),
    onError: (message, err) => calls.errors.push(String(err?.message || err || message)),
  });
  return { source, calls, channel };
}

// settle lets the async command path finish.
const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

test("a CDP command reaches the bridge with its tier and params", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c1", method: "Page.captureScreenshot", params: { format: "png" }, tier: "act" }));
  await settle();
  assert.deepEqual(calls.invoke, [
    ["btab_cdp_call", { id: "7", method: "Page.captureScreenshot", paramsJson: '{"format":"png"}', tier: "act", domains: [] }],
  ]);
  assert.deepEqual(calls.post, [{ id: "c1", output: { ok: "btab_cdp_call" } }]);
});

test("a command without a tier is treated as read", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c2", method: "DOM.getDocument" }));
  await settle();
  assert.equal(calls.invoke[0][1].tier, "read");
  assert.equal(calls.invoke[0][1].paramsJson, "{}");
});

test("the events verb goes to the shell ring, not to the page", async () => {
  const { source, calls } = harness();
  source.frame(JSON.stringify({ id: "c3", method: EVENTS_VERB, params: { since: 12 } }));
  await settle();
  assert.deepEqual(calls.invoke, [["btab_cdp_events", { id: "7", since: 12 }]]);
  assert.deepEqual(calls.post, [{ id: "c3", output: { ok: "btab_cdp_events" } }]);
});

test("no work-browser tab on screen is an answer, not a hang", async () => {
  const { source, calls } = harness({ tab: "t:5" });
  source.frame(JSON.stringify({ id: "c4", method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.equal(calls.post.length, 1);
  assert.equal(calls.post[0].id, "c4");
  assert.match(calls.post[0].error, /no work-browser tab/);
});

test("a bridge refusal travels back as the command's error", async () => {
  const { source, calls } = harness({
    invoke: async () => {
      throw new Error("Runtime.evaluate needs the act tier; this agent has read");
    },
  });
  source.frame(JSON.stringify({ id: "c5", method: "Runtime.evaluate", tier: "read" }));
  await settle();
  assert.equal(calls.post[0].id, "c5");
  assert.match(calls.post[0].error, /needs the act tier/);
});

test("frames that are not commands are ignored", async () => {
  const { source, calls } = harness();
  source.frame("not json");
  source.frame(JSON.stringify({ method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.invoke, []);
  assert.deepEqual(calls.post, []);
});

test("a failed result post is reported and does not throw", async () => {
  const { source, calls } = harness({
    post: async () => {
      throw new Error("result 500");
    },
  });
  source.frame(JSON.stringify({ id: "c6", method: "DOM.getDocument" }));
  await settle();
  assert.deepEqual(calls.post, []);
  assert.equal(calls.errors.length, 1);
  assert.match(calls.errors[0], /result 500/);
});

test("close stops the stream from answering", async () => {
  const { source, calls, channel } = harness();
  channel.close();
  assert.equal(source.listening, false);
});
